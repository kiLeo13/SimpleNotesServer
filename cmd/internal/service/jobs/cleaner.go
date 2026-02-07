package jobs

import (
	"context"
	"simplenotes/cmd/internal/contract"
	"simplenotes/cmd/internal/domain/entity"
	"time"

	"simplenotes/cmd/internal/domain/events"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/utils"

	"github.com/labstack/gommon/log"
)

type ConnectionCleaner struct {
	wsService *service.WebSocketService
}

func NewConnectionCleaner(wsService *service.WebSocketService) *ConnectionCleaner {
	return &ConnectionCleaner{wsService: wsService}
}

func (c *ConnectionCleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
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
	heartbeatCutoff := now - ((entity.HeartbeatPeriodMillis * 2) + entity.HeartbeatToleranceMillis)
	conns, err := c.wsService.ConnRepo.FindStale(now, heartbeatCutoff)
	if err != nil {
		log.Errorf("Cleaner: failed to fetch stale connections: %v", err)
		return
	}

	if len(conns) == 0 {
		return
	}

	log.Infof("Cleaner: Found %d stale connections. Terminating...", len(conns))

	for _, conn := range conns {
		var envelope contract.OutgoingSocketMessage

		if conn.ExpiresAt < now {
			envelope = contract.OutgoingSocketMessage{
				Type: contract.EventSessionExpired,
			}
		} else {
			envelope = contract.OutgoingSocketMessage{
				Type: contract.EventConnectionKill,
				Data: events.ConnectionKill{
					Code: contract.CodeIdleTimeout,
				},
			}
		}

		bgCtx := context.Background()
		_ = c.wsService.Gateway.PostToConnection(bgCtx, conn.ConnectionID, envelope)

		_ = c.wsService.Gateway.DeleteConnection(bgCtx, conn.ConnectionID)

		_ = c.wsService.ConnRepo.Delete(conn.ConnectionID)
	}
}
