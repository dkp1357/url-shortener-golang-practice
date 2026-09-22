package models

import (
	"time"

	"github.com/google/uuid"
)

// Auth DTOs
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// URL DTOs
type CreateURLRequest struct {
	OriginalURL string     `json:"original_url"`
	CustomAlias string     `json:"custom_alias,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type UpdateURLRequest struct {
	OriginalURL *string    `json:"original_url,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type URLResponse struct {
	ID          uuid.UUID  `json:"id"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	CustomAlias *string    `json:"custom_alias,omitempty"`
	ShortURL    string     `json:"short_url"`
	IsActive    bool       `json:"is_active"`
	ClickCount  int64      `json:"click_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CachedURL is stored in Redis
type CachedURL struct {
	ID          uuid.UUID  `json:"id"`
	OriginalURL string     `json:"original_url"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}
