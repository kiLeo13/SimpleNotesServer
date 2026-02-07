package events

import "simplenotes/cmd/internal/contract"

type EventType string

const (
	TypePing EventType = "ping"

	TypeConnectionKill EventType = "CONNECTION_KILL"
	TypeSessionExpired EventType = "SESSION_EXPIRED"
	TypeAck            EventType = "ACK"

	TypeNoteCreated EventType = "NOTE_CREATED"
	TypeNoteUpdated EventType = "NOTE_UPDATED"
	TypeNoteDeleted EventType = "NOTE_DELETED"

	TypeUserUpdated EventType = "USER_UPDATED"
)

type Wrapper struct {
	Type EventType   `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

type SocketEvent interface {
	GetType() EventType
}

type Ack struct{}

func (Ack) GetType() EventType {
	return TypeAck
}

type ConnectionKill struct {
	Reason *string `json:"reason,omitempty"`
}

func (e *ConnectionKill) GetType() EventType {
	return TypeConnectionKill
}

type NoteCreated struct {
	*contract.NoteResponse
}

func (e *NoteCreated) GetType() EventType {
	return TypeNoteCreated
}

type NoteUpdated struct {
	*contract.NoteResponse
}

func (e *NoteUpdated) GetType() EventType {
	return TypeNoteUpdated
}

type NoteDeleted struct {
	NoteID int `json:"id"`
}

func (e *NoteDeleted) GetType() EventType {
	return TypeNoteDeleted
}

type UserUpdated struct {
	*contract.UserResponse
}

func (e *UserUpdated) GetType() EventType {
	return TypeUserUpdated
}
