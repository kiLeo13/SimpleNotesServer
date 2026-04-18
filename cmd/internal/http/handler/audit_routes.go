package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"simplenotes/cmd/internal/contract"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
)

type AuditService interface {
	GetAuditLogs(actor *entity.User, req *contract.AuditLogListRequest) (*contract.AuditLogListResponse, apierror.ErrorResponse)
}

type DefaultAuditRoute struct {
	AuditService AuditService
}

func NewAuditDefault(auditService AuditService) *DefaultAuditRoute {
	return &DefaultAuditRoute{AuditService: auditService}
}

func (a *DefaultAuditRoute) GetAuditLogs(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	req, apierr := bindAuditLogRequest(c)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}

	resp, err := a.AuditService.GetAuditLogs(user, req)
	if err != nil {
		return c.JSON(err.Code(), err)
	}
	return c.JSON(http.StatusOK, resp)
}

func bindAuditLogRequest(c echo.Context) (*contract.AuditLogListRequest, apierror.ErrorResponse) {
	req := &contract.AuditLogListRequest{}

	if rawLimit := strings.TrimSpace(c.QueryParam("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return nil, apierror.NewInvalidParamTypeError("limit", "int")
		}
		req.Limit = limit
	}

	if rawBeforeID := strings.TrimSpace(c.QueryParam("before_id")); rawBeforeID != "" {
		beforeID, err := strconv.ParseInt(rawBeforeID, 10, 64)
		if err != nil {
			return nil, apierror.NewInvalidParamTypeError("before_id", "int64")
		}
		req.BeforeID = &beforeID
	}

	if rawActorID := strings.TrimSpace(c.QueryParam("actor_user_id")); rawActorID != "" {
		actorID, err := strconv.Atoi(rawActorID)
		if err != nil {
			return nil, apierror.NewInvalidParamTypeError("actor_user_id", "int")
		}
		req.ActorUserID = &actorID
	}

	if rawSubjectType := strings.TrimSpace(c.QueryParam("subject_type")); rawSubjectType != "" {
		req.SubjectType = &rawSubjectType
	}

	if rawSubjectID := strings.TrimSpace(c.QueryParam("subject_id")); rawSubjectID != "" {
		req.SubjectID = &rawSubjectID
	}

	if rawActionType := strings.TrimSpace(c.QueryParam("action_type")); rawActionType != "" {
		req.ActionType = &rawActionType
	}

	return req, nil
}
