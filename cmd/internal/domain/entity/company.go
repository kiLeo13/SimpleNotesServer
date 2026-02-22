package entity

type RegStatus string

const (
	StatusActive    RegStatus = "ACTIVE"
	StatusClosed    RegStatus = "CLOSED"
	StatusSuspended RegStatus = "SUSPENDED"
	StatusUnfit     RegStatus = "UNFIT"
	StatusUnknown   RegStatus = "UNKNOWN"
)

type Company struct {
	CNPJ                string `gorm:"primaryKey;column:cnpj"`
	LegalName           string
	TradeName           string
	LegalNature         string
	CompanySize         string
	BusinessStartDate   string
	ShareCapital        int64
	RegStatus           RegStatus
	RegReason           string
	RegDate             string
	AddressType         string
	AddressStreetName   string
	AddressNumber       string
	AddressNeighborhood string
	AddressCity         string

	// Found controls the negative caching strategy for external API lookups:
	//
	// - true: The CNPJ is valid and the company data is cached.
	//
	// - false: The CNPJ was queried, returned a 404, and is safely cached as invalid.
	//
	// This prevents repeated API calls for CNPJs that we already know do not exist.
	Found    bool  `gorm:"default:true"`
	CachedAt int64 `gorm:"autoUpdateTime:false"`

	// Relationships
	Partners []*CompanyPartner `gorm:"foreignKey:CompanyCNPJ;references:CNPJ"`
}

type CompanyPartner struct {
	ID          int    `gorm:"primaryKey"`
	CompanyCNPJ string `gorm:"index"`
	Name        string
	Role        string
	RoleCode    int
	AgeRange    string
}
