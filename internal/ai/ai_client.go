package ai_client

import (
	"context"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiClient はGemini APIとの連携を担当します。
type GeminiClient struct {
	client *genai.GenerativeModel
	ctx    context.Context
}

// NewGeminiClient は新しいGeminiClientのインスタンスを作成します。
func NewGeminiClient(apiKey string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is missing")
	}
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	model := client.GenerativeModel("gemini-1.5-flash")
	return &GeminiClient{client: model, ctx: ctx}, nil
}

// Generate は指定されたプロンプトに基づいてAIコンテンツを生成します。
func (gc *GeminiClient) Generate(prompt string) (string, error) {
	if gc.client == nil {
		var emptyString string = ""
		return emptyString, fmt.Errorf("Gemini client is not initialized")
	}
	log.Printf("Sending prompt to Gemini: \n%s\n", prompt)
	resp, err := gc.client.GenerateContent(gc.ctx, genai.Text(prompt))
	if err != nil {
		var emptyString string = ""
		return emptyString, fmt.Errorf("failed to generate content: %w", err)
	}

	var answer string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if txt, ok := part.(genai.Text); ok {
					answer += string(txt)
				}
			}
		}
	}
	if answer == "" {
		log.Println("Gemini API returned an empty answer.")
		return "", nil
	}
	return answer, nil
}
