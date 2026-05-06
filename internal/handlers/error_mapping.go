package handlers

import (
	"errors"
	"net/http"

	"bank-service/internal/services"
)

type errorRule struct {
	target     error
	statusCode int
	message    string
}

type errorRules []errorRule

func writeMappedError(w http.ResponseWriter, err error, rules errorRules, fallbackMessage string) {
	for _, rule := range rules {
		if errors.Is(err, rule.target) {
			writeError(w, rule.statusCode, rule.message)
			return
		}
	}

	writeError(w, http.StatusInternalServerError, fallbackMessage)
}

func joinErrorRules(groups ...errorRules) errorRules {
	var result errorRules
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

var accountErrorRules = errorRules{
	{target: services.ErrAccountNotFound, statusCode: http.StatusNotFound, message: "account not found"},
	{target: services.ErrAccountBlocked, statusCode: http.StatusForbidden, message: "account is blocked"},
	{target: services.ErrInsufficientFunds, statusCode: http.StatusConflict, message: "insufficient funds"},
}

var mfaErrorRules = errorRules{
	{target: services.ErrMFACodeRequired, statusCode: http.StatusBadRequest, message: "mfa code required"},
	{target: services.ErrInvalidMFACode, statusCode: http.StatusForbidden, message: "invalid mfa code"},
	{target: services.ErrInvalidMFAPurpose, statusCode: http.StatusBadRequest, message: "invalid mfa purpose"},
	{target: services.ErrInvalidMFAOperation, statusCode: http.StatusBadRequest, message: "invalid mfa operation"},
}

var notificationErrorRules = errorRules{
	{target: services.ErrNotificationsDisabled, statusCode: http.StatusServiceUnavailable, message: "smtp notifications disabled"},
	{target: services.ErrNotificationUserNotFound, statusCode: http.StatusNotFound, message: "user not found"},
}
