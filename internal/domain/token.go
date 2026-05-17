package domain

import "time"

// TokenPair contains access and refresh token payload returned to clients.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// AccessTokenClaims is the parsed payload stored in JWT access tokens.
type AccessTokenClaims struct {
	UserID string
	Email  string
	Roles  []string
	Exp    time.Time
}
