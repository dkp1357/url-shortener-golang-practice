package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/models"
	"url-shortener/internal/repository/postgres"
	"url-shortener/internal/repository/redis"
	"url-shortener/internal/utils"

	"github.com/google/uuid"
)

var (
	ErrURLExpired        = errors.New("short URL has expired")
	ErrURLInactive       = errors.New("short URL is disabled")
	ErrInvalidExpiryDate = errors.New("expiry date must be in the future")
	ErrForbidden         = errors.New("you do not have permission to modify this URL")
)

type URLService struct {
	urlRepo   *postgres.URLRepository
	cacheRepo *redis.CacheRepository
	analytics *AnalyticsService
	cfg       *config.Config
}

func NewURLService(
	urlRepo *postgres.URLRepository,
	cacheRepo *redis.CacheRepository,
	analytics *AnalyticsService,
	cfg *config.Config,
) *URLService {
	return &URLService{
		urlRepo:   urlRepo,
		cacheRepo: cacheRepo,
		analytics: analytics,
		cfg:       cfg,
	}
}

func (s *URLService) toResponse(u *models.URL) models.URLResponse {
	identifier := u.ActiveIdentifier()
	shortURL := fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.BaseURL, "/"), identifier)

	return models.URLResponse{
		ID:          u.ID,
		OriginalURL: u.OriginalURL,
		ShortCode:   u.ShortCode,
		CustomAlias: u.CustomAlias,
		ShortURL:    shortURL,
		IsActive:    u.IsActive,
		ClickCount:  u.ClickCount,
		ExpiresAt:   u.ExpiresAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func (s *URLService) Create(ctx context.Context, req models.CreateURLRequest, userID *uuid.UUID) (*models.URLResponse, error) {
	if err := utils.ValidateTargetURL(req.OriginalURL); err != nil {
		return nil, err
	}

	// Validate expiry date if provided
	if req.ExpiresAt != nil {
		if req.ExpiresAt.Before(time.Now()) {
			return nil, ErrInvalidExpiryDate
		}
	} else if userID == nil {
		// Anonymous link default expiry: 7 days
		defaultExpiry := time.Now().Add(7 * 24 * time.Hour)
		req.ExpiresAt = &defaultExpiry
	}

	var customAlias *string
	if req.CustomAlias != "" {
		if err := utils.ValidateCustomAlias(req.CustomAlias); err != nil {
			return nil, err
		}
		ca := strings.TrimSpace(req.CustomAlias)
		customAlias = &ca
	}

	// Attempt generation with collision handling
	var urlRecord *models.URL
	var createErr error

	for range 5 {
		slug, err := utils.GenerateRandomSlug(7)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random slug: %w", err)
		}

		u := &models.URL{
			UserID:      userID,
			OriginalURL: req.OriginalURL,
			ShortCode:   slug,
			CustomAlias: customAlias,
			IsActive:    true,
			ExpiresAt:   req.ExpiresAt,
		}

		err = s.urlRepo.Create(ctx, u)
		if err == nil {
			urlRecord = u
			createErr = nil
			break
		}

		if errors.Is(err, postgres.ErrAliasAlreadyUsed) {
			if customAlias != nil {
				return nil, postgres.ErrAliasAlreadyUsed
			}
			// Random collision, retry
			continue
		}

		createErr = err
		break
	}

	if createErr != nil {
		return nil, createErr
	}
	if urlRecord == nil {
		return nil, errors.New("failed to generate unique short code after multiple attempts")
	}

	// Pre-populate Redis cache
	s.cacheURL(ctx, urlRecord)

	resp := s.toResponse(urlRecord)
	return &resp, nil
}

func (s *URLService) GetAndTrack(ctx context.Context, identifier, referrer, userAgent, ip string) (string, error) {
	// 1. Check Redis Cache
	cached, err := s.cacheRepo.GetURL(ctx, identifier)
	if err == nil && cached != nil {
		if !cached.IsActive {
			return "", ErrURLInactive
		}
		if cached.ExpiresAt != nil && time.Now().After(*cached.ExpiresAt) {
			_ = s.cacheRepo.DeleteURL(ctx, identifier)
			return "", ErrURLExpired
		}

		// Asynchronously dispatch click analytics
		s.analytics.TrackClick(models.Click{
			URLID:     cached.ID,
			Referrer:  referrer,
			UserAgent: userAgent,
			IPAddress: ip,
			CreatedAt: time.Now(),
		})

		return cached.OriginalURL, nil
	}

	// 2. Cache Miss: Query PostgreSQL
	urlRecord, err := s.urlRepo.GetByCodeOrAlias(ctx, identifier)
	if err != nil {
		return "", err
	}

	if !urlRecord.IsActive {
		return "", ErrURLInactive
	}

	if urlRecord.IsExpired() {
		return "", ErrURLExpired
	}

	// 3. Write back to Redis Cache
	s.cacheURL(ctx, urlRecord)

	// 4. Asynchronously dispatch click analytics
	s.analytics.TrackClick(models.Click{
		URLID:     urlRecord.ID,
		Referrer:  referrer,
		UserAgent: userAgent,
		IPAddress: ip,
		CreatedAt: time.Now(),
	})

	return urlRecord.OriginalURL, nil
}

