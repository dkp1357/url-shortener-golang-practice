package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url-shortener/internal/config"
	"url-shortener/internal/handler"
	"url-shortener/internal/middleware"
	"url-shortener/internal/repository/postgres"
	"url-shortener/internal/repository/redis"
	"url-shortener/internal/service"
)

func main() {
	log.Println("Starting URL Shortener service...")

	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. Initialize PostgreSQL
	db, err := postgres.NewDB(ctx, cfg.PostgresDSN())
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}
	defer db.Close()

	// Run auto migrations
	if err := db.RunMigrations(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// 3. Initialize Redis
	redisClient, err := redis.NewRedisClient(ctx, cfg.RedisAddr(), cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redisClient.Close()

	// 4. Initialize Repositories
	userRepo := postgres.NewUserRepository(db)
	urlRepo := postgres.NewURLRepository(db)
	clickRepo := postgres.NewClickRepository(db)
	cacheRepo := redis.NewCacheRepository(redisClient)
	rateLimiter := redis.NewRateLimiter(redisClient)

	// 5. Initialize Services
	authService := service.NewAuthService(userRepo, cfg)
	analyticsService := service.NewAnalyticsService(clickRepo, urlRepo)
	defer analyticsService.Stop()

	urlService := service.NewURLService(urlRepo, cacheRepo, analyticsService, cfg)

	// 6. Initialize Middlewares
	authMiddleware := middleware.NewAuthMiddleware(authService)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter, cfg)

	// 7. Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
	urlHandler := handler.NewURLHandler(urlService, analyticsService)
	redirectHandler := handler.NewRedirectHandler(urlService)

	// 8. Router
	router := handler.SetupRouter(&handler.RouterDependencies{
		Config:          cfg,
		DB:              db,
		RedisClient:     redisClient,
		AuthHandler:     authHandler,
		URLHandler:      urlHandler,
		RedirectHandler: redirectHandler,
		AuthMiddleware:  authMiddleware,
		RateLimiter:     rateLimitMiddleware,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on port %s (Base URL: %s)", cfg.Port, cfg.BaseURL)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}
