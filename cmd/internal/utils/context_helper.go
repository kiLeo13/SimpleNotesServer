package utils

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils/apierror"
)

func GetUserFromContext(c echo.Context) (*entity.User, apierror.ErrorResponse) {
	val := c.Get("user")
	if val == nil {
		log.Warnf("route %s attempted to read nil user from context", c.Request().URL)
		return nil, apierror.UnauthorizedError
	}

	user, ok := val.(*entity.User)
	if !ok {
		log.Warnf("expected user type at 'user' context key, got %v", user)
		return nil, apierror.InternalServerError
	}
	return user, nil
}
