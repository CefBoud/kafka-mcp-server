package main

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/CefBoud/kafka-mcp-server/pkg/kafka"
	iolog "github.com/CefBoud/kafka-mcp-server/pkg/log"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var version = "version"
var commit = "commit"
var date = "date"

var (
	rootCmd = &cobra.Command{
		Use:     "server",
		Short:   "Kafka MCP Server",
		Long:    `A Kafka MCP server that handles various tools and resources.`,
		Version: fmt.Sprintf("%s (%s) %s", version, commit, date),
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Start stdio server",
		Long:  `Start a server that communicates via standard input/output streams using JSON-RPC messages.`,
		Run: func(_ *cobra.Command, _ []string) {
			logFile := viper.GetString("log-file")
			readOnly := viper.GetBool("read-only")
			multiplex := viper.GetBool("enable-multiplex")
			multiplexModel := viper.GetString("multiplex-model")

			logger, err := initLogger(logFile)
			if err != nil {
				stdlog.Fatal("Failed to initialize logger:", err)
			}
			logCommands := viper.GetBool("enable-command-logging")

			// either via command line of KAFKA_MCP_BOOTSTRAP_SERVERS env var
			bootstrapServers := viper.GetString("bootstrap-servers")
			if bootstrapServers == "" {
				stdlog.Fatal("bootstrap-servers or KAFKA_MCP_BOOTSTRAP_SERVERS env not set")
			}

			cfg := Config{
				readOnly:    readOnly,
				logger:      logger,
				logCommands: logCommands,
				KafkaConfig: &kafka.Config{
					BootstrapServers: strings.Split(bootstrapServers, ","),
				},
				Multiplex:      multiplex,
				MultiplexModel: multiplexModel,
			}
			if err := runStdioServer(cfg); err != nil {
				stdlog.Fatal("failed to run stdio server:", err)
			}
		},
	}
)

func initLogger(outPath string) (*log.Logger, error) {
	if outPath == "" {
		return log.New(), nil
	}

	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New()
	logger.SetLevel(log.DebugLevel)
	logger.SetOutput(file)

	return logger, nil
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add global flags that will be shared by all commands
	rootCmd.PersistentFlags().Bool("read-only", false, "Restrict the server to read-only operations")
	rootCmd.PersistentFlags().String("log-file", "", "Path to log file")
	rootCmd.PersistentFlags().Bool("enable-command-logging", false, "When enabled, the server will log all command requests and responses to the log file")
	rootCmd.PersistentFlags().Bool("enable-multiplex", false, "Enable multiplexing/batching multiple tool calls together.")
	rootCmd.PersistentFlags().String("multiplex-model", "", "When multiplexing is enabled, this model is used to infer PROMPT_ARGUMENTs which are dynamic tool arguments derived from previous tool results and a prompt supplied by the MCP client. (Only gemini is supported for now. 'GEMINI_API_KEY' env var is expected.)")

	// Bind flag to viper
	_ = viper.BindPFlag("read-only", rootCmd.PersistentFlags().Lookup("read-only"))
	_ = viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("enable-command-logging", rootCmd.PersistentFlags().Lookup("enable-command-logging"))
	_ = viper.BindPFlag("bootstrap-servers", rootCmd.PersistentFlags().Lookup("bootstrap-servers"))
	_ = viper.BindPFlag("enable-multiplex", rootCmd.PersistentFlags().Lookup("enable-multiplex"))
	_ = viper.BindPFlag("multiplex-model", rootCmd.PersistentFlags().Lookup("multiplex-model"))

	// Add subcommands
	rootCmd.AddCommand(stdioCmd)
}

func initConfig() {
	// Initialize Viper configuration
	viper.SetEnvPrefix("KAFKA_MCP")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
}

type Config struct {
	readOnly       bool
	logger         *log.Logger
	logCommands    bool
	KafkaConfig    *kafka.Config
	Multiplex      bool
	MultiplexModel string
}

func runStdioServer(cfg Config) error {
	// Create app context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hooks := &server.Hooks{}
	if cfg.Multiplex {
		hooks.OnBeforeCallTool = []server.OnBeforeCallToolFunc{kafka.BeforeToolCallPromptArgumentHook}
		hooks.OnAfterCallTool = []server.OnAfterCallToolFunc{kafka.AfterToolCallPromptArgumentHook}
		ctx = context.WithValue(ctx, "MultiplexModel", cfg.MultiplexModel)
	}
	// Create
	kafkaServer := kafka.NewServer(version, cfg.readOnly, cfg.Multiplex, cfg.MultiplexModel, cfg.KafkaConfig, server.WithHooks(hooks))
	stdioServer := server.NewStdioServer(kafkaServer)

	stdLogger := stdlog.New(cfg.logger.Writer(), "stdioserver", 0)
	stdioServer.SetErrorLogger(stdLogger)

	// Start listening for messages
	errC := make(chan error, 1)
	go func() {
		in, out := io.Reader(os.Stdin), io.Writer(os.Stdout)

		if cfg.logCommands {
			loggedIO := iolog.NewIOLogger(in, out, cfg.logger)
			in, out = loggedIO, loggedIO
		}

		errC <- stdioServer.Listen(ctx, in, out)
	}()

	_, _ = fmt.Fprintf(os.Stderr, "Kafka MCP Server running on stdio\n")

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		cfg.logger.Infof("shutting down server...")
	case err := <-errC:
		if err != nil {
			return fmt.Errorf("error running server: %w", err)
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
