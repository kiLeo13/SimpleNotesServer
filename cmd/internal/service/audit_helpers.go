package service

import (
	"encoding/json"
	"simplenotes/cmd/internal/domain/entity"
	"strconv"
)

func auditValuePtr(value string) *string {
	return &value
}

func auditJSONString(values []string) string {
	payload, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func newAuditCreateValue(field string, valueType entity.AuditValueType, value string) *entity.AuditLogChange {
	return &entity.AuditLogChange{
		FieldName: field,
		NewValue:  auditValuePtr(value),
		ValueType: valueType,
	}
}

func newAuditDeleteValue(field string, valueType entity.AuditValueType, value string) *entity.AuditLogChange {
	return &entity.AuditLogChange{
		FieldName: field,
		OldValue:  auditValuePtr(value),
		ValueType: valueType,
	}
}

func appendAuditStringChange(changes *[]*entity.AuditLogChange, field, oldValue, newValue string) {
	if oldValue == newValue {
		return
	}
	*changes = append(*changes, &entity.AuditLogChange{
		FieldName: field,
		OldValue:  auditValuePtr(oldValue),
		NewValue:  auditValuePtr(newValue),
		ValueType: entity.AuditValueTypeString,
	})
}

func appendAuditEnumChange(changes *[]*entity.AuditLogChange, field, oldValue, newValue string) {
	if oldValue == newValue {
		return
	}
	*changes = append(*changes, &entity.AuditLogChange{
		FieldName: field,
		OldValue:  auditValuePtr(oldValue),
		NewValue:  auditValuePtr(newValue),
		ValueType: entity.AuditValueTypeEnum,
	})
}

func appendAuditBoolChange(changes *[]*entity.AuditLogChange, field string, oldValue, newValue bool) {
	if oldValue == newValue {
		return
	}
	*changes = append(*changes, &entity.AuditLogChange{
		FieldName: field,
		OldValue:  auditValuePtr(strconv.FormatBool(oldValue)),
		NewValue:  auditValuePtr(strconv.FormatBool(newValue)),
		ValueType: entity.AuditValueTypeBool,
	})
}

func appendAuditIntChange(changes *[]*entity.AuditLogChange, field string, oldValue, newValue int64) {
	if oldValue == newValue {
		return
	}
	*changes = append(*changes, &entity.AuditLogChange{
		FieldName: field,
		OldValue:  auditValuePtr(strconv.FormatInt(oldValue, 10)),
		NewValue:  auditValuePtr(strconv.FormatInt(newValue, 10)),
		ValueType: entity.AuditValueTypeInt,
	})
}

func appendAuditStringArrayChange(changes *[]*entity.AuditLogChange, field string, oldValue, newValue []string) {
	oldJSON := auditJSONString(oldValue)
	newJSON := auditJSONString(newValue)
	if oldJSON == newJSON {
		return
	}
	*changes = append(*changes, &entity.AuditLogChange{
		FieldName: field,
		OldValue:  auditValuePtr(oldJSON),
		NewValue:  auditValuePtr(newJSON),
		ValueType: entity.AuditValueTypeStringArray,
	})
}
