package services

import (
	"math/big"
	"regexp"
	"strings"
)

var amountValidationRegexp = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

// normalizeAmount keeps money values in the API format used by PostgreSQL numeric fields:
// positive RUB amounts with no more than two fractional digits.
func normalizeAmount(amount string) (string, error) {
	amount = strings.TrimSpace(amount)

	if !amountValidationRegexp.MatchString(amount) {
		return "", ErrInvalidAmount
	}

	value := new(big.Rat)
	if _, ok := value.SetString(amount); !ok {
		return "", ErrInvalidAmount
	}

	if value.Sign() <= 0 {
		return "", ErrInvalidAmount
	}

	return amount, nil
}
