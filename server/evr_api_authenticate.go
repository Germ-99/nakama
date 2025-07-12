package server

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

func generateClientToken(encKey string, userID, username string) (token string, tokenID string, err error) {
	exp := time.Now().Add(3650 * 24 * time.Hour).Unix() // 10 years
	tokenID = uuid.Must(uuid.NewV4()).String()
	tokenIssuedAt := time.Now().Unix()
	var vars map[string]string
	token, _ = generateTokenWithExpiry(encKey, tokenID, tokenIssuedAt, userID, username, vars, exp)
	// Generate a new auth token for the user.
	return token, tokenID, nil
}
