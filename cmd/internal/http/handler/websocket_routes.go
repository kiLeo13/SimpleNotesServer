package handler

import (
	"net/http"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"

	"github.com/labstack/echo/v4"
)

type DefaultWSRoute struct {
	WSService *service.WebSocketService
}

func NewWSDefault(wsService *service.WebSocketService) *DefaultWSRoute {
	return &DefaultWSRoute{WSService: wsService}
}

func (h *DefaultWSRoute) HandleConnect(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	connID := c.Request().Header.Get("X-Connection-Id")
	if connID == "" {
		return c.JSON(http.StatusBadRequest, apierror.NewMissingParamError("connectionId"))
	}

	if err := h.WSService.RegisterConnection(user.ID, connID); err != nil {
		return c.JSON(err.Code(), err)
	}
	return c.NoContent(http.StatusOK)
}

func (h *DefaultWSRoute) HandleDisconnect(c echo.Context) error {
	connID := c.Request().Header.Get("X-Connection-Id")
	if connID != "" {
		h.WSService.RemoveConnection(connID)
	}
	return c.NoContent(http.StatusOK)
}
