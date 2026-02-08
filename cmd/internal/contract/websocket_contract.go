package contract

type EventType string

const (
	EventPing EventType = "ping"

	EventConnectionKill EventType = "CONNECTION_KILL"
	EventSessionExpired EventType = "SESSION_EXPIRED"
	EventAck            EventType = "ACK"

	EventNoteCreated EventType = "NOTE_CREATED"
	EventNoteUpdated EventType = "NOTE_UPDATED"
	EventNoteDeleted EventType = "NOTE_DELETED"

	EventUserUpdated EventType = "USER_UPDATED"
)

type KillCode string

const (
	CodeSuspendedAccount KillCode = "SUSPENDED_ACCOUNT"
	CodeIdleTimeout      KillCode = "IDLE_TIMEOUT"
	CodeLogout           KillCode = "LOGOUT"
)

// IncomingSocketMessage is used for messages we receive from the users.
type IncomingSocketMessage struct {
	Type EventType `json:"type"`
}

// OutgoingSocketMessage is what we send to the Client
type OutgoingSocketMessage struct {
	Type EventType   `json:"type"`
	Data interface{} `json:"data,omitempty"`
}
