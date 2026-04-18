package contract

type AuditLogListRequest struct {
	Limit       int
	BeforeID    *int64
	ActorUserID *int
	SubjectType *string
	SubjectID   *string
	ActionType  *string
}

type AuditLogListResponse struct {
	Entries      []*AuditLogEventResponse `json:"entries"`
	NextBeforeID *string                  `json:"next_before_id,omitempty"`
}

type AuditLogEventResponse struct {
	ID          string                    `json:"id"`
	ActorUserID *int                      `json:"actor_user_id,omitempty"`
	ActionType  string                    `json:"action_type"`
	SubjectType string                    `json:"subject_type"`
	SubjectID   string                    `json:"subject_id"`
	Source      string                    `json:"source"`
	OccurredAt  string                    `json:"occurred_at"`
	Changes     []*AuditLogChangeResponse `json:"changes"`
}

type AuditLogChangeResponse struct {
	ID        int     `json:"id"`
	FieldName string  `json:"field_name"`
	OldValue  *string `json:"old_value,omitempty"`
	NewValue  *string `json:"new_value,omitempty"`
	ValueType string  `json:"value_type"`
}
