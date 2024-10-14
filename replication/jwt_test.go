package replication

import (
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := []byte("test_secret")
	nodeID := "test_node"
	nodeURL := "http://localhost:8080"

	// Generate token
	token, err := GenerateToken(nodeID, nodeURL, secret)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate token
	validatedNodeID, validatedURL, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if validatedNodeID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, validatedNodeID)
	}

	if validatedURL != nodeURL {
		t.Errorf("Expected node URL %s, got %s", nodeURL, validatedURL)
	}

	// Test expired token
	claims := jwt.MapClaims{
		"node_id":  nodeID,
		"node_url": nodeURL,
		"exp":      time.Now().Add(-1 * time.Hour).Unix(), // Set expiration to 1 hour ago
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenString, _ := expiredToken.SignedString(secret)
	_, _, err = ValidateToken(expiredTokenString, secret)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}

	// Test invalid token
	_, _, err = ValidateToken("invalid_token", secret)
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}

	// Test wrong secret
	wrongSecret := []byte("wrong_secret")
	_, _, err = ValidateToken(token, wrongSecret)
	if err == nil {
		t.Error("Expected error for wrong secret, got nil")
	}
}
