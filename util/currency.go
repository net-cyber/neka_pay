package util

const (
	USD = "USD"
	ETB = "ETB"
)


func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, ETB:
		return true
	}
	return false
}