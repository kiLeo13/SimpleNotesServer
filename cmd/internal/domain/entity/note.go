package entity

type NoteType string

const (
	NoteTypeReference NoteType = "REFERENCE"
	NoteTypeMarkdown  NoteType = "MARKDOWN"
	NoteTypeFlowchart NoteType = "FLOWCHART"
)

type NoteVisibility string

const (
	VisibilityPublic  NoteVisibility = "PUBLIC"
	VisibilityOrg     NoteVisibility = "ORG"
	VisibilityPrivate NoteVisibility = "PRIVATE"
)

type Note struct {
	ID          int64          `gorm:"primaryKey"`
	Name        string         `gorm:"not null"`
	Content     string         `gorm:"not null"`
	CreatedByID int64          `gorm:"not null"` // References: users(id)
	Tags        string         `gorm:"not null"`
	NoteType    NoteType       `gorm:"not null"`
	ContentSize int            `gorm:"not null"`
	Visibility  NoteVisibility `gorm:"not null"`
	CreatedAt   int64          `gorm:"not null"`
	UpdatedAt   int64          `gorm:"not null;autoUpdateTime:false"`

	// Relations
	CreatedBy User `gorm:"foreignKey:CreatedByID;references:ID"`
}
