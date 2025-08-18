package entity

// NoteAlias is meant to be immutable
type NoteAlias struct {
	ID        int    `gorm:"primaryKey"`
	NoteID    int    `gorm:"not null"`
	Value     string `gorm:"not null"`
	CreatedAt int64  `gorm:"not null"`
}
