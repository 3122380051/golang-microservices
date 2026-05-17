package domain

import "time"

// User models an account in the authentication domain.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Status       string
	Roles        []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
