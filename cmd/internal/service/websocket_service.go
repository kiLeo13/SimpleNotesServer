package service

import (
	"context"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/domain/events"
	"simplenotes/cmd/internal/infrastructure/aws/websocket"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"time"

	"github.com/labstack/gommon/log"
)

type ConnectionRepository interface {
	Save(conn *entity.Connection) error
	Delete(connID string) error
	FindByUserID(userID int) ([]string, error)
	FindAll() ([]string, error)
}

type WebSocketService struct {
	ConnRepo ConnectionRepository
	Gateway  websocket.GatewayClient
}

func NewWebSocketService(repo ConnectionRepository, gateway websocket.GatewayClient) *WebSocketService {
	return &WebSocketService{
		ConnRepo: repo,
		Gateway:  gateway,
	}
}

func (s *WebSocketService) RegisterConnection(userID int, connectionID string) apierror.ErrorResponse {
	conn := &entity.Connection{
		ConnectionID: connectionID,
		UserID:       userID,
		CreatedAt:    utils.NowUTC(),
	}

	if err := s.ConnRepo.Save(conn); err != nil {
		log.Errorf("failed to save connection: %v", err)
		return apierror.InternalServerError
	}
	return nil
}

func (s *WebSocketService) RemoveConnection(connectionID string) {
	// We don't return error here because if it fails, it's not the client's fault
	_ = s.ConnRepo.Delete(connectionID)
}

func (s *WebSocketService) PushToUser(ctx context.Context, userID int, payload interface{}) {
	conns, err := s.ConnRepo.FindByUserID(userID)
	if err != nil {
		log.Errorf("failed to fetch connections for user %d: %v", userID, err)
		return
	}

	for _, connID := range conns {
		// We ignore errors here so one stale connection doesn't block others
		_ = s.Gateway.PostToConnection(ctx, connID, payload)
	}
}

// TerminateUserConnections sends a "poison pill" message and then disconnects
func (s *WebSocketService) TerminateUserConnections(ctx context.Context, userID int, reason string) {
	conns, _ := s.ConnRepo.FindByUserID(userID)

	msg := events.WebSocketEvent{
		Type: events.EventConnectionKill,
		Data: map[string]interface{}{
			"reason": reason,
		},
	}

	for _, connID := range conns {
		_ = s.Gateway.PostToConnection(ctx, connID, msg)

		go func(cid string) {
			time.Sleep(200 * time.Millisecond)
			_ = s.Gateway.DeleteConnection(context.Background(), cid)
			_ = s.ConnRepo.Delete(cid)
		}(connID)
	}
}

func (s *WebSocketService) Dispatch(ctx context.Context, userID int, evt events.SocketEvent) {
	envelope := &events.WebSocketEvent{
		Type: evt.GetType(),
		Data: evt,
	}
	s.PushToUser(ctx, userID, envelope)
}

// Broadcast sends an event to ALL connected users.
// This iterates through every active connection in the DB.
func (s *WebSocketService) Broadcast(ctx context.Context, evt events.SocketEvent) {
	conns, err := s.ConnRepo.FindAll()
	if err != nil {
		log.Errorf("failed to fetch all connections for broadcast: %v", err)
		return
	}

	envelope := &events.WebSocketEvent{
		Type: evt.GetType(),
		Data: evt,
	}

	for _, connID := range conns {
		_ = s.Gateway.PostToConnection(ctx, connID, envelope)
	}
}
