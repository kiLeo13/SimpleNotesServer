package handler

import (
	"net/http"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/service"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strings"

	"github.com/labstack/echo/v4"
)

type UserService interface {
	GetUsers(requester *entity.User) ([]*service.UserResponse, apierror.ErrorResponse)
	GetUser(requester *entity.User, rawId string) (*service.UserResponse, apierror.ErrorResponse)
	UpdateUser(requester *entity.User, targetId string, req *service.UpdateUserRequest) (*service.UserResponse, apierror.ErrorResponse)
	CheckEmail(req *service.UserStatusRequest) (*service.EmailStatus, apierror.ErrorResponse)
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

func (u *DefaultUserRoute) GetUsers(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	users, apierr := u.UserService.GetUsers(user)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}

	resp := echo.Map{"users": users}
	return c.JSON(http.StatusOK, &resp)
}

func (u *DefaultUserRoute) GetUser(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	targetId := strings.TrimSpace(c.Param("id"))
	if targetId == "" {
		return c.JSON(http.StatusBadRequest, apierror.NewMissingParamError("id"))
	}

	resp, apierr := u.UserService.GetUser(user, targetId)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, resp)
}

func (u *DefaultUserRoute) UpdateUser(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	targetId := strings.TrimSpace(c.Param("id"))
	var req service.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	newUser, apierr := u.UserService.UpdateUser(user, targetId, &req)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, newUser)
}

func (u *DefaultUserRoute) CheckEmail(c echo.Context) error {
	var req service.UserStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apierror.MalformedBodyError)
	}

	status, err := u.UserService.CheckEmail(&req)
	if err != nil {
		return c.JSON(err.Code(), err)
	}

	resp := echo.Map{"status": status}
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
