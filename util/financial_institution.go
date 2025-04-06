package util

// Financial institution types
const (
	BankType   = "bank"
	WalletType = "wallet"
	MFIType    = "mfi"
)

// IsValidFinancialInstitutionType verifies if a type is valid
func IsValidFinancialInstitutionType(fiType string) bool {
	switch fiType {
	case BankType, WalletType, MFIType:
		return true
	}
	return false
}
