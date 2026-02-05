package events

type EventType string

const (
	EventConnectionKill EventType = "CONNECTION_KILL"

	EventNoteCreate EventType = "NOTE_CREATE"
	EventNoteUpdate EventType = "NOTE_UPDATE"
	EventNoteDelete EventType = "NOTE_DELETE"
)

type WebSocketEvent struct {
	Type   EventType   `json:"type"`
	Data   interface{} `json:"data,omitempty"`
	UserID int
}
