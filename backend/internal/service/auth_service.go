package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/models"
	"url-shortener/internal/repository/postgres"
	"url-shortener/internal/utils"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidUsername = errors.New("invalid username")
	ErrPasswordTooWeak = errors.New("password must be at least 8 characters")
	ErrInvalidCreds    = errors.New("invalid username or password")
	ErrInvalidToken    = errors.New("invalid or expired token")
)

type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	jwt.RegisteredClaims
}

type AuthService struct {
	userRepo *postgres.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *postgres.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func IsValidUsername(username string) bool {
	// Rules:
	// 1. Must start with a letter.
	// 2. Can contain letters, numbers, and underscores.
	// 3. Length between 3 and 30 characters.
	const pattern = `^[a-zA-Z][a-zA-Z0-9_]{2,29}$`

	matched, err := regexp.MatchString(pattern, username)
	if err != nil {
		return false
	}
	return matched
}

func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	claims := JWTClaims{
		UserID:   uuid.UUID(user.ID),
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTExpirationHours)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	// type assertion
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) Register(ctx context.Context, username, password string) (*models.AuthResponse, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	if !IsValidUsername(username) {
		return nil, ErrInvalidUsername
	}

	if len(password) < 8 {
		return nil, ErrPasswordTooWeak
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username:     username,
		PasswordHash: hash,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*models.AuthResponse, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, postgres.ErrUserNotFound) {
			return nil, ErrInvalidCreds
		}
		return nil, err
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidCreds
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}
