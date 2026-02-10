package policy

import (
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils/apierror"
)

const (
	seeHiddenNotes = entity.PermissionSeeHiddenNotes
	editNotes      = entity.PermissionEditNotes
	deleteNotes    = entity.PermissionDeleteNotes
)

// NotePolicy encapsulates all business rules for note manipulation.
// It returns apierror.ErrorResponse directly for seamless integration with handlers.
type NotePolicy struct{}

func NewNotePolicy() *NotePolicy {
	return &NotePolicy{}
}

func (p *NotePolicy) CanSee(note *entity.Note, actor *entity.User) apierror.ErrorResponse {
	if note == nil {
		return apierror.NotFoundError
	}

	if !actor.Permissions.HasEffective(seeHiddenNotes) &&
		note.Visibility == entity.VisibilityPrivate {
		return apierror.NotFoundError // ^^
	}
	return nil
}

func (p *NotePolicy) CanUpdate(note *entity.Note, actor *entity.User) apierror.ErrorResponse {
	if !actor.Permissions.HasEffective(editNotes) {
		return permError(editNotes)
	}
	return p.CanSee(note, actor)
}

func (p *NotePolicy) CanDelete(note *entity.Note, actor *entity.User) apierror.ErrorResponse {
	if !actor.Permissions.HasEffective(deleteNotes) {
		return permError(deleteNotes)
	}
	return p.CanSee(note, actor)
}
