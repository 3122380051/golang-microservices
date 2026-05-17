package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrWeakPassword       = errors.New("weak password")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredential  = errors.New("invalid credential")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenRevoked       = errors.New("token revoked")
)

// UserRepository defines persistence operations for auth workflows.
type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	AssignRoleByName(ctx context.Context, userID, roleName string) error
	GetRolesByUserID(ctx context.Context, userID string) ([]string, error)
}

type refreshSession struct {
	UserID    string
	Email     string
	Roles     []string
	ExpiresAt time.Time
	Revoked   bool
}

// Service contains business logic for auth service.
type Service struct {
	repo             UserRepository
	jwtSecret        []byte
	accessTTL        time.Duration
	refreshTTL       time.Duration
	mu               sync.Mutex
	refreshSessions  map[string]refreshSession
	revokedAccessJWT map[string]time.Time
}

func NewService(repo UserRepository, jwtSecret string, accessTTL, refreshTTL time.Duration) *Service {
	if accessTTL <= 0 {
		accessTTL = time.Hour
	}
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}

	return &Service{
		repo:             repo,
		jwtSecret:        []byte(jwtSecret),
		accessTTL:        accessTTL,
		refreshTTL:       refreshTTL,
		refreshSessions:  make(map[string]refreshSession),
		revokedAccessJWT: make(map[string]time.Time),
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (string, domain.TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", domain.TokenPair{}, ErrInvalidEmail
	}
	if len(password) < 8 {
		return "", domain.TokenPair{}, ErrWeakPassword
	}

	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return "", domain.TokenPair{}, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", domain.TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	created, err := s.repo.CreateUser(ctx, domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Status:       "active",
	})
	if err != nil {
		return "", domain.TokenPair{}, fmt.Errorf("create user: %w", err)
	}

	if err := s.repo.AssignRoleByName(ctx, created.ID, "user"); err != nil {
		return "", domain.TokenPair{}, fmt.Errorf("assign default role: %w", err)
	}

	roles, err := s.repo.GetRolesByUserID(ctx, created.ID)
	if err != nil {
		return "", domain.TokenPair{}, fmt.Errorf("get roles: %w", err)
	}

	tokens, err := s.newTokenPair(created.ID, created.Email, roles)
	if err != nil {
		return "", domain.TokenPair{}, err
	}

	return created.ID, tokens, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, domain.TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", domain.TokenPair{}, ErrInvalidCredential
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", domain.TokenPair{}, ErrInvalidCredential
	}

	roles, err := s.repo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return "", domain.TokenPair{}, fmt.Errorf("get roles: %w", err)
	}

	tokens, err := s.newTokenPair(user.ID, user.Email, roles)
	if err != nil {
		return "", domain.TokenPair{}, err
	}

	return user.ID, tokens, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (domain.TokenPair, error) {
	_ = ctx
	s.mu.Lock()
	s.pruneRevokedLocked(time.Now())
	session, ok := s.refreshSessions[refreshToken]
	if !ok {
		s.mu.Unlock()
		return domain.TokenPair{}, ErrInvalidToken
	}
	if session.Revoked {
		s.mu.Unlock()
		return domain.TokenPair{}, ErrTokenRevoked
	}
	if time.Now().After(session.ExpiresAt) {
		session.Revoked = true
		s.refreshSessions[refreshToken] = session
		s.mu.Unlock()
		return domain.TokenPair{}, ErrTokenExpired
	}
	session.Revoked = true
	s.refreshSessions[refreshToken] = session
	s.mu.Unlock()

	return s.newTokenPair(session.UserID, session.Email, session.Roles)
}

func (s *Service) Logout(ctx context.Context, accessToken string) error {
	_ = ctx
	claims, err := s.parseAccessToken(accessToken)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.revokedAccessJWT[accessToken] = claims.Exp
	s.mu.Unlock()
	return nil
}

func (s *Service) ValidateAccessToken(ctx context.Context, accessToken string) (domain.AccessTokenClaims, error) {
	_ = ctx
	s.mu.Lock()
	exp, found := s.revokedAccessJWT[accessToken]
	if found && time.Now().Before(exp) {
		s.mu.Unlock()
		return domain.AccessTokenClaims{}, ErrTokenRevoked
	}
	s.pruneRevokedLocked(time.Now())
	s.mu.Unlock()

	return s.parseAccessToken(accessToken)
}

func (s *Service) parseAccessToken(accessToken string) (domain.AccessTokenClaims, error) {
	tok, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})
	if err != nil || !tok.Valid {
		return domain.AccessTokenClaims{}, ErrInvalidToken
	}

	claimsMap, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return domain.AccessTokenClaims{}, ErrInvalidToken
	}

	expValue, err := claimsMap.GetExpirationTime()
	if err != nil || expValue == nil {
		return domain.AccessTokenClaims{}, ErrInvalidToken
	}
	if time.Now().After(expValue.Time) {
		return domain.AccessTokenClaims{}, ErrTokenExpired
	}

	userID, _ := claimsMap["user_id"].(string)
	email, _ := claimsMap["email"].(string)
	roles := toStringSlice(claimsMap["roles"])

	return domain.AccessTokenClaims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		Exp:    expValue.Time,
	}, nil
}

func (s *Service) newTokenPair(userID, email string, roles []string) (domain.TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(s.accessTTL)
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"roles":   roles,
		"iat":     now.Unix(),
		"exp":     accessExp.Unix(),
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err := randomToken(32)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	s.mu.Lock()
	s.refreshSessions[refreshToken] = refreshSession{
		UserID:    userID,
		Email:     email,
		Roles:     roles,
		ExpiresAt: now.Add(s.refreshTTL),
	}
	s.mu.Unlock()

	return domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *Service) pruneRevokedLocked(now time.Time) {
	for token, exp := range s.revokedAccessJWT {
		if now.After(exp) {
			delete(s.revokedAccessJWT, token)
		}
	}
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func toStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			str, ok := item.(string)
			if ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}
