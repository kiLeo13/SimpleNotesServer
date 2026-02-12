package events

import "simplenotes/cmd/internal/contract"

type SocketEvent interface {
	GetType() contract.EventType
}

type Ack struct{}

func (*Ack) GetType() contract.EventType {
	return contract.EventAck
}

type ConnectionKill struct {
	Code   contract.KillCode `json:"code"`
	Reason *string           `json:"reason,omitempty"`
}

func (e *ConnectionKill) GetType() contract.EventType {
	return contract.EventConnectionKill
}

type NoteCreated struct {
	*contract.NoteResponse
}

func (e *NoteCreated) GetType() contract.EventType {
	return contract.EventNoteCreated
}

type NoteUpdated struct {
	*contract.NoteResponse
}

func (e *NoteUpdated) GetType() contract.EventType {
	return contract.EventNoteUpdated
}

type NoteDeleted struct {
	NoteID int `json:"id"`
}

func (e *NoteDeleted) GetType() contract.EventType {
	return contract.EventNoteDeleted
}

type UserUpdated struct {
	*contract.UserResponse
}

func (e *UserUpdated) GetType() contract.EventType {
	return contract.EventUserUpdated
}
