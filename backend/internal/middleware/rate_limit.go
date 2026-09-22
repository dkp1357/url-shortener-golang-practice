package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"url-shortener/internal/config"
	"url-shortener/internal/repository/redis"
	"url-shortener/internal/utils"
)

type RateLimitMiddleware struct {
	limiter *redis.RateLimiter
	cfg     *config.Config
}

func NewRateLimitMiddleware(limiter *redis.RateLimiter, cfg *config.Config) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		cfg:     cfg,
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (m *RateLimitMiddleware) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := GetUserID(ctx)

		var key string
		var limit int

		if userID != nil {
			key = "user:" + userID.String()
			limit = m.cfg.RateLimitAuthenticated
		} else {
			ip := getClientIP(r)
			key = "ip:" + ip
			limit = m.cfg.RateLimitAnonymous
		}

		result, err := m.limiter.Allow(ctx, key, limit, m.cfg.RateLimitWindow)
		if err != nil {
			// Fail open on Redis errors to prevent taking down the entire service
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))

		if !result.Allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(result.ResetIn.Seconds())))
			utils.ErrorResponse(w, http.StatusTooManyRequests, fmt.Sprintf("rate limit exceeded: try again in %v", result.ResetIn))
			return
		}

		next.ServeHTTP(w, r)
	})
}
