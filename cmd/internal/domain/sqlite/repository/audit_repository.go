package repository

import (
	"gorm.io/gorm"
	"simplenotes/cmd/internal/domain/entity"
)

type AuditLogFilter struct {
	Limit       int
	BeforeID    *int64
	ActorUserID *int
	SubjectType *entity.AuditSubjectType
	SubjectID   *string
	ActionType  *entity.AuditActionType
}

type DefaultAuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *DefaultAuditRepository {
	return &DefaultAuditRepository{db: db}
}

func (d *DefaultAuditRepository) List(filter *AuditLogFilter) ([]*entity.AuditLogEvent, error) {
	var events []*entity.AuditLogEvent

	query := d.db.
		Preload("Changes", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Order("id DESC").
		Limit(filter.Limit)

	if filter.BeforeID != nil {
		query = query.Where("id < ?", *filter.BeforeID)
	}
	if filter.ActorUserID != nil {
		query = query.Where("actor_user_id = ?", *filter.ActorUserID)
	}
	if filter.SubjectType != nil {
		query = query.Where("subject_type = ?", *filter.SubjectType)
	}
	if filter.SubjectID != nil {
		query = query.Where("subject_id = ?", *filter.SubjectID)
	}
	if filter.ActionType != nil {
		query = query.Where("action_type = ?", *filter.ActionType)
	}

	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
