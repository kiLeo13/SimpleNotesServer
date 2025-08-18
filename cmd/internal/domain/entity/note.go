package entity

type Note struct {
	ID        int    `gorm:"primaryKey"`
	Name      string `gorm:"not null"`
	Type      string `gorm:"not null"`
	Content   string `gorm:"not null"`
	CreatedAt int64  `gorm:"not null"`
	UpdatedAt int64  `gorm:"not null"`
}
