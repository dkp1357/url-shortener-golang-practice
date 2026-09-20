package models

import (
	"time"

	"github.com/google/uuid"
)

type URL struct {
	ID          uuid.UUID  `json:"id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	CustomAlias *string    `json:"custom_alias,omitempty"`
	IsActive    bool       `json:"is_active"`
	ClickCount  int64      `json:"click_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.ExpiresAt)
}

// ActiveIdentifier returns the custom alias if set, otherwise the short code.
func (u *URL) ActiveIdentifier() string {
	if u.CustomAlias != nil && *u.CustomAlias != "" {
		return *u.CustomAlias
	}
	return u.ShortCode
}
