package postgres

import (
	"context"
	"fmt"
	"time"
	"url-shortener/internal/models"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type ClickRepository struct {
	db *DB
}

func NewClickRepository(db *DB) *ClickRepository {
	return &ClickRepository{db: db}
}

func (r *ClickRepository) RecordClick(ctx context.Context, c *models.Click) error {
	query := `
		INSERT INTO clicks (url_id, referrer, user_agent, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	now := time.Now()
	return r.db.Pool.QueryRow(ctx, query, c.URLID, c.Referrer, c.UserAgent, c.IPAddress, now).
		Scan(&c.ID, &c.CreatedAt)
}

func (r *ClickRepository) RecordBatch(ctx context.Context, clicks []models.Click) error {
	if len(clicks) == 0 {
		return nil
	}

	rows := make([][]any, len(clicks))
	for i, c := range clicks {
		createdAt := c.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		rows[i] = []any{c.URLID, c.Referrer, c.UserAgent, c.IPAddress, createdAt}
	}

	_, err := r.db.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"clicks"},
		[]string{"url_id", "referrer", "user_agent", "ip_address", "created_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("failed to batch insert clicks: %w", err)
	}

	return nil
}

func (r *ClickRepository) GetStatsByURLID(ctx context.Context, urlID uuid.UUID) (*models.ClickStats, error) {
	stats := &models.ClickStats{
		Referrers:     make(map[string]int64),
		DailyTimeline: make(map[string]int64),
	}

	// Total clicks
	var total int64
	err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM clicks WHERE url_id = $1", urlID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count clicks: %w", err)
	}
	stats.TotalClicks = total

	// Top referrers
	refQuery := `
		SELECT COALESCE(NULLIF(referrer, ''), 'Direct / Unknown') as ref, COUNT(*) as cnt
		FROM clicks
		WHERE url_id = $1
		GROUP BY ref
		ORDER BY cnt DESC
		LIMIT 10
	`
	rows, err := r.db.Pool.Query(ctx, refQuery, urlID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ref string
			var cnt int64
			if err := rows.Scan(&ref, &cnt); err == nil {
				stats.Referrers[ref] = cnt
			}
		}
	}

	// Daily timeline (last 30 days)
	timelineQuery := `
		SELECT TO_CHAR(created_at, 'YYYY-MM-DD') as day, COUNT(*) as cnt
		FROM clicks
		WHERE url_id = $1 AND created_at >= NOW() - INTERVAL '30 days'
		GROUP BY day
		ORDER BY day ASC
	`
	tRows, err := r.db.Pool.Query(ctx, timelineQuery, urlID)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var day string
			var cnt int64
			if err := tRows.Scan(&day, &cnt); err == nil {
				stats.DailyTimeline[day] = cnt
			}
		}
	}

	return stats, nil
}