func (s *URLService) GetDetails(ctx context.Context, identifier string) (*models.URLResponse, error) {
	urlRecord, err := s.urlRepo.GetByCodeOrAlias(ctx, identifier)
	if err != nil {
		return nil, err
	}

	resp := s.toResponse(urlRecord)
	return &resp, nil
}

func (s *URLService) Update(ctx context.Context, identifier string, userID uuid.UUID, req models.UpdateURLRequest) (*models.URLResponse, error) {
	urlRecord, err := s.urlRepo.GetByCodeOrAlias(ctx, identifier)
	if err != nil {
		return nil, err
	}

	if urlRecord.UserID == nil || *urlRecord.UserID != userID {
		return nil, ErrForbidden
	}

	if req.OriginalURL != nil {
		if err := utils.ValidateTargetURL(*req.OriginalURL); err != nil {
			return nil, err
		}
		urlRecord.OriginalURL = *req.OriginalURL
	}

	if req.IsActive != nil {
		urlRecord.IsActive = *req.IsActive
	}

	if req.ExpiresAt != nil {
		if req.ExpiresAt.Before(time.Now()) {
			return nil, ErrInvalidExpiryDate
		}
		urlRecord.ExpiresAt = req.ExpiresAt
	}

	if err := s.urlRepo.Update(ctx, urlRecord); err != nil {
		return nil, err
	}

	// Evict from Redis cache
	s.invalidateCache(ctx, urlRecord)

	resp := s.toResponse(urlRecord)
	return &resp, nil
}

func (s *URLService) Delete(ctx context.Context, identifier string, userID uuid.UUID) error {
	urlRecord, err := s.urlRepo.GetByCodeOrAlias(ctx, identifier)
	if err != nil {
		return err
	}

	if urlRecord.UserID == nil || *urlRecord.UserID != userID {
		return ErrForbidden
	}

	if err := s.urlRepo.Delete(ctx, urlRecord.ID, &userID); err != nil {
		return err
	}

	// Invalidate cache
	s.invalidateCache(ctx, urlRecord)

	return nil
}

func (s *URLService) ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]models.URLResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	records, totalCount, err := s.urlRepo.ListByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.URLResponse, len(records))
	for i, r := range records {
		responses[i] = s.toResponse(&r)
	}

	return responses, totalCount, nil
}

func (s *URLService) cacheURL(ctx context.Context, u *models.URL) {
	cached := &models.CachedURL{
		ID:          u.ID,
		OriginalURL: u.OriginalURL,
		IsActive:    u.IsActive,
		ExpiresAt:   u.ExpiresAt,
	}

	ttl := s.cfg.CacheDefaultTTL
	if u.ExpiresAt != nil {
		remaining := time.Until(*u.ExpiresAt)
		if remaining < ttl {
			ttl = remaining
		}
	}

	if ttl <= 0 {
		return
	}

	if err := s.cacheRepo.SetURL(ctx, u.ShortCode, cached, ttl); err != nil {
		log.Printf("Warning: failed to cache short code %s: %v", u.ShortCode, err)
	}

	if u.CustomAlias != nil && *u.CustomAlias != "" {
		if err := s.cacheRepo.SetURL(ctx, *u.CustomAlias, cached, ttl); err != nil {
			log.Printf("Warning: failed to cache custom alias %s: %v", *u.CustomAlias, err)
		}
	}
}

func (s *URLService) invalidateCache(ctx context.Context, u *models.URL) {
	_ = s.cacheRepo.DeleteURL(ctx, u.ShortCode)
	if u.CustomAlias != nil && *u.CustomAlias != "" {
		_ = s.cacheRepo.DeleteURL(ctx, *u.CustomAlias)
	}
}
