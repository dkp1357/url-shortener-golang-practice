package handler

import (
	"net/http"

	"url-shortener/internal/config"
	"url-shortener/internal/middleware"
	"url-shortener/internal/repository/postgres"
	"url-shortener/internal/repository/redis"
	"url-shortener/internal/utils"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type RouterDependencies struct {
	Config          *config.Config
	DB              *postgres.DB
	RedisClient     *redis.RedisClient
	AuthHandler     *AuthHandler
	URLHandler      *URLHandler
	RedirectHandler *RedirectHandler
	AuthMiddleware  *middleware.AuthMiddleware
	RateLimiter     *middleware.RateLimitMiddleware
}

func SetupRouter(deps *RouterDependencies) http.Handler {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestLogger)
	r.Use(chimiddleware.Recoverer)

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-RateLimit-Limit", "X-RateLimit-Remaining", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health Check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		dbErr := deps.DB.Pool.Ping(ctx)
		redisErr := deps.RedisClient.RDB.Ping(ctx).Err()

		status := http.StatusOK
		dbStatus := "healthy"
		redisStatus := "healthy"

		if dbErr != nil {
			status = http.StatusServiceUnavailable
			dbStatus = "unhealthy: " + dbErr.Error()
		}
		if redisErr != nil {
			status = http.StatusServiceUnavailable
			redisStatus = "unhealthy: " + redisErr.Error()
		}

		utils.WriteJSON(w, status, map[string]any{
			"status":   "ok",
			"database": dbStatus,
			"redis":    redisStatus,
		})
	})

	// API v1 Routes
	r.Route("/api/v1", func(api chi.Router) {
		// Rate Limiting applied to API routes
		api.Use(deps.RateLimiter.Limit)

		// Auth Routes
		api.Route("/auth", func(auth chi.Router) {
			auth.Post("/register", deps.AuthHandler.Register)
			auth.Post("/login", deps.AuthHandler.Login)

			// Authenticated auth routes
			auth.Group(func(protected chi.Router) {
				protected.Use(deps.AuthMiddleware.RequireAuth)
				protected.Get("/me", deps.AuthHandler.Me)
			})
		})

		// URL Routes
		api.Route("/urls", func(urls chi.Router) {
			// Create short URL: Optional auth so guest users can also create links
			urls.With(deps.AuthMiddleware.OptionalAuth).Post("/", deps.URLHandler.Create)

			// Get URL details
			urls.Get("/{code}", deps.URLHandler.Get)
			urls.With(deps.AuthMiddleware.OptionalAuth).Get("/{code}/analytics", deps.URLHandler.GetAnalytics)

			// Protected URL routes (owner only)
			urls.Group(func(protected chi.Router) {
				protected.Use(deps.AuthMiddleware.RequireAuth)
				protected.Get("/", deps.URLHandler.List)
				protected.Patch("/{code}", deps.URLHandler.Update)
				protected.Delete("/{code}", deps.URLHandler.Delete)
			})
		})
	})

	// Redirection route (e.g. GET /{code})
	// With rate limiting applied
	r.Group(func(redirects chi.Router) {
		redirects.Use(deps.RateLimiter.Limit)
		redirects.Get("/{code}", deps.RedirectHandler.Redirect)
	})

	return r
}
