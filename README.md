## Kafka MCP Server
 The Kafka MCP Server is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction)
server that provides integration with Apache Kafka to enable interaction with a Kafka cluster using natural language (LLMs).

[demo](https://github.com/user-attachments/assets/67d3207a-bfe1-4549-a5c5-b2058bd65717)


## Prerequisites 
You will either need Docker or Golang to run the MCP server locally.   

## Getting Started

You need to have access to a Kafka cluster. Following the [quickstart](https://kafka.apache.org/quickstart) doc:

```
 docker pull apache/kafka:4.0.0
 docker run -p 9092:9092 apache/kafka:4.0.0
```
Kafka is available on `localhost:9092` now.

### Usage with Claude Desktop

#### Docker:

```json
{
  "mcpServers": {
    "kafka": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e",
        "KAFKA_MCP_BOOTSTRAP_SERVERS",
        "ghcr.io/cefboud/kafka-mcp-server"
      ],
      "env": {
        "KAFKA_MCP_BOOTSTRAP_SERVERS": "localhost:9092"
      }
    }
  }
}
```
#### Building locally

```
cd <workdir>
git clone https://github.com/CefBoud/kafka-mcp-server.git
cd kafka-mcp-server
go build -o kafka-mcp-server  cmd/kafka-mcp-server/main.go 
```

```json
{
  "mcpServers": {
    "kafka": {
      "command": "<workdir>/kafka-mcp-server/kafka-mcp-server",
      "args": [
        "stdio",
      ],
      "env": {
        "KAFKA_MCP_BOOTSTRAP_SERVERS": "localhost:9092"
      }
    }
  }
}
```

Options:

```
      --bootstrap-servers string   Comma-separated list of the Kafka servers to connect to.
      --enable-command-logging     When enabled, the server will log all command requests and responses to the log file
      --log-file string            Path to log file
      --read-only                  Restrict the server to read-only operations
```

All options can be passed as environment variables, uppercased, with hyphens replaced by underscores, and prefixed with `MCP_KAFKA_` e.g., `--bootstrap-servers` becomes `MCP_KAFKA_BOOTSTRAP_SERVERS`.

## Available MCP Tools

- [x] List topics
- [x] Create topic
- [x] Consuming messages.
- [x] Produce messages.
- [x] Describe the clusters (list of brokers and controller)
- [x] List consumer groups and their lag.
- [x] Get topic's earliest and latest offsets (GetOffsetShell)
- [ ] Reset consumer group offsets.
- [ ] Kafka Connect ??
- [ ] Schema Registry ??

##
## 🔀 MultiplexTool

[demo MultiplexTool ](https://github.com/user-attachments/assets/ad264bab-f135-462d-9fed-e8aedf7ffcc7)


Running many sequential tools, especially when each depends on the output of the previous, can be tedious and time-consuming. This requires multiple round-trips between the client and server/

**MultiplexTool** solves this by allowing the client to batch a list of tool calls into a single request, executing them in order. It supports dynamic dependencies between tools by letting you reference earlier outputs using prompt-based placeholders.

If a tool input depends on a previous result, the client uses the `PROMPT_ARGUMENT:` format to generate that input dynamically via a prompt to an LLM (currently only Gemini is supported).  
Example:  
`"userId": "PROMPT_ARGUMENT: the ID of the created user"`

**CLI Flags:**

- `--enable-multiplex`: Enables multiplexing of tool calls.
- `--multiplex-model`: Specifies the model (e.g. `gemini`) used to infer `PROMPT_ARGUMENT`s. Requires `GEMINI_API_KEY` env variable.

```json
{
  "mcpServers": {
    "kafka": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e",
        "KAFKA_MCP_BOOTSTRAP_SERVERS",
        "ghcr.io/cefboud/kafka-mcp-server",
        "--enable-multiplex",
        "--multiplex-model",
        "gemini"
      ],
      "env": {
        "KAFKA_MCP_BOOTSTRAP_SERVERS": "localhost:9092",
        "GEMINI_API_KEY": "....."
      }
    }
  }
}
```


# Credits
* github.com/github/github-mcp-server
* github.com/mark3labs/mcp-go 
* github.com/IBM/sarama 