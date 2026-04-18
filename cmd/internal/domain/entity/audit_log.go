package entity

type AuditSource string

const (
	AuditSourceHTTPAPI AuditSource = "HTTP_API"
)

type AuditSubjectType string

const (
	AuditSubjectNote    AuditSubjectType = "NOTE"
	AuditSubjectUser    AuditSubjectType = "USER"
	AuditSubjectCompany AuditSubjectType = "COMPANY"
)

type AuditActionType string

const (
	AuditActionNoteCreate    AuditActionType = "NOTE_CREATE"
	AuditActionNoteUpdate    AuditActionType = "NOTE_UPDATE"
	AuditActionNoteDelete    AuditActionType = "NOTE_DELETE"
	AuditActionUserUpdate    AuditActionType = "USER_UPDATE"
	AuditActionUserSuspend   AuditActionType = "USER_SUSPEND"
	AuditActionUserUnsuspend AuditActionType = "USER_UNSUSPEND"
	AuditActionUserDelete    AuditActionType = "USER_DELETE"
	AuditActionCompanyLookup AuditActionType = "COMPANY_LOOKUP"
)

type AuditValueType string

const (
	AuditValueTypeString      AuditValueType = "STRING"
	AuditValueTypeInt         AuditValueType = "INT"
	AuditValueTypeBool        AuditValueType = "BOOL"
	AuditValueTypeEnum        AuditValueType = "ENUM"
	AuditValueTypeStringArray AuditValueType = "STRING_ARRAY"
)

type AuditLogEvent struct {
	ID          int64            `gorm:"primaryKey;autoIncrement:false;index:idx_audit_events_actor_id,priority:2;index:idx_audit_events_subject_id,priority:3;index:idx_audit_events_action_id,priority:2"`
	ActorUserID *int             `gorm:"index:idx_audit_events_actor_id,priority:1"`
	ActionType  AuditActionType  `gorm:"not null;index:idx_audit_events_action_id,priority:1"`
	SubjectType AuditSubjectType `gorm:"not null;index:idx_audit_events_subject_id,priority:1"`
	SubjectID   string           `gorm:"not null;index:idx_audit_events_subject_id,priority:2"`
	Source      AuditSource      `gorm:"not null"`
	OccurredAt  int64            `gorm:"not null"`

	ActorUser *User             `gorm:"foreignKey:ActorUserID;references:ID"`
	Changes   []*AuditLogChange `gorm:"foreignKey:EventID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type AuditLogChange struct {
	ID        int    `gorm:"primaryKey"`
	EventID   int64  `gorm:"not null;index:idx_audit_log_changes_event_position,priority:1"`
	Position  int    `gorm:"not null;index:idx_audit_log_changes_event_position,priority:2"`
	FieldName string `gorm:"not null"`
	OldValue  *string
	NewValue  *string
	ValueType AuditValueType `gorm:"not null"`
}
