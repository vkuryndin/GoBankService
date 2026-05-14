package services

import "time"

const (
	minimumUsernameLength = 3
	minimumPasswordLength = 8

	maxDescriptionLength        = 500
	maxPredictionDays           = 365
	maxOperationStatisticsLimit = 500
	maxCreditTermMonths         = 120
	maxCardCreationAttempts     = 3

	// Five minutes is a trade-off: short enough to limit reuse, long enough to copy from email.
	mfaCodeLifetime = 5 * time.Minute
)
