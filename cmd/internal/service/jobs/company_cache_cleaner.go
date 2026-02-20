package jobs

import (
	"context"
	"time"

	"github.com/labstack/gommon/log"
	"simplenotes/cmd/internal/utils"
)

const (
	CacheTTLMillis = 10 * 60 * 60 * 1000
	CleanInterval  = 1 * time.Hour
)

type CompanyRepository interface {
	DeleteExpired(before int64) error
}

type CompanyCacheCleaner struct {
	companyRepo CompanyRepository
}

func NewCompanyCacheCleaner(repo CompanyRepository) *CompanyCacheCleaner {
	return &CompanyCacheCleaner{companyRepo: repo}
}

func (c *CompanyCacheCleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(CleanInterval)
	defer ticker.Stop()

	log.Info("Company cache cleaner cron started")

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping company cache cleaner...")
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

func (c *CompanyCacheCleaner) cleanup() {
	now := utils.NowUTC()
	cutoff := now - CacheTTLMillis

	err := c.companyRepo.DeleteExpired(cutoff)
	if err != nil {
		log.Errorf("Cleaner: failed to delete expired company cache: %v", err)
		return
	}

	log.Debugf("Cleaner: successfully swept company caches older than %d", cutoff)
}
