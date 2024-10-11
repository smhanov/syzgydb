package replication

import (
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

func GenerateToken(nodeID string, secret []byte) (string, error) {
	claims := jwt.MapClaims{
		"node_id": nodeID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ValidateToken(tokenString string, secret []byte) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	nodeID, ok := claims["node_id"].(string)
	if !ok {
		return "", errors.New("node_id not found in token")
	}
	return nodeID, nil
}
