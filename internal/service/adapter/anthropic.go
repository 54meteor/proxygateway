package adapter

import (
	"context"
	"ai-gateway/internal/config"
	"ai-gateway/internal/model"

	"github.com/sashabaranov/go-openai"
)

// AnthropicAdapter Anthropic (Claude) 适配器
type AnthropicAdapter struct {
	client *openai.Client
	cfg    *config.Config
}

func NewAnthropicAdapter(cfg *config.Config) *AnthropicAdapter {
	cfgCopy := *cfg
	client := openai.NewClient(cfgCopy.Models.Anthropic.APIKey)
	return &AnthropicAdapter{
		client: client,
		cfg:    cfg,
	}
}

func (a *AnthropicAdapter) ChatComplete(req model.ChatRequest) (*model.ChatResponse, error) {
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

func (a *AnthropicAdapter) CountTokens(model, text string) (int, error) {
	return len(text) / 4, nil
}

func (a *AnthropicAdapter) GetModelName() string {
	return "anthropic"
}
