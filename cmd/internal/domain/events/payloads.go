package events

type DeleteNoteEvent struct {
	NoteID int `json:"id"`
}

func (e DeleteNoteEvent) GetType() EventType {
	return EventNoteDelete
}
