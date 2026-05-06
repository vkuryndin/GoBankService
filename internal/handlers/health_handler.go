package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HealthHandler struct {
	db *sql.DB
}

type HealthResponse struct {
	Status   string         `json:"status"`
	Database DatabaseStatus `json:"database"`
}

type DatabaseStatus struct {
	Connected bool   `json:"connected"`
	Name      string `json:"name,omitempty"`
	User      string `json:"user,omitempty"`
	Schema    string `json:"schema,omitempty"`
	Host      string `json:"host,omitempty"`
	Port      string `json:"port,omitempty"`
	Error     string `json:"error,omitempty"`
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dbStatus := h.getDatabaseStatus(ctx)

	statusCode := http.StatusOK
	status := "ok"

	if !dbStatus.Connected {
		statusCode = http.StatusServiceUnavailable
		status = "degraded"
	}

	response := HealthResponse{
		Status:   status,
		Database: dbStatus,
	}

	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *HealthHandler) getDatabaseStatus(ctx context.Context) DatabaseStatus {
	var (
		databaseName string
		databaseUser string
		schemaName   string
		host         sql.NullString
		port         sql.NullInt64
	)

	query := `
		SELECT
			current_database(),
			current_user,
			current_schema(),
			inet_server_addr()::text,
			inet_server_port()
	`

	err := h.db.QueryRowContext(ctx, query).Scan(
		&databaseName,
		&databaseUser,
		&schemaName,
		&host,
		&port,
	)
	if err != nil {
		return DatabaseStatus{
			Connected: false,
			Error:     "database unavailable",
		}
	}

	dbHost := ""
	if host.Valid {
		dbHost = host.String
	}

	dbPort := ""
	if port.Valid {
		dbPort = fmt.Sprintf("%d", port.Int64)
	}

	return DatabaseStatus{
		Connected: true,
		Name:      databaseName,
		User:      databaseUser,
		Schema:    schemaName,
		Host:      dbHost,
		Port:      dbPort,
	}
}
