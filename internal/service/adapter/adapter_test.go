package adapter

import (
	"ai-gateway/internal/config"
	"testing"
)

func TestFactoryRegistration(t *testing.T) {
	cfg := &config.Config{
		Models: config.ModelsConfig{
			MiniMax: config.ModelProviderConfig{
				Enabled: true,
				APIKey:  "test-key",
			},
		},
	}

	factory := NewFactory(cfg)
	models := factory.ListModels()

	if len(models) == 0 {
		t.Errorf("Expected at least one model registered")
	}

	// Check MiniMax models are registered
	_, ok := factory.Get("abab6.5s-chat")
	if !ok {
		t.Errorf("Expected abab6.5s-chat to be registered")
	}

	// Check non-existent model
	_, ok = factory.Get("non-existent-model")
	if ok {
		t.Errorf("Expected non-existent model to return false")
	}
}

func TestConfigPricing(t *testing.T) {
	cfg := &config.Config{
		Pricing: config.PricingConfig{
			"gpt-4": config.PricingItem{
				Prompt:     0.03,
				Completion: 0.06,
			},
		},
	}

	pricing, ok := cfg.Pricing["gpt-4"]
	if !ok {
		t.Errorf("Expected pricing for gpt-4")
	}

	if pricing.Prompt != 0.03 {
		t.Errorf("Expected prompt price 0.03, got %f", pricing.Prompt)
	}
}
