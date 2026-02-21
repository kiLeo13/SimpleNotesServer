package handler

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"simplenotes/cmd/internal/contract"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
	"strings"
)

type MiscService interface {
	GetCompanyByCNPJ(actor *entity.User, cnpj string) (*contract.CompanyResponse, apierror.ErrorResponse)
}

type DefaultMiscRoute struct {
	MiscService MiscService
}

func NewMiscRoute(miscService MiscService) *DefaultMiscRoute {
	return &DefaultMiscRoute{MiscService: miscService}
}

func (u *DefaultMiscRoute) GetCompany(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	cnpj := strings.TrimSpace(c.Param("cnpj"))
	if !utils.IsCNPJValid(cnpj) {
		apierr := apierror.InvalidCNPJError
		return c.JSON(apierr.Code(), apierr)
	}

	company, apierr := u.MiscService.GetCompanyByCNPJ(user, cnpj)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, company)
}
