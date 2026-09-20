package models

import (
	"time"

	"github.com/google/uuid"
)

type Click struct {
	ID        uuid.UUID `json:"id"`
	URLID     uuid.UUID `json:"url_id"`
	Referrer  string    `json:"referrer"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
}

type ClickStats struct {
	TotalClicks   int64            `json:"total_clicks"`
	Referrers     map[string]int64 `json:"referrers"`
	DailyTimeline map[string]int64 `json:"daily_timeline"`
}
