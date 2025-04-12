package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/IBM/sarama"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	CONSUMER_TIMEOUT = 15 * time.Second
)

type ConsumerMessage struct {
	Key, Value string
	Partition  int32
	Offset     int64
	Timestamp  string
}

type ConsumerHandler struct {
	msgCount int
	messages []ConsumerMessage
	cancel   context.CancelFunc
}

func (c *ConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (c *ConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (c *ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	consumed := 0
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				fmt.Fprintf(os.Stderr, "message channel was closed")
				return nil
			}
			fmt.Fprintf(os.Stderr, "Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
			session.MarkMessage(message, "")
			c.messages = append(c.messages, ConsumerMessage{
				Key:       string(message.Key),
				Value:     string(message.Value),
				Partition: message.Partition,
				Offset:    message.Offset,
				Timestamp: message.Timestamp.String(),
			})
			consumed++
			if c.msgCount == consumed {
				c.cancel()
				continue
			}

		case <-session.Context().Done():
			fmt.Fprintf(os.Stderr, "session.Context().Done()...")
			return nil
		}
	}
}

func ConsumeMessagesTool(cfg *Config) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("consumerMessages",
			mcp.WithDescription("Consumes numMessages from a topic starting from the beginning (offset -2), from the end i.e. latest (offset -1) or from a specific offset (>= 0). The partition must be specified when consuming from a specific offset. If there are not enough messages, we timeout after 5 seconds."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the topic to consume messages from."),
			),
			mcp.WithNumber("numMessages",
				mcp.Required(),
				mcp.Description("Number of messages to consume."),
			),
			mcp.WithNumber("offset",
				mcp.Required(),
				mcp.Description("Offset to start consuming from. -2 means starting from the beginning, -1 means starting from the end (latest). Any value ≥ 0 indicates a specific position in a partition, which must be specified in that case."),
			),
			mcp.WithNumber("partitionIndex",
				mcp.Description("The index of the topic's partition to consume from. This is required if the offset is ≥ 0."),
			),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			topic := request.Params.Arguments["name"].(string)
			numMessages := request.Params.Arguments["numMessages"].(float64)
			offset := request.Params.Arguments["offset"].(float64)
			// partitionIndex, ok := request.Params.Arguments["partitionIndex"].(float64)
			log.Printf("topic: %v, numMessages %v, offset: %v", topic, numMessages, offset)
			config := sarama.NewConfig()

			if offset == -1 {
				config.Consumer.Offsets.Initial = sarama.OffsetNewest
			} else if offset == -2 {
				config.Consumer.Offsets.Initial = sarama.OffsetOldest
			} else {
				err := fmt.Errorf("offset should be -1 or -2")
				return mcp.NewToolResultError(err.Error()), err
			}
			group := fmt.Sprintf("kafka-mcp-server-group-%v", time.Now().UnixMilli())
			consumer, err := sarama.NewConsumerGroup(cfg.BootstrapServers, group, config)
			if err != nil {
				err := fmt.Errorf("error creating consumer group: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			defer consumer.Close()

			ctx, cancel := context.WithTimeout(context.Background(), CONSUMER_TIMEOUT)
			defer cancel()
			handler := &ConsumerHandler{msgCount: int(numMessages), cancel: cancel}
			log.Println("starting consumer")
			err = consumer.Consume(ctx, []string{topic}, handler)
			if err != nil {
				err := fmt.Errorf("Error from consumer: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			result, _ := json.Marshal(handler.messages)
			return mcp.NewToolResultText(string(result)), nil
		}
}
