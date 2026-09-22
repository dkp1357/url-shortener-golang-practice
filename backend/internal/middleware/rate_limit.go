package middleware

import (
	"net"
	"net/http"
	"strings"
	"url-shortener/internal/config"
	"url-shortener/internal/repository/redis"
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
