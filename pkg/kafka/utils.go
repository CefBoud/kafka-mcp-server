package kafka

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// topicPartition takes in a topic and partition and returns `{topic}-{partition}`
func topicPartition(topic string, partition int32) string {
	return fmt.Sprintf("%v-%v", topic, partition)
}

func ValidateLLMConfig(model string) error {
	if ModelType(model) == GeminiModel {
		_, ok := os.LookupEnv("GEMINI_API_KEY")
		if !ok {
			return fmt.Errorf("GEMINI_API_KEY not defined")
		}
		return nil
	}
	return fmt.Errorf("Only 'gemini' is supported for now.")

}

// queryLLM takes in a prompt and return the model's response.
// Only Gemini is supported for now
func QueryLLM(prompt string, modelLLM ModelType) (string, error) {

	if modelLLM == GeminiModel {
		return queryGemini(prompt)
	}
	return "", nil
}

func queryGemini(prompt string) (string, error) {
	GEMINI_API_KEY := os.Getenv("GEMINI_API_KEY")

	// fmt.Fprintf(os.Stderr, "queryGemini prompt %v\n", prompt)
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(GEMINI_API_KEY))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.0-flash")
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))

	if err != nil {
		log.Fatal(err)
	}
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			result := ""
			for _, part := range cand.Content.Parts {
				result += fmt.Sprintf("%v", part)
			}
			// fmt.Fprintf(os.Stderr, "queryGemini result : %v", result)
			return strings.TrimSpace(result), nil
		}
	}
	return "", nil
}
