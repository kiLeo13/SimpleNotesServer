package contract

type CompanyResponse struct {
	CNPJ              string               `json:"cnpj"`
	LegalName         string               `json:"legal_name"`
	TradeName         string               `json:"trade_name"`
	LegalNature       string               `json:"legal_nature"`
	CompanySize       string               `json:"company_size"`
	BusinessStartDate string               `json:"business_start_date"`
	ShareCapital      int64                `json:"share_capital"`
	Registration      *CompanyRegistration `json:"registration"`
	Address           *CompanyAddress      `json:"address"`
	Partners          []*PartnerResponse   `json:"partners"`
	Cached            bool                 `json:"cached"`
}

type CompanyRegistration struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

type CompanyAddress struct {
	Type         string `json:"type"`
	StreetName   string `json:"street_name"`
	Number       string `json:"number"`
	Neighborhood string `json:"neighborhood"`
	ZipCode      string `json:"zip_code"`
	City         string `json:"city"`
}

type PartnerResponse struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	RoleCode int    `json:"role_code"`
	AgeRange string `json:"age_range"`
}
