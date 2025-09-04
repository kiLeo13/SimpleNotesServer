package routes

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"simplenotes/internal/service"
	"strconv"
)

type UserService interface {
	QueryUsers(req *service.QueryUsersRequest) ([]*service.UserResponse, *service.APIError)
	GetUser(id int) (*service.UserResponse, *service.APIError)
	CreateUser(req *service.CreateUserRequest) *service.APIError
	Login(req *service.UserLoginRequest) (*service.UserLoginResponse, *service.APIError)
	ConfirmSignup(req *service.ConfirmSignupRequest) *service.APIError
	ResendConfirmation(req *service.ResendConfirmRequest) *service.APIError
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
		return c.JSON(http.StatusBadRequest, service.MalformedJSONError)
	}

	users, err := u.UserService.QueryUsers(&req)
	if err != nil {
		return c.JSON(err.Status, err)
	}

	resp := echo.Map{"users": users}
	return c.JSON(http.StatusOK, &resp)
}

func (u *DefaultUserRoute) GetUser(c echo.Context) error {
	rawId := c.Request().PathValue("id")
	id, err := strconv.Atoi(rawId)
	if err != nil {
		apierr := service.NewInvalidParamTypeError("id", "int32")
		return c.JSON(apierr.Status, apierr)
	}

	user, apierr := u.UserService.GetUser(id)
	if apierr != nil {
		return c.JSON(apierr.Status, apierr)
	}
	return c.JSON(http.StatusOK, user)
}

func (u *DefaultUserRoute) CreateUser(c echo.Context) error {
	var req service.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, service.MalformedJSONError)
	}

	err := u.UserService.CreateUser(&req)
	if err != nil {
		return c.JSON(err.Status, err)
	}
	return c.NoContent(http.StatusCreated)
}

func (u *DefaultUserRoute) CreateLogin(c echo.Context) error {
	var req service.UserLoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, service.MalformedJSONError)
	}

	resp, apierr := u.UserService.Login(&req)
	if apierr != nil {
		return c.JSON(apierr.Status, apierr)
	}
	return c.JSON(http.StatusOK, resp)
}

func (u *DefaultUserRoute) ConfirmSignup(c echo.Context) error {
	var req service.ConfirmSignupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, service.MalformedJSONError)
	}

	apierr := u.UserService.ConfirmSignup(&req)
	if apierr != nil {
		return c.JSON(apierr.Status, apierr)
	}
	return c.NoContent(http.StatusOK)
}

func (u *DefaultUserRoute) ResendConfirmation(c echo.Context) error {
	var req service.ResendConfirmRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, service.MalformedJSONError)
	}

	apierr := u.UserService.ResendConfirmation(&req)
	if apierr != nil {
		return c.JSON(apierr.Status, apierr)
	}
	return c.NoContent(http.StatusOK)
}
