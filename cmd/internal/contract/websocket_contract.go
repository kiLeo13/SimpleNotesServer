package contract

import "simplenotes/cmd/internal/domain/events"

// WebSocketMessage is used for messages we receive from the users.
type WebSocketMessage struct {
	Type events.EventType `json:"type"`
}
