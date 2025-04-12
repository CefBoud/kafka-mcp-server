package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func DescribeClusterTool(cfg *Config) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("describeCluster",
			mcp.WithDescription("Describe the Kafka cluster. Returns brokers and controllerID."),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			config := sarama.NewConfig()
			admin, err := sarama.NewClusterAdmin(cfg.BootstrapServers, config)
			if err != nil {
				err = fmt.Errorf("Error init kafka admin client: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			brokers, controllerID, err := admin.DescribeCluster()
			if err != nil {
				err = fmt.Errorf("Error describing the cluster: %v", err)
				return mcp.NewToolResultError(err.Error()), err
			}
			var returnedBrokers []Broker
			for _, b := range brokers {
				returnedBrokers = append(returnedBrokers, Broker{
					ID:   b.ID(),
					Addr: b.Addr(),
					Rack: b.Rack(),
				})
			}

			result, _ := json.Marshal(struct {
				Brokers      []Broker
				ControllerID int32
			}{
				Brokers:      returnedBrokers,
				ControllerID: controllerID,
			})

			return mcp.NewToolResultText(string(result)), nil
		}
}
