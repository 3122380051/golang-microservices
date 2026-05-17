package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	authapp "github.com/3122380051/golang-microservices/internal/application/auth"
	"github.com/3122380051/golang-microservices/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type memoryUserRepo struct {
	byEmail map[string]domain.User
	byID    map[string]domain.User
	nextID  int
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{
		byEmail: make(map[string]domain.User),
		byID:    make(map[string]domain.User),
		nextID:  1,
	}
}

func (r *memoryUserRepo) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	if _, ok := r.byEmail[user.Email]; ok {
		return domain.User{}, errors.New("duplicate")
	}
	id := "user-" + string(rune('0'+r.nextID))
	r.nextID++
	user.ID = id
	if user.Status == "" {
		user.Status = "active"
	}
	r.byEmail[user.Email] = user
	r.byID[user.ID] = user
	return user, nil
}

func (r *memoryUserRepo) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return domain.User{}, errors.New("not found")
	}
	return u, nil
}

func (r *memoryUserRepo) AssignRoleByName(_ context.Context, userID, roleName string) error {
	u, ok := r.byID[userID]
	if !ok {
		return errors.New("not found")
	}
	u.Roles = appendUnique(u.Roles, roleName)
	r.byID[userID] = u
	r.byEmail[u.Email] = u
	return nil
}

func (r *memoryUserRepo) GetRolesByUserID(_ context.Context, userID string) ([]string, error) {
	u, ok := r.byID[userID]
	if !ok {
		return nil, errors.New("not found")
	}
	if len(u.Roles) == 0 {
		return []string{"user"}, nil
	}
	return u.Roles, nil
}

func TestAuthRegisterDuplicateAndWeakPassword(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := authapp.NewService(repo, "secret", time.Hour, 24*time.Hour)
	ctx := context.Background()

	if _, _, err := svc.Register(ctx, "user@example.com", "1234567"); !errors.Is(err, authapp.ErrWeakPassword) {
		t.Fatalf("expected weak password error, got %v", err)
	}

	_, _, err := svc.Register(ctx, "user@example.com", "12345678")
	if err != nil {
		t.Fatalf("register should succeed: %v", err)
	}

	_, _, err = svc.Register(ctx, "user@example.com", "12345678")
	if !errors.Is(err, authapp.ErrEmailAlreadyExists) {
		t.Fatalf("expected duplicate email error, got %v", err)
	}
}

func TestAuthLoginInvalidCredential(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := authapp.NewService(repo, "secret", time.Hour, 24*time.Hour)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	created, err := repo.CreateUser(ctx, domain.User{Email: "login@example.com", PasswordHash: string(hash), Status: "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	_ = repo.AssignRoleByName(ctx, created.ID, "user")

	_, _, err = svc.Login(ctx, "login@example.com", "wrong-password")
	if !errors.Is(err, authapp.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestAuthRefreshAndLogout(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := authapp.NewService(repo, "secret", time.Hour, 24*time.Hour)
	ctx := context.Background()

	_, tokens, err := svc.Register(ctx, "rotate@example.com", "verysecure")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	nextTokens, err := svc.Refresh(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if nextTokens.AccessToken == "" || nextTokens.RefreshToken == "" {
		t.Fatalf("expected rotated tokens")
	}

	_, err = svc.Refresh(ctx, tokens.RefreshToken)
	if !errors.Is(err, authapp.ErrTokenRevoked) {
		t.Fatalf("expected revoked refresh token, got %v", err)
	}

	if err := svc.Logout(ctx, nextTokens.AccessToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	_, err = svc.ValidateAccessToken(ctx, nextTokens.AccessToken)
	if !errors.Is(err, authapp.ErrTokenRevoked) {
		t.Fatalf("expected revoked access token, got %v", err)
	}
}

func appendUnique(input []string, value string) []string {
	for _, item := range input {
		if item == value {
			return input
		}
	}
	return append(input, value)
}
