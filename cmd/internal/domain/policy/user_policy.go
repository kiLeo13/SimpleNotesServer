package policy

import (
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils/apierror"
)

const (
	admin       = entity.PermissionAdministrator
	mngPerms    = entity.PermissionManagePerms
	punishUsers = entity.PermissionPunishUsers
	mngUsers    = entity.PermissionManageUsers
)

// UserPolicy encapsulates all business rules for user manipulation.
// It returns apierror.ErrorResponse directly for seamless integration with handlers.
type UserPolicy struct{}

func NewUserPolicy() *UserPolicy {
	return &UserPolicy{}
}

// CanUpdateProfile checks if 'actor' can update mutable fields of 'target'
func (p *UserPolicy) CanUpdateProfile(actor, target *entity.User) apierror.ErrorResponse {
	if actor.ID == target.ID {
		return nil
	}

	if target.Permissions.Has(admin) {
		return forbiddenError("administrators cannot be modified")
	}

	if !actor.Permissions.HasEffective(mngUsers) {
		return permError(mngUsers)
	}
	return nil
}

// CanUpdatePermissions checks if 'actor' can change 'target' permissions to 'newPerms'
func (p *UserPolicy) CanUpdatePermissions(actor, target *entity.User, newPerms entity.Permission) apierror.ErrorResponse {
	// Rule 1: Actor must have ManagePerms
	if !actor.Permissions.HasEffective(mngPerms) {
		return permError(mngPerms)
	}

	// Rule 2: Admin Immunity
	if target.Permissions.Has(admin) {
		return forbiddenError("administrators cannot be modified")
	}

	// Rule 3: Cannot grant Admin via API
	if newPerms.Has(admin) {
		return forbiddenError("cannot grant administrator privileges via API")
	}

	// Rule 4: Sticky Permission Constraint
	// Users with this permission cannot remove PermissionManageUsers of other users
	wasManager := target.Permissions.Has(mngUsers)
	isManager := newPerms.Has(mngUsers)

	if wasManager && !isManager {
		return forbiddenError("cannot revoke 'Manage Users' capability from an existing manager")
	}
	return nil
}

// CanPunishUser checks if 'actor' can suspend/ban 'target'.
func (p *UserPolicy) CanPunishUser(actor, target *entity.User) apierror.ErrorResponse {
	if !actor.Permissions.HasEffective(punishUsers) {
		return permError(punishUsers)
	}

	// Rule 2: Immunity check
	// Users with Admin and PermissionManagePerms are immune
	if target.Permissions.Has(admin) ||
		target.Permissions.Has(mngPerms) {
		return forbiddenError("target user is immune to punishment actions")
	}
	return nil
}

func permError(perm entity.Permission) *apierror.APIError {
	return apierror.NewPermissionError(int64(perm))
}

func forbiddenError(msg string) *apierror.APIError {
	return apierror.NewForbiddenError(msg)
}
