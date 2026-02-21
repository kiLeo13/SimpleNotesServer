package service

import (
	"context"
	"errors"
	"github.com/labstack/gommon/log"
	"simplenotes/cmd/internal/contract"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/infrastructure/minhareceita"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/apierror"
)

type CompanyRepository interface {
	Save(company *entity.Company) error
	FindByCNPJ(cnpj string) (*entity.Company, error)
}

type MiscService struct {
	ReceitaClient *minhareceita.Client
	CompanyRepo   CompanyRepository
}

func NewMiscService(client *minhareceita.Client, companyRepo CompanyRepository) *MiscService {
	return &MiscService{
		ReceitaClient: client,
		CompanyRepo:   companyRepo,
	}
}

func (u *MiscService) GetCompanyByCNPJ(actor *entity.User, cnpj string) (*contract.CompanyResponse, apierror.ErrorResponse) {
	if !actor.Permissions.HasEffective(entity.PermissionPerformLookup) {
		return nil, apierror.UserMissingPermsError
	}

	company, fromCache, err := u.findCompany(cnpj)
	if err != nil {
		return nil, err
	}
	return toCompanyResp(company, fromCache), nil
}

// findCompany is a utility function that will try to resolve the CNPJ into a company.
// It returns the company, a boolean (true = cached, false = API fetch) and a possible error response.
func (u *MiscService) findCompany(cnpj string) (*entity.Company, bool, apierror.ErrorResponse) {
	cached, err := u.CompanyRepo.FindByCNPJ(cnpj)
	if err != nil {
		log.Errorf("failed to find company by cnpj %s: %v", cnpj, err)
		return nil, false, apierror.InternalServerError
	}

	// If we have some kind of cache
	if cached != nil {
		if cached.Found {
			return cached, true, nil
		} else {
			return nil, false, apierror.NotFoundError
		}
	}

	// Cache miss
	apiCompany, apierr := u.fetchFromAPI(cnpj)
	if apierr != nil {
		return nil, false, apierr
	}

	err = u.CompanyRepo.Save(apiCompany)
	if err != nil {
		// We don't return a 500 here, since we have the data we need
		// and only the cache has failed. We can just log it and proceed.
		log.Errorf("failed to save company cache for CNPJ %s: %v", cnpj, err)
	}

	return apiCompany, false, nil
}

func (u *MiscService) fetchFromAPI(cnpj string) (*entity.Company, apierror.ErrorResponse) {
	company, err := u.ReceitaClient.GetByCNPJ(context.Background(), cnpj)
	if err != nil {
		if errors.Is(err, minhareceita.ErrNotFound) {
			u.cacheNegativeResult(cnpj)
			return nil, apierror.NotFoundError
		}
		log.Errorf("failed to fetch company by cnpj %s: %v", cnpj, err)
		return nil, apierror.InternalServerError
	}

	company.Found = true
	company.CachedAt = utils.NowUTC()
	return company, nil
}

func (u *MiscService) cacheNegativeResult(cnpj string) {
	emptyCompany := &entity.Company{
		CNPJ:  cnpj,
		Found: false,
	}
	_ = u.CompanyRepo.Save(emptyCompany)
}

func toCompanyResp(c *entity.Company, cached bool) *contract.CompanyResponse {
	return &contract.CompanyResponse{
		CNPJ:        c.CNPJ,
		LegalName:   c.LegalName,
		TradeName:   c.TradeName,
		LegalNature: c.LegalNature,
		RegStatus:   string(c.RegStatus),
		Partners:    toPartnersResponse(c.Partners),
		Cached:      cached,
	}
}

func toPartnersResponse(ps []*entity.CompanyPartner) []*contract.PartnerResponse {
	partners := make([]*contract.PartnerResponse, len(ps))
	for i, p := range ps {
		partners[i] = toPartnerResp(p)
	}
	return partners
}

func toPartnerResp(p *entity.CompanyPartner) *contract.PartnerResponse {
	return &contract.PartnerResponse{
		Name:     p.Name,
		Role:     p.Role,
		RoleCode: p.RoleCode,
		AgeRange: p.AgeRange,
	}
}
