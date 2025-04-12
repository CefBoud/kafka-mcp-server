package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func ListConsumerGroupsTool(cfg *Config) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("listConsumerGroups",
			mcp.WithDescription("List consumer groups present in the cluster"),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			config := sarama.NewConfig()
			admin, err := sarama.NewClusterAdmin(cfg.BootstrapServers, config)
			if err != nil {
				err = fmt.Errorf("Error init kafka admin client: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			groups, err := admin.ListConsumerGroups()
			if err != nil {
				err = fmt.Errorf("Error listing consumer groups: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}

			result, err := json.Marshal(groups)
			if err != nil {
				err = fmt.Errorf("Error marshaling consumer groups: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}

			return mcp.NewToolResultText(string(result)), nil
		}
}

func DescribeConsumerGroupsTool(cfg *Config) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("describeConsumerGroups",
			mcp.WithDescription("List Kafka consumer groups with topic/partition offsets and lag."),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			config := sarama.NewConfig()

			client, err := sarama.NewClient(cfg.BootstrapServers, config)
			if err != nil {
				err = fmt.Errorf("Error creating Kafka client: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			defer client.Close()

			admin, err := sarama.NewClusterAdmin(cfg.BootstrapServers, config)
			if err != nil {
				err = fmt.Errorf("Error creating Kafka admin: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			defer admin.Close()

			groups, err := admin.ListConsumerGroups()
			if err != nil {
				err = fmt.Errorf("Error listing consumer groups: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}

			var resultData []GroupInfo
			logEndOffsets := make(map[string]int64)

			for groupID := range groups {
				desc, err := admin.DescribeConsumerGroups([]string{groupID})
				if err != nil {
					continue
				}

				for _, group := range desc {
					if group.Err != sarama.ErrNoError {
						continue
					}

					offsets, err := admin.ListConsumerGroupOffsets(group.GroupId, nil)
					if err != nil {
						continue
					}

					var partitionOffsets []GroupPartitionInfo

					for topic, partitions := range offsets.Blocks {
						for partition, block := range partitions {
							tp := fmt.Sprintf("%s-%d", topic, partition)
							if _, ok := logEndOffsets[tp]; !ok {
								logEndOffsets[tp], err = client.GetOffset(topic, partition, sarama.OffsetNewest)
								if err != nil {
									continue
								}
							}
							partitionOffsets = append(partitionOffsets, GroupPartitionInfo{
								Topic:         topic,
								Partition:     partition,
								CurrentOffset: block.Offset,
								LogEndOffset:  logEndOffsets[tp],
								Lag:           logEndOffsets[tp] - block.Offset,
							})
						}
					}

					resultData = append(resultData, GroupInfo{
						GroupID: group.GroupId,
						Offsets: partitionOffsets,
					})
				}
			}

			resultJSON, _ := json.Marshal(resultData)
			return mcp.NewToolResultText(string(resultJSON)), nil
		}
}
