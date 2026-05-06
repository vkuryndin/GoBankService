package services

import (
	"math/big"
	"regexp"
	"strings"
)

var amountValidationRegexp = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

// normalizeAmount keeps money as a decimal string to avoid float rounding before PostgreSQL NUMERIC handles it.
func normalizeAmount(amount string) (string, error) {
	amount = strings.TrimSpace(amount)
	if !amountValidationRegexp.MatchString(amount) {
		return "", ErrInvalidAmount
	}

	value := new(big.Rat)
	if _, ok := value.SetString(amount); !ok || value.Sign() <= 0 {
		return "", ErrInvalidAmount
	}

	return amount, nil
}
