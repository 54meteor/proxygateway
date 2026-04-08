package adapter

import (
	"context"
	"fmt"

	"ai-gateway/internal/config"
	"ai-gateway/internal/model"

	"github.com/sashabaranov/go-openai"
)

// OpenAIAdapter OpenAI 适配器
type OpenAIAdapter struct {
	client *openai.Client
	cfg    *config.Config
}

func NewOpenAIAdapter(cfg *config.Config) *OpenAIAdapter {
	client := openai.NewClient(cfg.Models.OpenAI.APIKey)
	return &OpenAIAdapter{
		client: client,
		cfg:    cfg,
	}
}

func (a *OpenAIAdapter) ChatComplete(req model.ChatRequest) (*model.ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}
	}

	openaiReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
	}

	resp, err := a.client.CreateChatCompletion(context.Background(), openaiReq)
	if err != nil {
		return nil, err
	}

	choices := make([]model.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = model.Choice{
			Index:        c.Index,
			Message:      model.ChatMessage{Role: c.Message.Role, Content: c.Message.Content},
			FinishReason: string(c.FinishReason),
		}
	}

	return &model.ChatResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage: model.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (a *OpenAIAdapter) CountTokens(model, text string) (int, error) {
	return len(text) / 4, nil
}

func (a *OpenAIAdapter) Embeddings(req model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	// OpenAI Embeddings 暂未实现
	return nil, fmt.Errorf("openai embeddings not implemented")
}

func (a *OpenAIAdapter) GetModelName() string {
	return "openai"
}
