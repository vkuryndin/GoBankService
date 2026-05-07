package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"bank-service/internal/audit"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, event audit.Event) error {
	details := event.Details
	if details == nil {
		details = map[string]any{}
	}

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}

	query := `
		INSERT INTO audit_logs (
			user_id,
			action,
			resource_type,
			resource_id,
			status,
			ip_address,
			user_agent,
			details
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		nullInt64(event.UserID),
		event.Action,
		nullString(event.ResourceType),
		nullInt64(event.ResourceID),
		event.Status,
		nullString(event.IPAddress),
		nullString(event.UserAgent),
		string(detailsJSON),
	)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

func nullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: *value, Valid: true}
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}

	return sql.NullString{String: value, Valid: true}
}
