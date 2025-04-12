package kafka

import "github.com/mark3labs/mcp-go/server"

type Config struct {
	BootstrapServers []string
}

// NewServer creates a new Kafka MCP server with the specified GH client and logger.
func NewServer(version string, readOnly bool, multiplex bool, multiplexModel string, cfg *Config, opts ...server.ServerOption) *server.MCPServer {
	// Add default options
	defaultOpts := []server.ServerOption{
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	}
	opts = append(defaultOpts, opts...)

	// Create a new MCP server
	s := server.NewMCPServer(
		"kafka-mcp-server",
		version,
		opts...,
	)

	s.AddTool(ConsumeMessagesTool(cfg))
	s.AddTool(ListTopicsTool(cfg))
	s.AddTool(TopicOffsetsTool(cfg))
	s.AddTool(DescribeClusterTool(cfg))
	s.AddTool(ListConsumerGroupsTool(cfg))
	s.AddTool(DescribeConsumerGroupsTool(cfg))
	if !readOnly {
		s.AddTool(ProducerMessagesTool(cfg))
		s.AddTool(CreateTopicTool(cfg))
	}

	// Multiplexer
	if multiplex {
		if err := ValidateLLMConfig(multiplexModel); err != nil {
			panic(err)
		}
		s.AddTool(MultiplexToolsTool(cfg, multiplexModel, s))
	}

	return s
}
