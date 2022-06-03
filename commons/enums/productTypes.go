package enums

type ProductType string

const (
	ClaimsReadyResidential = "ClaimsReadyResidential"
	ECPremiumResidential   = "ECPremiumResidential"
	PremiumResidential     = "PremiumResidential"
)

func ProductTypeList() []string {
	return []string{ClaimsReadyResidential, ECPremiumResidential, PremiumResidential}
}

func IsHipsterCompatible(productName string) bool {
	l := ProductTypeList()
	for _, v := range l {
		if v == productName {
			return true
		}
	}
	return false
}
