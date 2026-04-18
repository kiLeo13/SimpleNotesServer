package service

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/sony/sonyflake/v2"
	"gorm.io/gorm"

	"simplenotes/cmd/internal/contract"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/domain/sqlite/repository"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
)

const (
	defaultAuditLogLimit = 50
	maxAuditLogLimit     = 100
)

type auditLogRepository interface {
	List(filter *repository.AuditLogFilter) ([]*entity.AuditLogEvent, error)
}

type auditIDGenerator interface {
	NextID() (int64, error)
}

type sonyflakeGenerator struct {
	flake *sonyflake.Sonyflake
}

func newSonyflakeGenerator() (auditIDGenerator, error) {
	flake, err := sonyflake.New(sonyflake.Settings{
		StartTime: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		MachineID: resolveAuditMachineID,
	})
	if err != nil {
		return nil, err
	}
	return &sonyflakeGenerator{flake: flake}, nil
}

func (s *sonyflakeGenerator) NextID() (int64, error) {
	return s.flake.NextID()
}

func resolveAuditMachineID() (int, error) {
	rawMachineID := strings.TrimSpace(os.Getenv("AUDIT_MACHINE_ID"))
	if rawMachineID != "" {
		machineID, err := strconv.ParseUint(rawMachineID, 10, 16)
		if err != nil {
			return 0, fmt.Errorf("parse AUDIT_MACHINE_ID: %w", err)
		}
		return int(machineID), nil
	}

	host, err := os.Hostname()
	if err == nil && host != "" {
		return int(crc32.ChecksumIEEE([]byte(host)) & 0xffff), nil
	}
	return os.Getpid() & 0xffff, nil
}

type AuditService struct {
	DB        *gorm.DB
	AuditRepo auditLogRepository
	IDGen     auditIDGenerator
}

func NewAuditService(db *gorm.DB, auditRepo auditLogRepository, idGen auditIDGenerator) (*AuditService, error) {
	if idGen == nil {
		var err error
		idGen, err = newSonyflakeGenerator()
		if err != nil {
			return nil, err
		}
	}

	return &AuditService{
		DB:        db,
		AuditRepo: auditRepo,
		IDGen:     idGen,
	}, nil
}

func (a *AuditService) Record(tx *gorm.DB, event *entity.AuditLogEvent) error {
	if event == nil {
		return errors.New("audit event is nil")
	}

	targetDB := a.DB
	if tx != nil {
		targetDB = tx
	}
	if targetDB == nil {
		return errors.New("audit database is nil")
	}

	id, err := a.IDGen.NextID()
	if err != nil {
		return err
	}

	event.ID = id
	if event.Source == "" {
		event.Source = entity.AuditSourceHTTPAPI
	}
	if event.OccurredAt == 0 {
		event.OccurredAt = utils.NowUTC()
	}

	if err := targetDB.Omit("Changes", "ActorUser").Create(event).Error; err != nil {
		return err
	}

	if len(event.Changes) == 0 {
		return nil
	}

	for i := range event.Changes {
		event.Changes[i].EventID = event.ID
		event.Changes[i].Position = i + 1
	}
	return targetDB.Create(&event.Changes).Error
}

func (a *AuditService) GetAuditLogs(actor *entity.User, req *contract.AuditLogListRequest) (*contract.AuditLogListResponse, apierror.ErrorResponse) {
	if !actor.Permissions.HasEffective(entity.PermissionManageUsers) {
		return nil, apierror.NewPermissionError(int64(entity.PermissionManageUsers))
	}

	filter, apierr := a.toAuditFilter(req)
	if apierr != nil {
		return nil, apierr
	}

	events, err := a.AuditRepo.List(filter)
	if err != nil {
		log.Errorf("failed to fetch audit logs: %v", err)
		return nil, apierror.InternalServerError
	}

	resp := &contract.AuditLogListResponse{
		Entries: make([]*contract.AuditLogEventResponse, len(events)),
	}

	for i, event := range events {
		resp.Entries[i] = toAuditEventResponse(event)
	}

	if len(events) == filter.Limit {
		next := strconv.FormatInt(events[len(events)-1].ID, 10)
		resp.NextBeforeID = &next
	}
	return resp, nil
}

func (a *AuditService) toAuditFilter(req *contract.AuditLogListRequest) (*repository.AuditLogFilter, apierror.ErrorResponse) {
	filter := &repository.AuditLogFilter{
		Limit: defaultAuditLogLimit,
	}

	if req != nil {
		if req.Limit != 0 {
			if req.Limit < 1 || req.Limit > maxAuditLogLimit {
				return nil, apierror.NewSimple(400, "Limit must be between 1 and %d", maxAuditLogLimit)
			}
			filter.Limit = req.Limit
		}
		filter.BeforeID = req.BeforeID
		filter.ActorUserID = req.ActorUserID

		if req.SubjectType != nil {
			subjectType := entity.AuditSubjectType(strings.TrimSpace(*req.SubjectType))
			if !isAuditSubjectTypeValid(subjectType) {
				return nil, apierror.NewSimple(400, "Invalid audit subject type")
			}
			filter.SubjectType = &subjectType
		}

		if req.SubjectID != nil {
			subjectID := strings.TrimSpace(*req.SubjectID)
			if subjectID == "" {
				return nil, apierror.NewSimple(400, "Subject ID cannot be empty")
			}
			filter.SubjectID = &subjectID
		}

		if req.ActionType != nil {
			actionType := entity.AuditActionType(strings.TrimSpace(*req.ActionType))
			if !isAuditActionTypeValid(actionType) {
				return nil, apierror.NewSimple(400, "Invalid audit action type")
			}
			filter.ActionType = &actionType
		}
	}

	return filter, nil
}

func isAuditSubjectTypeValid(subjectType entity.AuditSubjectType) bool {
	switch subjectType {
	case entity.AuditSubjectNote, entity.AuditSubjectUser, entity.AuditSubjectCompany:
		return true
	default:
		return false
	}
}

func isAuditActionTypeValid(actionType entity.AuditActionType) bool {
	switch actionType {
	case entity.AuditActionNoteCreate,
		entity.AuditActionNoteUpdate,
		entity.AuditActionNoteDelete,
		entity.AuditActionUserUpdate,
		entity.AuditActionUserSuspend,
		entity.AuditActionUserUnsuspend,
		entity.AuditActionUserDelete,
		entity.AuditActionCompanyLookup:
		return true
	default:
		return false
	}
}

func toAuditEventResponse(event *entity.AuditLogEvent) *contract.AuditLogEventResponse {
	resp := &contract.AuditLogEventResponse{
		ID:          strconv.FormatInt(event.ID, 10),
		ActorUserID: event.ActorUserID,
		ActionType:  string(event.ActionType),
		SubjectType: string(event.SubjectType),
		SubjectID:   event.SubjectID,
		Source:      string(event.Source),
		OccurredAt:  utils.FormatEpoch(event.OccurredAt),
		Changes:     make([]*contract.AuditLogChangeResponse, len(event.Changes)),
	}

	for i, change := range event.Changes {
		resp.Changes[i] = &contract.AuditLogChangeResponse{
			ID:        change.ID,
			FieldName: change.FieldName,
			OldValue:  change.OldValue,
			NewValue:  change.NewValue,
			ValueType: string(change.ValueType),
		}
	}
	return resp
}
