package jobs

import (
	"context"
	"github.com/labstack/gommon/log"
	"simplenotes/cmd/internal/domain/events"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/utils"
	"time"
)

type ConnectionCleaner struct {
	wsService *service.WebSocketService
}

func NewConnectionCleaner(wsService *service.WebSocketService) *ConnectionCleaner {
	return &ConnectionCleaner{wsService: wsService}
}

func (c *ConnectionCleaner) Start(ctx context.Context) {
	// Poll every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Info("Connection cleaner cron started")

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping connection cleaner...")
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

func (c *ConnectionCleaner) cleanup() {
	now := utils.NowUTC()
	conns, err := c.wsService.ConnRepo.FindExpired(now)
	if err != nil {
		log.Errorf("Cleaner: failed to fetch expired connections: %v", err)
		return
	}

	if len(conns) == 0 {
		return
	}

	log.Infof("Cleaner: Found %d expired connections. Terminating...", len(conns))

	envelope := events.Wrapper{
		Type: events.TypeSessionExpired,
	}

	for _, conn := range conns {
		// Use a fresh context for network calls, detached from the ticker's timing
		bgCtx := context.Background()

		// Notify Client (So they know NOT to try reconnecting)
		_ = c.wsService.Gateway.PostToConnection(bgCtx, conn.ConnectionID, envelope)

		// Tell AWS we are dropping the connection
		_ = c.wsService.Gateway.DeleteConnection(bgCtx, conn.ConnectionID)

		// Remove from our DB
		_ = c.wsService.ConnRepo.Delete(conn.ConnectionID)
	}
}
