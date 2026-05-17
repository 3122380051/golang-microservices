package domain

import "time"

// APIKey stores exchange credentials owned by a user.
type APIKey struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	Exchange           string    `json:"exchange"`
	APIKey             string    `json:"api_key"`
	APISecretEncrypted string    `json:"-"`
	Label              string    `json:"label"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// MaskedAPIKey is a safe payload returned by APIs.
type MaskedAPIKey struct {
	ID        string    `json:"id"`
	Exchange  string    `json:"exchange"`
	APIKey    string    `json:"api_key"`
	Label     string    `json:"label"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
