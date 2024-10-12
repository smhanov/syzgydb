package replication

import (
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
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
	claims := jwt.MapClaims{
		"node_id": nodeID,
		"exp":     time.Now().Add(-1 * time.Hour).Unix(), // Set expiration to 1 hour ago
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenString, _ := expiredToken.SignedString(secret)
	_, err = ValidateToken(expiredTokenString, secret)
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
