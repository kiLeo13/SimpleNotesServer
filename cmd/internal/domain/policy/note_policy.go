package policy

import (
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils/apierror"
)

const (
	seeHiddenNotes = entity.PermissionSeeHiddenNotes
)

// NotePolicy encapsulates all business rules for note manipulation.
// It returns apierror.ErrorResponse directly for seamless integration with handlers.
type NotePolicy struct{}

func NewNotePolicy() *NotePolicy {
	return &NotePolicy{}
}

func (p *NotePolicy) CanSee(note *entity.Note, actor *entity.User) apierror.ErrorResponse {
	if !actor.Permissions.HasEffective(seeHiddenNotes) &&
		note.Visibility == entity.VisibilityPrivate {
		return apierror.NotFoundError // ^^
	}
	return nil
}
