package entity

// Permission is a custom type for bitwise flags
type Permission int64

const (
	// PermissionAdministrator grants god-mode.
	// Admins are immune to all restrictions and cannot be modified via API.
	PermissionAdministrator Permission = 1 << iota

	// PermissionCreateNotes allows creating new notes.
	PermissionCreateNotes

	// PermissionEditNotes allows modifying owned or shared notes.
	PermissionEditNotes

	// PermissionDeleteNotes allows permanently removing notes.
	PermissionDeleteNotes

	// PermissionSeeHiddenNotes allows access to private/hidden notes.
	PermissionSeeHiddenNotes

	// PermissionManageUsers allows modifying mutable fields of other users.
	// It does NOT grant the ability to change permissions or delete users.
	PermissionManageUsers

	// PermissionDeleteUsers allows deleting user accounts.
	// Administrators are immune to this action.
	PermissionDeleteUsers

	// PermissionManagePerms allows granting/revoking permissions for others.
	//
	// Constraints:
	//
	// 1. Cannot modify Administrators.
	//
	// 2. Cannot revoke PermissionManageUsers from anyone.
	//
	// 3. CAN revoke PermissionManagePerms from others.
	PermissionManagePerms

	// PermissionPunishUsers allows suspending/banning accounts.
	// Immunity: Administrators and users with PermissionManagePerms cannot be punished.
	PermissionPunishUsers

	// PermissionPerformLookup allows users to call endpoints outside the
	// general platform scope. Like CNPJ/places/IP lookups.
	PermissionPerformLookup
)

// Has checks if the permission bitmask contains ALL bits
// requested in 'target'. It ignores Administrator status.
// Logic: (p & target) == target
func (p Permission) Has(target Permission) bool {
	return (p & target) == target
}

// HasAny returns true if the user has ANY of the target permissions
func (p Permission) HasAny(target Permission) bool {
	return (p & target) > 0
}

// Add appends a permission to the bitmask
func (p Permission) Add(perm Permission) Permission {
	return p | perm
}

// Remove clears a permission from the bitmask
func (p Permission) Remove(perm Permission) Permission {
	return p &^ perm
}

// HasEffective checks if the permission bitmask contains the target bits
// OR if the permission includes Administrator
func (p Permission) HasEffective(target Permission) bool {
	return p.Has(PermissionAdministrator) || p.Has(target)
}
