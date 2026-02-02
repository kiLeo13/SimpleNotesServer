package middleware

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
)

type UserRepository interface {
	FindActiveBySub(sub string) (*entity.User, error)
}

type AuthMiddlewareConfig struct {
	UserRepo UserRepository
}

// NewAuthMiddleware creates the handler with dependencies injected
func NewAuthMiddleware(cfg *AuthMiddlewareConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenData, err := utils.ParseTokenDataCtx(c)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, apierror.InvalidAuthTokenError)
			}

			user, err := cfg.UserRepo.FindActiveBySub(tokenData.Sub)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, apierror.InternalServerError)
			}

			if user == nil {
				// User deleted in DB but still has a valid token???
				return c.JSON(http.StatusUnauthorized, apierror.IDPUserNotFoundError)
			}

			if user.Suspended || !user.Active {
				return c.JSON(http.StatusForbidden, apierror.NewForbiddenError("Missing access"))
			}

			c.Set("user", user)
			c.Set("sub", tokenData.Sub)
			return next(c)
		}
	}
}
