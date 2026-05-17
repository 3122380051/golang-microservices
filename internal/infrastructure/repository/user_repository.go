package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// UserRepository provides PostgreSQL-backed user and role operations.
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	const query = `
		INSERT INTO users (email, password_hash, full_name, status)
		VALUES ($1, $2, COALESCE($3, ''), COALESCE($4, 'active'))
		RETURNING id, email, password_hash, full_name, status, created_at, updated_at
	`

	var out domain.User
	err := r.pool.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(user.Email)), user.PasswordHash, user.FullName, user.Status).
		Scan(&out.ID, &out.Email, &out.PasswordHash, &out.FullName, &out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, fmt.Errorf("duplicate email: %w", err)
		}
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}

	return out, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	const query = `
		SELECT id, email, password_hash, full_name, status, created_at, updated_at
		FROM users WHERE email = $1
	`

	var out domain.User
	err := r.pool.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(email))).
		Scan(&out.ID, &out.Email, &out.PasswordHash, &out.FullName, &out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, fmt.Errorf("select user by email: %w", err)
	}
	return out, nil
}

func (r *UserRepository) AssignRoleByName(ctx context.Context, userID, roleName string) error {
	const query = `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = $2
		ON CONFLICT (user_id, role_id) DO NOTHING
	`
	cmd, err := r.pool.Exec(ctx, query, userID, roleName)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("role not found or already assigned: %s", roleName)
	}
	return nil
}

func (r *UserRepository) GetRolesByUserID(ctx context.Context, userID string) ([]string, error) {
	const query = `
		SELECT r.name
		FROM roles r
		INNER JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	roles := make([]string, 0, 2)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate roles: %w", rows.Err())
	}

	if len(roles) == 0 {
		roles = append(roles, "user")
	}
	return roles, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	const query = `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.status,
		       COALESCE(us.timezone, 'UTC'), COALESCE(us.language, 'en'),
		       COALESCE(us.notification_prefs_json, '{}'::jsonb),
		       u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_settings us ON us.user_id = u.id
		WHERE u.id = $1
	`

	var out domain.User
	var prefs []byte
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&out.ID,
		&out.Email,
		&out.PasswordHash,
		&out.FullName,
		&out.Status,
		&out.Timezone,
		&out.Language,
		&prefs,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, fmt.Errorf("select user by id: %w", err)
	}

	out.NotificationPreferences = map[string]any{}
	if len(prefs) > 0 {
		if err := json.Unmarshal(prefs, &out.NotificationPreferences); err != nil {
			return domain.User{}, fmt.Errorf("parse notification prefs: %w", err)
		}
	}

	return out, nil
}

func (r *UserRepository) UpsertUserSettings(ctx context.Context, user domain.User) error {
	prefs, err := json.Marshal(user.NotificationPreferences)
	if err != nil {
		return fmt.Errorf("marshal notification prefs: %w", err)
	}

	const updateUser = `UPDATE users SET full_name = $2, updated_at = NOW() WHERE id = $1`
	if _, err := r.pool.Exec(ctx, updateUser, user.ID, user.FullName); err != nil {
		return fmt.Errorf("update user profile: %w", err)
	}

	const upsertSettings = `
		INSERT INTO user_settings (user_id, timezone, language, notification_prefs_json)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id)
		DO UPDATE SET
			timezone = EXCLUDED.timezone,
			language = EXCLUDED.language,
			notification_prefs_json = EXCLUDED.notification_prefs_json,
			updated_at = NOW()
	`
	if _, err := r.pool.Exec(ctx, upsertSettings, user.ID, user.Timezone, user.Language, prefs); err != nil {
		return fmt.Errorf("upsert user settings: %w", err)
	}

	return nil
}

func (r *UserRepository) CreateAPIKey(ctx context.Context, apiKey domain.APIKey) (domain.APIKey, error) {
	const query = `
		INSERT INTO user_api_keys (user_id, exchange, api_key, api_secret_encrypted, label, is_active)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, TRUE))
		RETURNING id, user_id, exchange, api_key, api_secret_encrypted, label, is_active, created_at, updated_at
	`

	var out domain.APIKey
	err := r.pool.QueryRow(ctx, query,
		apiKey.UserID,
		apiKey.Exchange,
		apiKey.APIKey,
		apiKey.APISecretEncrypted,
		apiKey.Label,
		apiKey.IsActive,
	).Scan(
		&out.ID,
		&out.UserID,
		&out.Exchange,
		&out.APIKey,
		&out.APISecretEncrypted,
		&out.Label,
		&out.IsActive,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return domain.APIKey{}, fmt.Errorf("insert user api key: %w", err)
	}

	return out, nil
}

func (r *UserRepository) ListAPIKeys(ctx context.Context, userID string) ([]domain.APIKey, error) {
	const query = `
		SELECT id, user_id, exchange, api_key, api_secret_encrypted, label, is_active, created_at, updated_at
		FROM user_api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user api keys: %w", err)
	}
	defer rows.Close()

	items := make([]domain.APIKey, 0)
	for rows.Next() {
		var item domain.APIKey
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Exchange,
			&item.APIKey,
			&item.APISecretEncrypted,
			&item.Label,
			&item.IsActive,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user api key: %w", err)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate user api keys: %w", rows.Err())
	}

	return items, nil
}

func (r *UserRepository) DeleteAPIKey(ctx context.Context, userID, apiKeyID string) error {
	const query = `DELETE FROM user_api_keys WHERE id = $1 AND user_id = $2`
	cmd, err := r.pool.Exec(ctx, query, apiKeyID, userID)
	if err != nil {
		return fmt.Errorf("delete user api key: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) WriteAuditLog(
	ctx context.Context,
	userID,
	action,
	entityType,
	entityID string,
	metadata map[string]any,
	traceID string,
) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = "n/a"
	}

	const query = `
		INSERT INTO audit_logs (user_id, action, entity_type, entity_id, metadata_json, trace_id)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6)
	`
	if _, err := r.pool.Exec(ctx, query, userID, action, entityType, entityID, metadataJSON, traceID); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
