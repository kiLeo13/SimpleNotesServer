package routes

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/utils/apierror"
	"strings"
)

type UserService interface {
	QueryUsers(req *service.QueryUsersRequest) ([]*service.UserResponse, apierror.ErrorResponse)
	GetUser(token, rawId string) (*service.UserResponse, apierror.ErrorResponse)
	ExistsEmail(req *service.UserExistsRequest) (bool, apierror.ErrorResponse)
	CreateUser(req *service.CreateUserRequest) apierror.ErrorResponse
	Login(req *service.UserLoginRequest) (*service.UserLoginResponse, apierror.ErrorResponse)
	ConfirmSignup(req *service.ConfirmSignupRequest) apierror.ErrorResponse
	ResendConfirmation(req *service.ResendConfirmRequest) apierror.ErrorResponse
}

type DefaultUserRoute struct {
	UserService UserService
}

func NewUserDefault(userService UserService) *DefaultUserRoute {
	return &DefaultUserRoute{UserService: userService}
}

func (u *DefaultUserRoute) QueryUsers(c echo.Context) error {
	var req service.QueryUsersRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	users, err := u.UserService.QueryUsers(&req)
	if err != nil {
		return c.JSON(err.Code(), err)
	}

	resp := echo.Map{"users": users}
	return c.JSON(http.StatusOK, &resp)
}

func (u *DefaultUserRoute) GetUser(c echo.Context) error {
	rawId := strings.TrimSpace(c.Param("id"))
	token := c.Request().Header.Get("Authorization")
	if rawId == "" {
		return c.JSON(http.StatusBadRequest, apierror.NewMissingParamError("id"))
	}

	user, apierr := u.UserService.GetUser(token, rawId)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, user)
}

func (u *DefaultUserRoute) EmailExists(c echo.Context) error {
	var req service.UserExistsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	exists, err := u.UserService.ExistsEmail(&req)
	if err != nil {
		return c.JSON(err.Code(), err)
	}

	resp := echo.Map{"exists": exists}
	return c.JSON(http.StatusOK, &resp)
}

func (u *DefaultUserRoute) CreateUser(c echo.Context) error {
	var req service.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	err := u.UserService.CreateUser(&req)
	if err != nil {
		return c.JSON(err.Code(), err)
	}
	return c.NoContent(http.StatusCreated)
}

func (u *DefaultUserRoute) CreateLogin(c echo.Context) error {
	var req service.UserLoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	resp, apierr := u.UserService.Login(&req)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, resp)
}

func (u *DefaultUserRoute) ConfirmSignup(c echo.Context) error {
	var req service.ConfirmSignupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	apierr := u.UserService.ConfirmSignup(&req)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.NoContent(http.StatusOK)
}

func (u *DefaultUserRoute) ResendConfirmation(c echo.Context) error {
	var req service.ResendConfirmRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	apierr := u.UserService.ResendConfirmation(&req)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.NoContent(http.StatusOK)
}
