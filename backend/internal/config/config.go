package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	BaseURL     string
	Environment string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// JWT
	JWTSecret          string
	JWTExpirationHours time.Duration

	// Cache
	CacheDefaultTTL time.Duration

	// Rate Limiting
	RateLimitAnonymous     int
	RateLimitAuthenticated int
	RateLimitWindow        time.Duration
}

func LoadConfig() (*Config, error) {

	_ = godotenv.Load()

	jwtHours := getEnvAsInt("JWT_EXPIRATION_HOURS", 72)
	cacheTTL := getEnvAsInt("CACHE_DEFAULT_TTL", 86400)
	rateWindow := getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60)

	cfg := &Config{
		Port:                   getEnv("PORT", "8080"),
		BaseURL:                getEnv("BASE_URL", "http://localhost:8080"),
		Environment:            getEnv("ENV", "development"),
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBUser:                 getEnv("DB_USER", "postgres"),
		DBPassword:             getEnv("DB_PASSWORD", "postgres"),
		DBName:                 getEnv("DB_NAME", "url_shortener"),
		DBSSLMode:              getEnv("DB_SSLMODE", "disable"),
		RedisHost:              getEnv("REDIS_HOST", "localhost"),
		RedisPort:              getEnv("REDIS_PORT", "6379"),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		RedisDB:                getEnvAsInt("REDIS_DB", 0),
		JWTSecret:              getEnv("JWT_SECRET", "super-secret-key-change-this-in-production-123456"),
		JWTExpirationHours:     time.Duration(jwtHours) * time.Hour,
		CacheDefaultTTL:        time.Duration(cacheTTL) * time.Second,
		RateLimitAnonymous:     getEnvAsInt("RATE_LIMIT_ANONYMOUS", 30),
		RateLimitAuthenticated: getEnvAsInt("RATE_LIMIT_AUTHENTICATED", 120),
		RateLimitWindow:        time.Duration(rateWindow) * time.Second,
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}

	return val
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}
