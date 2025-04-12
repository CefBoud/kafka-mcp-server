package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/IBM/sarama"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func ListTopicsTool(cfg *Config) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("listTopics",
			mcp.WithDescription("List topics present in the cluster"),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			config := sarama.NewConfig()
			admin, err := sarama.NewClusterAdmin(cfg.BootstrapServers, config)
			if err != nil {
				err = fmt.Errorf("Error init kafka admin client: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			topics, err := admin.ListTopics()
			if err != nil {
				err = fmt.Errorf("Error listing topics: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}

			result, err := json.Marshal(topics)
			if err != nil {
				err = fmt.Errorf("Error listing topics: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}

			return mcp.NewToolResultText(string(result)), nil
		}
}

func CreateTopicTool(cfg *Config) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("createTopic",
			mcp.WithDescription("Create a topic in the cluster kafka"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the topic"),
			),
			mcp.WithNumber("replicationFactor",
				mcp.Required(),
				mcp.Description("Replication factor"),
			),
			mcp.WithNumber("numPartitions",
				mcp.Required(),
				mcp.Description("Number of partitions"),
			),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			Name := request.Params.Arguments["name"].(string)
			replicationFactor := request.Params.Arguments["replicationFactor"].(float64)
			numPartitions := request.Params.Arguments["numPartitions"].(float64)

			admin, err := sarama.NewClusterAdmin(cfg.BootstrapServers, sarama.NewConfig())
			if err != nil {
				err = fmt.Errorf("Error init kafka admin client: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			err = admin.CreateTopic(Name, &sarama.TopicDetail{NumPartitions: int32(numPartitions), ReplicationFactor: int16(replicationFactor)}, false)
			if err != nil {
				err = fmt.Errorf("Error creating topic: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}

			return mcp.NewToolResultText("Topic created."), nil
		}
}

func TopicOffsetsTool(cfg *Config) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("topicOffsets",
			mcp.WithDescription("Fetches start and end offsets for all partitions of a Kafka topic."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the Kafka topic."),
			),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			topic := request.Params.Arguments["name"].(string)

			config := sarama.NewConfig()

			client, err := sarama.NewClient(cfg.BootstrapServers, config)
			if err != nil {
				err = fmt.Errorf("Failed to create Kafka client: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			defer client.Close()

			partitions, err := client.Partitions(topic)
			if err != nil {
				err = fmt.Errorf("Failed to fetch partitions: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}

			offsets := []PartitionOffset{}

			for _, partition := range partitions {
				startOffset, err := client.GetOffset(topic, partition, sarama.OffsetOldest)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting start offset for partition %d: %v", partition, err)
					continue
				}

				endOffset, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting end offset for partition %d: %v", partition, err)
					continue
				}

				offsets = append(offsets, PartitionOffset{
					Partition:   partition,
					StartOffset: startOffset,
					EndOffset:   endOffset,
				})
			}

			offsetsJson, _ := json.Marshal(offsets)
			return mcp.NewToolResultText(string(offsetsJson)), nil
		}
}
