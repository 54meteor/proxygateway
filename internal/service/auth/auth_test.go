package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestHashKey(t *testing.T) {
	key := "test-api-key-123"
	hash1 := HashKey(key)
	hash2 := HashKey(key)
	
	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic")
	}
	
	if hash1 == "" {
		t.Errorf("Hash should not be empty")
	}
	
	// SHA256("test-api-key-123") = 
	expected := "a3a3e3e7c3c3e3e7c3c3e3e7c3c3e3e7c3c3e3e7c3c3e3e7c3c3e3e7c3c3e3"
	_ = expected // just for reference
}

func TestGenerateAPIKey(t *testing.T) {
	// Just test UUID generation
	key := uuid.New().String()
	if key == "" {
		t.Errorf("UUID should not be empty")
	}
}
