package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	userapp "github.com/3122380051/golang-microservices/internal/application/user"
	"github.com/3122380051/golang-microservices/internal/domain"
)

type fakeEncryptor struct{}

func (fakeEncryptor) Encrypt(plaintext string) (string, error)  { return "enc:" + plaintext, nil }
func (fakeEncryptor) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

type memoryUserServiceRepo struct {
	users   map[string]domain.User
	apiKeys map[string][]domain.APIKey
	audits  int
}

func newMemoryUserServiceRepo() *memoryUserServiceRepo {
	return &memoryUserServiceRepo{
		users: map[string]domain.User{
			"u1": {
				ID:       "u1",
				Email:    "u1@example.com",
				FullName: "User One",
				Status:   "active",
			},
		},
		apiKeys: make(map[string][]domain.APIKey),
	}
}

func (r *memoryUserServiceRepo) GetUserByID(_ context.Context, userID string) (domain.User, error) {
	u, ok := r.users[userID]
	if !ok {
		return domain.User{}, errors.New("not found")
	}
	return u, nil
}

func (r *memoryUserServiceRepo) GetRolesByUserID(_ context.Context, userID string) ([]string, error) {
	if _, ok := r.users[userID]; !ok {
		return nil, errors.New("not found")
	}
	return []string{"user"}, nil
}

func (r *memoryUserServiceRepo) UpsertUserSettings(_ context.Context, user domain.User) error {
	u, ok := r.users[user.ID]
	if !ok {
		return errors.New("not found")
	}
	u.FullName = user.FullName
	u.Timezone = user.Timezone
	u.Language = user.Language
	u.NotificationPreferences = user.NotificationPreferences
	u.UpdatedAt = time.Now()
	r.users[user.ID] = u
	return nil
}

func (r *memoryUserServiceRepo) CreateAPIKey(_ context.Context, apiKey domain.APIKey) (domain.APIKey, error) {
	apiKey.ID = "k" + time.Now().Format("150405")
	apiKey.CreatedAt = time.Now()
	r.apiKeys[apiKey.UserID] = append(r.apiKeys[apiKey.UserID], apiKey)
	return apiKey, nil
}

func (r *memoryUserServiceRepo) ListAPIKeys(_ context.Context, userID string) ([]domain.APIKey, error) {
	return r.apiKeys[userID], nil
}

func (r *memoryUserServiceRepo) DeleteAPIKey(_ context.Context, userID, apiKeyID string) error {
	items := r.apiKeys[userID]
	out := items[:0]
	found := false
	for _, item := range items {
		if item.ID == apiKeyID {
			found = true
			continue
		}
		out = append(out, item)
	}
	if !found {
		return errors.New("not found")
	}
	r.apiKeys[userID] = out
	return nil
}

func (r *memoryUserServiceRepo) WriteAuditLog(_ context.Context, _, _, _, _ string, _ map[string]any, _ string) error {
	r.audits++
	return nil
}

func TestUserServiceProfileAndAPIKeys(t *testing.T) {
	repo := newMemoryUserServiceRepo()
	svc := userapp.NewService(repo, fakeEncryptor{})
	ctx := context.Background()

	updated, err := svc.UpdateProfile(ctx, "u1", "Updated Name", "Asia/Ho_Chi_Minh", "vi", map[string]any{"email": true})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if updated.FullName != "Updated Name" || updated.Language != "vi" {
		t.Fatalf("profile update not applied")
	}

	created, err := svc.CreateAPIKey(ctx, "u1", "binance", "ABCDEFGHIJK", "SECRETSECRET", "main")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if created.APIKey == "ABCDEFGHIJK" {
		t.Fatalf("api key should be masked")
	}

	items, err := svc.ListAPIKeys(ctx, "u1")
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 api key, got %d", len(items))
	}

	raw := repo.apiKeys["u1"][0]
	if err := svc.DeleteAPIKey(ctx, "u1", raw.ID); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}

	items, err = svc.ListAPIKeys(ctx, "u1")
	if err != nil {
		t.Fatalf("ListAPIKeys after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 api keys after delete, got %d", len(items))
	}

	if repo.audits < 3 {
		t.Fatalf("expected audit logs to be written, got %d", repo.audits)
	}
}
