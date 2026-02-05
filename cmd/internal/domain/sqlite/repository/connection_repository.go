package repository

import (
	"gorm.io/gorm"
	"simplenotes/cmd/internal/domain/entity"
)

type DefaultConnectionRepository struct {
	db *gorm.DB
}

func NewConnectionRepository(db *gorm.DB) *DefaultConnectionRepository {
	return &DefaultConnectionRepository{db: db}
}

func (c *DefaultConnectionRepository) Save(user *entity.Connection) error {
	return c.db.Save(user).Error
}

func (c *DefaultConnectionRepository) Delete(note *entity.Connection) error {
	return c.db.Delete(note).Error
}

func (c *DefaultConnectionRepository) FindByUserID(userID int) ([]string, error) {
	var ids []string
	result := c.db.Model(&entity.Connection{}).
		Where("user_id = ?", userID).
		Pluck("connection_id", &ids)

	if result.Error != nil {
		return nil, result.Error
	}

	return ids, nil
}

func (c *DefaultConnectionRepository) FindAll() ([]string, error) {
	var ids []string
	result := c.db.Model(&entity.Connection{}).Pluck("connection_id", &ids)
	return ids, result.Error
}
