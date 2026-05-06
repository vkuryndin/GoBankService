package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func GenerateCardNumber() (string, error) {
	const prefix = "2200"
	const cardNumberLengthWithoutCheckDigit = 15

	number := prefix

	for len(number) < cardNumberLengthWithoutCheckDigit {
		digit, err := randomDigit()
		if err != nil {
			return "", err
		}

		number += digit
	}

	checkDigit, err := calculateLuhnCheckDigit(number)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%d", number, checkDigit), nil
}

func GenerateCVV() (string, error) {
	result := ""

	for i := 0; i < 3; i++ {
		digit, err := randomDigit()
		if err != nil {
			return "", err
		}

		result += digit
	}

	return result, nil
}

func GenerateExpiry() string {
	return time.Now().AddDate(3, 0, 0).Format("01/06")
}

func ComputeHMAC(data string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))

	return hex.EncodeToString(mac.Sum(nil))
}

func MaskCardNumber(number string) string {
	number = strings.TrimSpace(number)

	if len(number) < 8 {
		return number
	}

	return number[:4] + " **** **** " + number[len(number)-4:]
}

func calculateLuhnCheckDigit(number string) (int, error) {
	sum := 0

	for i := len(number) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(number[i]))
		if err != nil {
			return 0, err
		}

		positionFromRight := len(number) - i
		if positionFromRight%2 == 1 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	return (10 - sum%10) % 10, nil
}

func randomDigit() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10))
	if err != nil {
		return "", err
	}

	return n.String(), nil
}
