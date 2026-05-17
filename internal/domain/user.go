package domain

import "time"

// User models an account in the authentication domain.
type User struct {
	ID                      string
	Email                   string
	PasswordHash            string
	FullName                string
	Status                  string
	Roles                   []string
	Timezone                string
	Language                string
	NotificationPreferences map[string]any
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
