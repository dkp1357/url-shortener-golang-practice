package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"url-shortener/internal/models"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrURLNotFound      = errors.New("short URL not found")
	ErrAliasAlreadyUsed = errors.New("custom alias or short code already in use")
	ErrUnauthorized     = errors.New("unauthorized to perform action on this URL")
)

type URLRepository struct {
	db *DB
}

func NewURLRepository(db *DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Create(ctx context.Context, u *models.URL) error {
	query := `
		INSERT INTO urls (user_id, original_url, short_code, custom_alias, is_active, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, click_count, created_at, updated_at
	`
	now := time.Now()
	err := r.db.Pool.QueryRow(ctx, query,
		u.UserID,
		u.OriginalURL,
		u.ShortCode,
		u.CustomAlias,
		u.IsActive,
		u.ExpiresAt,
		now,
		now,
	).Scan(&u.ID, &u.ClickCount, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return ErrAliasAlreadyUsed
		}
		return fmt.Errorf("failed to create URL: %w", err)
	}

	return nil
}

func (r *URLRepository) GetByCodeOrAlias(ctx context.Context, identifier string) (*models.URL, error) {
	query := `
		SELECT id, user_id, original_url, short_code, custom_alias, is_active, click_count, expires_at, created_at, updated_at
		FROM urls
		WHERE short_code = $1 OR custom_alias = $1
	`
	u := &models.URL{}
	err := r.db.Pool.QueryRow(ctx, query, identifier).
		Scan(&u.ID, &u.UserID, &u.OriginalURL, &u.ShortCode, &u.CustomAlias, &u.IsActive, &u.ClickCount, &u.ExpiresAt, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return u, nil
}

func (r *URLRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.URL, error) {
	query := `
		SELECT id, user_id, original_url, short_code, custom_alias, is_active, click_count, expires_at, created_at, updated_at
		FROM urls
		WHERE id = $1
	`
	u := &models.URL{}
	err := r.db.Pool.QueryRow(ctx, query, id).
		Scan(&u.ID, &u.UserID, &u.OriginalURL, &u.ShortCode, &u.CustomAlias, &u.IsActive, &u.ClickCount, &u.ExpiresAt, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("failed to get URL by id: %w", err)
	}

	return u, nil
}

func (r *URLRepository) Update(ctx context.Context, u *models.URL) error {
	query := `
		UPDATE urls
		SET original_url = $1, is_active = $2, expires_at = $3, updated_at = $4
		WHERE id = $5
		RETURNING updated_at
	`
	now := time.Now()
	err := r.db.Pool.QueryRow(ctx, query, u.OriginalURL, u.IsActive, u.ExpiresAt, now, u.ID).
		Scan(&u.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrURLNotFound
		}
		return fmt.Errorf("failed to update URL: %w", err)
	}

	return nil
}

func (r *URLRepository) Delete(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error {
	var query string
	var args []any

	if userID != nil {
		query = "DELETE FROM urls WHERE id = $1 AND user_id = $2"
		args = []any{id, *userID}
	} else {
		query = "DELETE FROM urls WHERE id = $1"
		args = []any{id}
	}

	cmdTag, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete URL: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrURLNotFound
	}

	return nil
}

func (r *URLRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.URL, int64, error) {
	countQuery := "SELECT COUNT(*) FROM urls WHERE user_id = $1"
	var totalCount int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, userID).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count user URLs: %w", err)
	}

	selectQuery := `
		SELECT id, user_id, original_url, short_code, custom_alias, is_active, click_count, expires_at, created_at, updated_at
		FROM urls
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, selectQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list user URLs: %w", err)
	}
	defer rows.Close()

	urls := make([]models.URL, 0)
	for rows.Next() {
		var u models.URL
		if err := rows.Scan(&u.ID, &u.UserID, &u.OriginalURL, &u.ShortCode, &u.CustomAlias, &u.IsActive, &u.ClickCount, &u.ExpiresAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan URL row: %w", err)
		}
		urls = append(urls, u)
	}

	return urls, totalCount, nil
}

func (r *URLRepository) IncrementClickCount(ctx context.Context, id uuid.UUID, count int) error {
	query := "UPDATE urls SET click_count = click_count + $2 WHERE id = $1"
	_, err := r.db.Pool.Exec(ctx, query, id, count)
	return err
}
