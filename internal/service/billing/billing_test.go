package billing

import (
	"ai-gateway/internal/config"
	"ai-gateway/internal/model"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	cfg := &config.Config{
		Pricing: config.PricingConfig{
			"gpt-4": config.PricingItem{
				Prompt:     0.03,
				Completion: 0.06,
			},
			"abab6.5s-chat": config.PricingItem{
				Prompt:     0.01,
				Completion: 0.01,
			},
		},
	}

	svc := NewBillingService(nil, cfg)

	// Test GPT-4 pricing: 1000 prompt + 1000 completion = $0.03 + $0.06 = $0.09
	cost, err := svc.CalculateCost("gpt-4", 1000, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if cost != 0.09 {
		t.Errorf("Expected cost 0.09, got %f", cost)
	}

	// Test MiniMax pricing: 1000 prompt + 1000 completion = $0.01 + $0.01 = $0.02
	cost, err = svc.CalculateCost("abab6.5s-chat", 1000, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if cost != 0.02 {
		t.Errorf("Expected cost 0.02, got %f", cost)
	}

	// Test unknown model
	_, err = svc.CalculateCost("unknown-model", 1000, 1000)
	if err == nil {
		t.Errorf("Expected error for unknown model")
	}
}

func TestChatRequestModel(t *testing.T) {
	req := model.ChatRequest{
		Model: "abab6.5s-chat",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:  1000,
	}

	if req.Model != "abab6.5s-chat" {
		t.Errorf("Expected model abab6.5s-chat")
	}

	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message")
	}

	if req.Messages[0].Role != "user" {
		t.Errorf("Expected role user")
	}
}
