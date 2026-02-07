package handler

import (
	"net/http"
	"simplenotes/cmd/internal/infrastructure/aws/websocket"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"

	"github.com/labstack/echo/v4"
)

type WebSocketService interface {
	RegisterConnection(userID int, connID string, exp int64) apierror.ErrorResponse
	RemoveConnection(connectionID string)
}

type DefaultWSRoute struct {
	WSService WebSocketService
}

func NewWSDefault(wsService WebSocketService) *DefaultWSRoute {
	return &DefaultWSRoute{WSService: wsService}
}

func (h *DefaultWSRoute) HandleConnect(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	connID := c.Request().Header.Get(websocket.HeaderConnectionID)
	if connID == "" {
		return c.JSON(http.StatusBadRequest, apierror.NewMissingParamError("connectionId"))
	}

	token, err := utils.ParseTokenDataCtx(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, apierror.InvalidAuthTokenError)
	}

	if apierr := h.WSService.RegisterConnection(user.ID, connID, token.Exp); apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.NoContent(http.StatusOK)
}

func (h *DefaultWSRoute) HandleDisconnect(c echo.Context) error {
	connID := c.Request().Header.Get(websocket.HeaderConnectionID)
	if connID != "" {
		h.WSService.RemoveConnection(connID)
	}
	return c.NoContent(http.StatusOK)
}
