package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

// LLMService provides operations for interacting with an LLM.
type LLMService struct {
	client *googleai.GoogleAI
}

// NewLLMService creates a new LLM service.
func NewLLMService() (*LLMService, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	client, err := googleai.New(context.Background(), googleai.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create googleai client: %w", err)
	}

	return &LLMService{client: client}, nil
}

// NarrowDownURLs uses the LLM to narrow down a list of URLs to 15.
func (s *LLMService) NarrowDownURLs(urls []string, siteCategory string) ([]string, error) {
	prompt := fmt.Sprintf(
		"From the following list of URLs, select the 20 most relevant URLs for a site also exploring different categories '%s'.\n\nURLs:\n%v\n\nReturn a json list of selected URLs.",
		siteCategory,
		strings.Join(urls, "\n"),
	)

	resp, err := s.client.GenerateContent(context.Background(),
		[]llms.MessageContent{
			{
				Role: llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{
					llms.TextContent{Text: prompt},
				},
			},
		},
		llms.WithMaxTokens(12048), llms.WithModel("gemini-2.5-flash"),
		llms.WithJSONMode(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	return parseJSONURLs(resp.Choices[0].Content)
}

// SelectAndCategorizeURLs uses the LLM to select 10 URLs and assign categories.
func (s *LLMService) SelectAndCategorizeURLs(urls []string, heads map[string]string, siteCategory string) ([]Result, error) {
	prompt := fmt.Sprintf(
		"From the following list of URLs and their HTML head sections, select the 10 most relevant URLs for a site with the category '%s'. For each selected URL, assign a relevant category.\n\n",
		siteCategory,
	)

	for _, url := range urls {
		prompt += fmt.Sprintf("URL: %s\nHead:\n%s\n\n", url, heads[url])
	}

	prompt += "Return the result as a JSON array of objects, where each object has 'url' and 'category' keys. For example: [{\"url\": \"https://example.com\", \"category\": \"e-commerce\"}]"

	resp, err := s.client.GenerateContent(context.Background(),
		[]llms.MessageContent{
			{
				Role: llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{
					llms.TextContent{Text: prompt},
				},
			},
		},
		llms.WithMaxTokens(14096), llms.WithModel("gemini-2.5-flash"),
		llms.WithJSONMode(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	return parseJSONResponse(resp.Choices[0].Content)
}

func parseJSONURLs(in string) ([]string, error) {
	// Parse JSON response containing a list of URL strings
	// The LLM can return a markdown code block, so we need to trim it.
	in = strings.TrimPrefix(in, "```json")
	in = strings.TrimSuffix(in, "```")
	in = strings.TrimSpace(in)

	var urls []string
	err := json.Unmarshal([]byte(in), &urls)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return urls, nil
}

func parseCommaSeparated(in string) []string {
	// Basic parsing, assuming a simple comma-separated string.
	// In a real-world scenario, this would need to be more robust.
	var result []string
	for _, url := range splitAndTrim(in, ",") {
		result = append(result, url)
	}
	return result
}

func parseJSONResponse(in string) ([]Result, error) {
	// Basic parsing, assuming a simple JSON array.
	// In a real-world scenario, this would need to be more robust.
	// The LLM can return a markdown code block, so we need to trim it.
	in = strings.TrimPrefix(in, "```json")
	in = strings.TrimSuffix(in, "```")
	var results []Result
	err := json.Unmarshal([]byte(in), &results)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return results, nil
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, item := range strings.Split(s, sep) {
		result = append(result, strings.TrimSpace(item))
	}
	return result
}
