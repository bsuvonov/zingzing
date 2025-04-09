package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJWT(t *testing.T) {
	// Set up test variables
	userID := uuid.New()
	validSecret := "valid-secret"
	invalidSecret := "invalid-secret"
	shortExpiration := 1 * time.Second
	
	// Test case 1: Create and validate a valid JWT
	t.Run("Valid JWT", func(t *testing.T) {
		// Create a JWT
		token, err := MakeJWT(userID, validSecret, 1*time.Hour)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		
		// Validate the JWT
		extractedID, err := ValidateJWT(token, validSecret)
		assert.NoError(t, err)
		assert.Equal(t, userID, extractedID)
	})
	
	// Test case 2: Reject JWT signed with wrong secret
	t.Run("Invalid Secret", func(t *testing.T) {
		// Create a JWT with validSecret
		token, err := MakeJWT(userID, validSecret, 1*time.Hour)
		assert.NoError(t, err)
		
		// Try to validate with invalidSecret
		_, err = ValidateJWT(token, invalidSecret)
		assert.Error(t, err)
	})
	
	// Test case 3: Reject expired JWT
t.Run("Expired Token", func(t *testing.T) {
    // Create a JWT that expires quickly
    token, err := MakeJWT(userID, validSecret, shortExpiration)
    assert.NoError(t, err)
    
    // Wait for token to expire
    time.Sleep(shortExpiration + 100*time.Millisecond)
    
    // Try to validate the expired token
    _, err = ValidateJWT(token, validSecret)
    assert.Error(t, err)
	})
}

