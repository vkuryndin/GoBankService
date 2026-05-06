package services

import "time"

const (
	minimumUsernameLength = 3
	minimumPasswordLength = 8

	maxDescriptionLength    = 500
	maxPredictionDays       = 365
	maxCreditTermMonths     = 120
	maxCardCreationAttempts = 3

	mfaCodeLifetime = 5 * time.Minute
)
