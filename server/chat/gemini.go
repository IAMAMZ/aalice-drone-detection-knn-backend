package chat

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

type GeminiClient struct {
	client *genai.Client
	ctx    context.Context
}

func NewGeminiClient() (*GeminiClient, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %v", err)
	}

	return &GeminiClient{
		client: client,
		ctx:    ctx,
	}, nil
}

func (g *GeminiClient) GenerateResponse(message string) (string, error) {
	systemPrompt := `You are AALIS Assistant, an AI assistant for the Acoustic Autonomous Lightweight Interception System. 
You help users with:
- Drone detection and acoustic analysis
- Audio processing and signal analysis
- System operations and troubleshooting
- General questions about acoustic surveillance

Provide helpful, accurate, and concise responses. Be technical when needed but explain complex concepts clearly.
Keep responses conversational and under 200 words unless more detail is specifically requested.`

	// Create content with system instruction and user message
	systemInstruction := genai.NewContentFromText(systemPrompt, genai.RoleModel)
	userContent := genai.NewContentFromText(message, genai.RoleUser)

	// Configure generation parameters
	config := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Temperature:       genai.Ptr(float32(0.7)),
		TopP:              genai.Ptr(float32(0.8)),
		TopK:              genai.Ptr(float32(40)),
		MaxOutputTokens:   int32(200),
	}

	// Generate response
	resp, err := g.client.Models.GenerateContent(
		g.ctx,
		"gemini-2.5-flash",
		[]*genai.Content{userContent},
		config,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	// Extract text from response
	text := resp.Text()
	if text == "" {
		return "I'm sorry, I couldn't generate a response. Please try rephrasing your question.", nil
	}

	cleanText := strings.ReplaceAll(text, "*", "")

	return cleanText, nil
}

// GenerateResponseStream generates a streaming response
func (g *GeminiClient) GenerateResponseStream(message string, onChunk func(string) error) error {
	systemPrompt := `You are AALIS Assistant, an AI assistant for the Acoustic Autonomous Lightweight Interception System. 
You help users with:
- Drone detection and acoustic analysis
- Audio processing and signal analysis
- System operations and troubleshooting
- General questions about acoustic surveillance

Provide helpful, accurate, and concise responses. Be technical when needed but explain complex concepts clearly.
Keep responses conversational and under 200 words unless more detail is specifically requested.`

	// Create content with system instruction and user message
	systemInstruction := genai.NewContentFromText(systemPrompt, genai.RoleModel)
	userContent := genai.NewContentFromText(message, genai.RoleUser)

	// Configure generation parameters
	config := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Temperature:       genai.Ptr(float32(0.7)),
		TopP:              genai.Ptr(float32(0.8)),
		TopK:              genai.Ptr(float32(40)),
		MaxOutputTokens:   int32(200),
	}

	// Generate streaming response
	stream := g.client.Models.GenerateContentStream(
		g.ctx,
		"gemini-2.5-flash",
		[]*genai.Content{userContent},
		config,
	)

	// Process stream chunks
	for resp, err := range stream {
		if err != nil {
			return fmt.Errorf("stream error: %v", err)
		}

		text := resp.Text()
		cleanText := strings.ReplaceAll(text, "*", "")
		if text != "" {
			if err := onChunk(cleanText); err != nil {
				return fmt.Errorf("chunk callback error: %v", err)
			}
		}
	}

	return nil
}

func (g *GeminiClient) Close() error {
	// The new client doesn't have an explicit Close method
	// Resources are managed automatically
	return nil
}
