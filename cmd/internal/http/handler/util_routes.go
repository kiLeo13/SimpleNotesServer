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

type UtilService interface {
	GetCompanyByCNPJ(actor *entity.User, cnpj string) (*contract.CompanyResponse, apierror.ErrorResponse)
}

type DefaultUtilRoute struct {
	UtilService UtilService
}

func NewUtilRoute(utilService UtilService) *DefaultUtilRoute {
	return &DefaultUtilRoute{UtilService: utilService}
}

func (u *DefaultUtilRoute) GetCompany(c echo.Context) error {
	user, cerr := utils.GetUserFromContext(c)
	if cerr != nil {
		return c.JSON(cerr.Code(), cerr)
	}

	cnpj := strings.TrimSpace(c.Param("cnpj"))
	if !utils.IsCNPJValid(cnpj) {
		apierr := apierror.InvalidCNPJError
		return c.JSON(apierr.Code(), apierr)
	}

	company, apierr := u.UtilService.GetCompanyByCNPJ(user, cnpj)
	if apierr != nil {
		return c.JSON(apierr.Code(), apierr)
	}
	return c.JSON(http.StatusOK, company)
}
