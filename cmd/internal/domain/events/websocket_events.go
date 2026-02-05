package events

type EventType string

const (
	EventConnectionKill EventType = "CONNECTION_KILL"

	EventUserUpdate EventType = "USER_UPDATE"
	EventNoteUpdate EventType = "NOTE_UPDATE"
)

type WebSocketEvent struct {
	Type   EventType   `json:"type"`
	Data   interface{} `json:"data,omitempty"`
	UserID int
}
