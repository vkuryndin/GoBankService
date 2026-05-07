package handlers

import (
	"errors"
	"net/http"

	"bank-service/internal/services"
)

type httpErrorMapping struct {
	target  error
	status  int
	message string
}

type errorRules []httpErrorMapping

func writeMappedError(w http.ResponseWriter, err error, rules errorRules, fallbackMessage string) {
	for _, rule := range rules {
		if errors.Is(err, rule.target) {
			writeError(w, rule.status, rule.message)
			return
		}
	}

	writeUnexpectedError(w, fallbackMessage)
}

func writeUnexpectedError(w http.ResponseWriter, message string) {
	writeError(w, http.StatusInternalServerError, message)
}

func joinErrorRules(groups ...errorRules) errorRules {
	var result errorRules
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

var accountErrorRules = errorRules{
	{target: services.ErrAccountNotFound, status: http.StatusNotFound, message: "account not found"},
	{target: services.ErrAccountBlocked, status: http.StatusForbidden, message: "account is blocked"},
	{target: services.ErrInsufficientFunds, status: http.StatusConflict, message: "insufficient funds"},
}

var mfaErrorRules = errorRules{
	{target: services.ErrMFACodeRequired, status: http.StatusBadRequest, message: "mfa code required"},
	{target: services.ErrInvalidMFACode, status: http.StatusForbidden, message: "invalid mfa code"},
	{target: services.ErrInvalidMFAPurpose, status: http.StatusBadRequest, message: "invalid mfa purpose"},
	{target: services.ErrInvalidMFAOperation, status: http.StatusBadRequest, message: "invalid mfa operation"},
}

var cardErrorRules = errorRules{
	{target: services.ErrCardNotFound, status: http.StatusNotFound, message: "card not found"},
	{target: services.ErrCardClosed, status: http.StatusConflict, message: "card is closed"},
	{target: services.ErrCardAlreadyClosed, status: http.StatusConflict, message: "card already closed"},
	{target: services.ErrInvalidCardData, status: http.StatusBadRequest, message: "invalid card data"},
}

var creditPolicyErrorRules = errorRules{
	{target: services.ErrActiveOverdueCreditExists, status: http.StatusConflict, message: "active overdue credit exists"},
	{target: services.ErrActiveCreditLimitExceeded, status: http.StatusConflict, message: "active credit limit exceeded"},
	{target: services.ErrCreditPrincipalLimitExceeded, status: http.StatusBadRequest, message: "credit principal limit exceeded"},
	{target: services.ErrCreditTotalPrincipalLimitExceeded, status: http.StatusConflict, message: "credit total principal limit exceeded"},
	{target: services.ErrCreditDebtLoadTooHigh, status: http.StatusConflict, message: "credit debt load too high"},
}

var notificationErrorRules = errorRules{
	{target: services.ErrNotificationsDisabled, status: http.StatusServiceUnavailable, message: "smtp notifications disabled"},
	{target: services.ErrNotificationUserNotFound, status: http.StatusNotFound, message: "user not found"},
}
