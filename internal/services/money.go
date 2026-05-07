package services

import (
	"math/big"
	"regexp"
	"strings"
)

var amountValidationRegexp = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

// normalizeAmount keeps money as a decimal string so financial values are not rounded by float arithmetic before PostgreSQL NUMERIC stores them.
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
