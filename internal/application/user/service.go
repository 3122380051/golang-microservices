package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
)

var (
	ErrInvalidUserID = errors.New("invalid user id")
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotFound      = errors.New("not found")
)

// Encryptor abstracts secret encryption for api keys.
type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// Repository defines persistence operations used by user service.
type Repository interface {
	GetUserByID(ctx context.Context, userID string) (domain.User, error)
	GetRolesByUserID(ctx context.Context, userID string) ([]string, error)
	UpsertUserSettings(ctx context.Context, user domain.User) error
	CreateAPIKey(ctx context.Context, apiKey domain.APIKey) (domain.APIKey, error)
	ListAPIKeys(ctx context.Context, userID string) ([]domain.APIKey, error)
	DeleteAPIKey(ctx context.Context, userID, apiKeyID string) error
	WriteAuditLog(ctx context.Context, userID, action, entityType, entityID string, metadata map[string]any, traceID string) error
}

// Service provides user profile and api key management use-cases.
type Service struct {
	repo      Repository
	encryptor Encryptor
}

func NewService(repo Repository, encryptor Encryptor) *Service {
	return &Service{repo: repo, encryptor: encryptor}
}

func (s *Service) GetProfile(ctx context.Context, userID string) (domain.User, error) {
	if strings.TrimSpace(userID) == "" {
		return domain.User{}, ErrInvalidUserID
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	roles, err := s.repo.GetRolesByUserID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("get roles: %w", err)
	}
	user.Roles = roles
	if user.Timezone == "" {
		user.Timezone = "UTC"
	}
	if user.Language == "" {
		user.Language = "en"
	}
	if user.NotificationPreferences == nil {
		user.NotificationPreferences = map[string]any{}
	}

	return user, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID, fullName, timezone, language string, notificationPrefs map[string]any) (domain.User, error) {
	if strings.TrimSpace(userID) == "" {
		return domain.User{}, ErrInvalidUserID
	}
	if strings.TrimSpace(language) == "" {
		language = "en"
	}
	if strings.TrimSpace(timezone) == "" {
		timezone = "UTC"
	}
	if notificationPrefs == nil {
		notificationPrefs = map[string]any{}
	}

	profile, err := s.GetProfile(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}

	profile.FullName = strings.TrimSpace(fullName)
	profile.Timezone = timezone
	profile.Language = language
	profile.NotificationPreferences = notificationPrefs
	profile.UpdatedAt = time.Now()

	if err := s.repo.UpsertUserSettings(ctx, profile); err != nil {
		return domain.User{}, fmt.Errorf("update settings: %w", err)
	}

	_ = s.repo.WriteAuditLog(ctx, userID, "user.profile.updated", "user", userID, map[string]any{
		"timezone": timezone,
		"language": language,
	}, "")

	return profile, nil
}

func (s *Service) CreateAPIKey(ctx context.Context, userID, exchange, apiKey, apiSecret, label string) (domain.MaskedAPIKey, error) {
	if strings.TrimSpace(userID) == "" {
		return domain.MaskedAPIKey{}, ErrInvalidUserID
	}
	if strings.TrimSpace(exchange) == "" || strings.TrimSpace(apiKey) == "" || strings.TrimSpace(apiSecret) == "" {
		return domain.MaskedAPIKey{}, ErrInvalidInput
	}
	if err := s.TestConnection(ctx, exchange, apiKey, apiSecret); err != nil {
		return domain.MaskedAPIKey{}, err
	}

	encryptedSecret, err := s.encryptor.Encrypt(apiSecret)
	if err != nil {
		return domain.MaskedAPIKey{}, fmt.Errorf("encrypt secret: %w", err)
	}

	created, err := s.repo.CreateAPIKey(ctx, domain.APIKey{
		UserID:             userID,
		Exchange:           strings.ToLower(strings.TrimSpace(exchange)),
		APIKey:             strings.TrimSpace(apiKey),
		APISecretEncrypted: encryptedSecret,
		Label:              strings.TrimSpace(label),
		IsActive:           true,
	})
	if err != nil {
		return domain.MaskedAPIKey{}, fmt.Errorf("create api key: %w", err)
	}

	_ = s.repo.WriteAuditLog(ctx, userID, "user.api_key.created", "user_api_key", created.ID, map[string]any{
		"exchange": created.Exchange,
		"label":    created.Label,
	}, "")

	return toMasked(created), nil
}

func (s *Service) ListAPIKeys(ctx context.Context, userID string) ([]domain.MaskedAPIKey, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidUserID
	}

	items, err := s.repo.ListAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	out := make([]domain.MaskedAPIKey, 0, len(items))
	for _, item := range items {
		out = append(out, toMasked(item))
	}
	return out, nil
}

func (s *Service) DeleteAPIKey(ctx context.Context, userID, apiKeyID string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(apiKeyID) == "" {
		return ErrInvalidInput
	}

	if err := s.repo.DeleteAPIKey(ctx, userID, apiKeyID); err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}

	_ = s.repo.WriteAuditLog(ctx, userID, "user.api_key.deleted", "user_api_key", apiKeyID, nil, "")
	return nil
}

// TestConnection is a safe placeholder for exchange key validation.
func (s *Service) TestConnection(ctx context.Context, exchange, apiKey, apiSecret string) error {
	_ = ctx
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	if exchange != "binance" && exchange != "bybit" && exchange != "okx" {
		return fmt.Errorf("unsupported exchange: %s", exchange)
	}
	if len(strings.TrimSpace(apiKey)) < 6 || len(strings.TrimSpace(apiSecret)) < 8 {
		return ErrInvalidInput
	}
	return nil
}

func toMasked(item domain.APIKey) domain.MaskedAPIKey {
	return domain.MaskedAPIKey{
		ID:        item.ID,
		Exchange:  item.Exchange,
		APIKey:    maskKey(item.APIKey),
		Label:     item.Label,
		IsActive:  item.IsActive,
		CreatedAt: item.CreatedAt,
	}
}

func maskKey(apiKey string) string {
	if len(apiKey) <= 6 {
		return "***"
	}
	return apiKey[:3] + strings.Repeat("*", len(apiKey)-6) + apiKey[len(apiKey)-3:]
}
