package entity

// Permission is a custom type for bitwise flags
type Permission uint64

const (
	// PermissionAdministrator grants all permissions automatically.
	PermissionAdministrator Permission = 1 << iota

	// PermissionCreateNotes allows the user to create new notes.
	PermissionCreateNotes

	// PermissionEditNotes allows the user to modify the content or title
	// of existing notes they own or have access to.
	PermissionEditNotes

	// PermissionDeleteNotes allows the user to permanently remove notes.
	PermissionDeleteNotes

	// PermissionSeeHiddenNotes allows the user to access private/hidden notes.
	PermissionSeeHiddenNotes

	// PermissionManageUsers allows the user to modify other users' accounts,
	// such as banning users or changing their roles.
	PermissionManageUsers
)

// HasRaw checks if the permission bitmask contains ALL bits
// requested in 'target'. It ignores Administrator status.
// Logic: (p & target) == target
func (p Permission) HasRaw(target Permission) bool {
	return (p & target) == target
}

// HasAny returns true if the user has ANY of the target permissions
func (p Permission) HasAny(target Permission) bool {
	return (p & target) > 0
}

// HasEffective checks if the permission bitmask contains the target bits
// OR if the permission includes Administrator
func (p Permission) HasEffective(target Permission) bool {
	return p.HasRaw(PermissionAdministrator) || p.HasRaw(target)
}

// Add appends a permission to the bitmask
func (p Permission) Add(perm Permission) Permission {
	return p | perm
}

// Remove clears a permission from the bitmask
func (p Permission) Remove(perm Permission) Permission {
	return p &^ perm
}
