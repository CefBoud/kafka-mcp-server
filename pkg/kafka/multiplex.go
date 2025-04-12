package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ModelType string

const (
	GeminiModel ModelType = "gemini"
)

const ToolContextKey = "tool"

var ToolContext = ""

func getModel(ctx context.Context) ModelType {
	return ModelType(ctx.Value("MultiplexModel").(string))
}

func InferArgumentPrompt(previousContext string, argument string) string {
	return fmt.Sprintf(`We are running a sequence of tools. We are trying to infer an argument for the next tool based on the previous requests and responses and a prompt describing the argument.

Previous requests and responses: %v


Argument Prompt: %v


Return the argument only with nothing else. This will be given as input to a function call.
	`, previousContext, argument)
}

func addStringToToolCallContext(s string) string {
	ToolContext += s + "\n"
	return ToolContext
}

func getToolCallContext() string {
	return ToolContext
}
func clearToolContext() {
	ToolContext = ""
}

// BeforeToolCallPromptArgumentHook is called before tool calls. It queries the configured LLM to infer PROMPT_ARGUMENT from the previous context. It also appends the messages to the context.
func BeforeToolCallPromptArgumentHook(ctx context.Context, id any, message *mcp.CallToolRequest) {
	messageJson, _ := json.Marshal(message)
	toolContext := addStringToToolCallContext("Tool call request:\n" + string(messageJson))

	for k, v := range message.Params.Arguments {
		if arg, ok := v.(string); ok {
			strings.HasPrefix(arg, "PROMPT_ARGUMENT")
			prompt := InferArgumentPrompt(toolContext, arg)
			inferredArg, err := QueryLLM(prompt, getModel(ctx))
			if err != nil {
				return
			}
			fmt.Fprintf(os.Stderr, "inferredArg %v", inferredArg)
			message.Params.Arguments[k] = inferredArg
		}

	}
}

// AfterToolCallPromptArgumentHook adds the tool result to the context.
func AfterToolCallPromptArgumentHook(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
	fmt.Fprintf(os.Stderr, "inside AfterToolCallPromptArgumentHook %v", message)
	resultJson, _ := json.Marshal(result)
	addStringToToolCallContext("Tool call result:\n" + string(resultJson))
}

func MultiplexToolsTool(cfg *Config, multiplexModel string, s *server.MCPServer) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("MultiplexTools",
			mcp.WithDescription("Takes a list of tool requests and executes each one, returning a list of their results. Use this tool when you need to call multiple tools in sequence. If an argument for a tool at position N depends on the result of a previous tool [1...N-1], you can express that argument as a prompt to the LLM using the format `PROMPT_ARGUMENT: your prompt here`. For example: `PROMPT_ARGUMENT: the ID of the created resource.`"),
			mcp.WithArray("tools",
				mcp.Description("List of tool requests"),
				mcp.Required(),
				mcp.Items(map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"jsonrpc": map[string]interface{}{
							"type":        "string",
							"description": "jsonrpc version.",
							"required":    true,
						},
						"id": map[string]interface{}{
							"type":        "number",
							"description": "the request ID.",
							"required":    true,
						},
						"method": map[string]interface{}{
							"type":        "string",
							"description": "The MCP Method. 'tools/call' in this case.",
							"required":    true,
						},
						"params": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "The tool's name.",
									"required":    true,
								},
								"arguments": map[string]interface{}{
									"type":        "object",
									"description": "The tool's arguments derived from each specific tool input schema.",
									"required":    true,
								},
							},
						},
					},
				},
				)),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			clearToolContext()
			tools := request.Params.Arguments["tools"].([]interface{})
			var result []any
			for _, tool := range tools {
				payload, _ := json.Marshal(tool)
				response := s.HandleMessage(ctx, payload)
				result = append(result, response)
			}

			jsonResult, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(jsonResult)), nil
		}
}
