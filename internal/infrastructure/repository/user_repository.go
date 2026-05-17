package repository

import (
	"context"
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
