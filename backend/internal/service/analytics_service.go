package service

import (
	"context"
	"log"
	"sync"
	"time"
	"url-shortener/internal/models"
	"url-shortener/internal/repository/postgres"

	"github.com/google/uuid"
)

type AnalyticsService struct {
	clickRepo *postgres.ClickRepository
	urlRepo   *postgres.URLRepository
	clickChan chan models.Click
	batchSize int
	flushFreq time.Duration
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewAnalyticsService(clickRepo *postgres.ClickRepository, urlRepo *postgres.URLRepository) *AnalyticsService {
	ctx, cancel := context.WithCancel(context.Background())

	s := &AnalyticsService{
		clickRepo: clickRepo,
		urlRepo:   urlRepo,
		clickChan: make(chan models.Click, 10000),
		batchSize: 50,
		flushFreq: 2 * time.Second,
		ctx:       ctx,
		cancel:    cancel,
	}

	s.wg.Add(1)
	// go s.worker()

	return s
}

func (s *AnalyticsService) Stop() {
	s.cancel()
	s.wg.Wait()
}

func (s *AnalyticsService) GetStats(ctx context.Context, urlID uuid.UUID) (*models.ClickStats, error) {
	return s.clickRepo.GetStatsByURLID(ctx, urlID)
}

func (s *AnalyticsService) worker() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.flushFreq)
	defer ticker.Stop()

	batch := make([]models.Click, 0, s.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.clickRepo.RecordBatch(ctx, batch); err != nil {
			log.Printf("Error flushing analytics batch: %v", err)
		}

		urlCounts := make(map[uuid.UUID]int)
		for _, c := range batch {
			urlCounts[uuid.UUID(c.URLID)]++
		}

		for uID, count := range urlCounts {
			_ = s.urlRepo.IncrementClickCount(ctx, uID, count)
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-s.ctx.Done():
			for {
				select {
				case c := <-s.clickChan:
					batch = append(batch, c)
					if len(batch) >= s.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case c := <-s.clickChan:
			batch = append(batch, c)
			if len(batch) >= s.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}

	}
}

// TrackClick queues a click event asynchronously. It never blocks redirection.
func (s *AnalyticsService) TrackClick(click models.Click) {
	select {
	case s.clickChan <- click:
	default:
		// Queue full, log and drop to avoid slowing down redirect latency
		log.Printf("Warning: analytics click channel full, dropping click for URL %s", click.URLID)
	}
}
