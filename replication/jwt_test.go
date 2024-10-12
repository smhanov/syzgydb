package replication

import (
	"testing"
	"time"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := []byte("test_secret")
	nodeID := "test_node"

	// Generate token
	token, err := GenerateToken(nodeID, secret)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate token
	validatedNodeID, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if validatedNodeID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, validatedNodeID)
	}

	// Test expired token
	expiredToken, _ := GenerateToken(nodeID, secret)
	time.Sleep(2 * time.Second)
	_, err = ValidateToken(expiredToken, secret)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}

	// Test invalid token
	_, err = ValidateToken("invalid_token", secret)
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}

	// Test wrong secret
	wrongSecret := []byte("wrong_secret")
	_, err = ValidateToken(token, wrongSecret)
	if err == nil {
		t.Error("Expected error for wrong secret, got nil")
	}
}
