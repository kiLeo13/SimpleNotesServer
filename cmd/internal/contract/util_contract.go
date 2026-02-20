package contract

type CompanyResponse struct {
	CNPJ        string             `json:"cnpj"`
	LegalName   string             `json:"legal_name"`
	TradeName   string             `json:"trade_name"`
	LegalNature string             `json:"legal_nature"`
	RegStatus   string             `json:"registration_status"`
	Partners    []*PartnerResponse `json:"qsa"`
}

type PartnerResponse struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	RoleCode int    `json:"role_code"`
	AgeRange string `json:"age_range"`
}
