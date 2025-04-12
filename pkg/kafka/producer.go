package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func ProducerMessagesTool(cfg *Config) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("producerMessages",
			mcp.WithDescription("Produces messages to a Kafka topic."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the topic to produce messages to."),
			),
			mcp.WithArray("messages",
				mcp.Required(),
				mcp.Description("List of messages to produce."),
			),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			topic := request.Params.Arguments["name"].(string)
			messages := request.Params.Arguments["messages"].([]any)

			config := sarama.NewConfig()
			config.Producer.Return.Successes = true
			producer, err := sarama.NewSyncProducer(cfg.BootstrapServers, config)
			if err != nil {
				err = fmt.Errorf("Failed to start Sarama producer: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			defer func() {
				if err := producer.Close(); err != nil {
					log.Println("Failed to close Kafka producer cleanly:", err)
				}
			}()

			partitionOffsets := []MessagePartitionOffset{}
			for _, m := range messages {
				msg := &sarama.ProducerMessage{
					Topic: topic,
					Value: sarama.StringEncoder(m.(string)),
				}

				partition, offset, err := producer.SendMessage(msg)
				partitionOffsets = append(partitionOffsets, MessagePartitionOffset{Partition: int(partition), Offset: int(offset)})
				if err != nil {
					log.Printf("Failed to send message: %v", err)
				} else {
					log.Printf("Message sent to partition %d at offset %d\n", partition, offset)
				}
			}
			partitionOffsetsJson, _ := json.Marshal(partitionOffsets)
			result := fmt.Sprintf("MessagePartitionOffset tuples of produced messages: %v", string(partitionOffsetsJson))
			return mcp.NewToolResultText(result), nil
		}
}
