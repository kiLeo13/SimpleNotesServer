package events

import "simplenotes/cmd/internal/service"

// CreateNoteEvent holds a whole note response body.
type CreateNoteEvent struct {
	*service.NoteResponse
}

func (e *CreateNoteEvent) GetType() EventType {
	return EventNoteCreate
}

// UpdateNoteEvent holds a whole note response body.
type UpdateNoteEvent struct {
	*service.NoteResponse
}

func (e *UpdateNoteEvent) GetType() EventType {
	return EventNoteUpdate
}

// DeleteNoteEvent holds only the note ID.
type DeleteNoteEvent struct {
	NoteID int `json:"id"`
}

func (e *DeleteNoteEvent) GetType() EventType {
	return EventNoteDelete
}
