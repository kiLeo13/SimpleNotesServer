package service

import (
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/domain/policy"
	"simplenotes/cmd/internal/utils/apierror"
)

// userUpdater acts as a "Change Set" context.
// It accumulates errors and tracks if a save is actually needed.
type userUpdater struct {
	actor  *entity.User
	target *entity.User
	policy *policy.UserPolicy

	// State
	err   apierror.ErrorResponse
	dirty bool
}

// setProfileString handles standard string fields (Username, Bio, etc.)
func (u *userUpdater) setProfileString(newVal *string, targetField *string) {
	if u.err != nil || newVal == nil {
		return
	}

	if *newVal == *targetField {
		return
	}

	// Policy Check: Can we modify the profile?
	if err := u.policy.CanUpdateProfile(u.actor, u.target); err != nil {
		u.err = err
		return
	}

	*targetField = *newVal
	u.dirty = true
}

// setPermissions handles the complex logic of permission bitmasks
func (u *userUpdater) setPermissions(newVal *int64) {
	if u.err != nil || newVal == nil {
		return
	}

	newPerms := entity.Permission(*newVal)

	if u.target.Permissions == newPerms {
		return
	}

	// Policy Check
	if err := u.policy.CanUpdatePermissions(u.actor, u.target, newPerms); err != nil {
		u.err = err
		return
	}

	u.target.Permissions = newPerms
	u.dirty = true
}
