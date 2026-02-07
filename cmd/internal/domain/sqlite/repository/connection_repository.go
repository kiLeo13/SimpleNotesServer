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

func (c *DefaultConnectionRepository) Delete(connID string) error {
	result := c.db.
		Where("connection_id = ?", connID).
		Delete(&entity.Connection{})

	return result.Error
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

func (c *DefaultConnectionRepository) FindStale(now int64, heartbeatThreshold int64) ([]*entity.Connection, error) {
	var conns []*entity.Connection
	err := c.db.Where("expires_at < ?", now).
		Or("last_heartbeat_at < ?", heartbeatThreshold).
		Find(&conns).Error

	return conns, err
}

func (c *DefaultConnectionRepository) UpdateHeartbeat(connID string, now int64) error {
	return c.db.Model(&entity.Connection{}).
		Where("connection_id = ?", connID).
		Update("last_heartbeat_at", now).Error
}
