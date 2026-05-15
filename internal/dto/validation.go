package dto

import (
	"errors"
	"strings"
)

var ErrInvalidRequest = errors.New("invalid request")

func (r RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" || strings.TrimSpace(r.Username) == "" || strings.TrimSpace(r.Password) == "" {
		return ErrInvalidRequest
	}
	return nil
}

func (r LoginRequest) Validate() error {
	if strings.TrimSpace(r.Login) == "" || strings.TrimSpace(r.Password) == "" {
		return ErrInvalidRequest
	}
	return nil
}

func (r DepositRequest) Validate() error {
	return requireNonBlank(r.Amount)
}

func (r WithdrawRequest) Validate() error {
	if err := requireNonBlank(r.Amount); err != nil {
		return err
	}
	return requireNonBlank(r.MFACode)
}

func (r TransferRequest) Validate() error {
	if r.FromAccountID <= 0 {
		return ErrInvalidRequest
	}

	if r.ToAccountID <= 0 && strings.TrimSpace(r.RecipientAccountNumber()) == "" {
		return ErrInvalidRequest
	}

	if r.ToAccountID > 0 && r.FromAccountID == r.ToAccountID {
		return ErrInvalidRequest
	}

	if err := requireNonBlank(r.Amount); err != nil {
		return err
	}
	return requireNonBlank(r.MFACode)
}

func (r CreateCardRequest) Validate() error {
	if r.AccountID <= 0 {
		return ErrInvalidRequest
	}
	return nil
}

func (r CardPaymentRequest) Validate() error {
	if err := requireNonBlank(r.Amount); err != nil {
		return err
	}
	if err := requireNonBlank(r.CVV); err != nil {
		return err
	}
	return requireNonBlank(r.MFACode)
}

func (r CardTransferRequest) Validate() error {
	if r.RecipientCardID() <= 0 && strings.TrimSpace(r.RecipientCardNumber()) == "" {
		return ErrInvalidRequest
	}
	if err := requireNonBlank(r.Amount); err != nil {
		return err
	}
	if err := requireNonBlank(r.CVV); err != nil {
		return err
	}
	return requireNonBlank(r.MFACode)
}

func (r CreateCreditRequest) Validate() error {
	if r.AccountID <= 0 || r.TermMonths <= 0 {
		return ErrInvalidRequest
	}
	if err := requireNonBlank(r.PrincipalAmount); err != nil {
		return err
	}
	return requireNonBlank(r.MFACode)
}

func (r CheckCreditRequest) Validate() error {
	if r.AccountID <= 0 || r.TermMonths <= 0 {
		return ErrInvalidRequest
	}
	return requireNonBlank(r.PrincipalAmount)
}

func (r CreditPrepaymentRequest) Validate() error {
	if err := requireNonBlank(r.Amount); err != nil {
		return err
	}
	if strings.TrimSpace(r.Mode) == "" {
		return ErrInvalidRequest
	}
	return requireNonBlank(r.MFACode)
}

func (r MFARequest) Validate() error {
	if strings.TrimSpace(r.Purpose) == "" {
		return ErrInvalidRequest
	}
	return nil
}

func requireNonBlank(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrInvalidRequest
	}
	return nil
}
