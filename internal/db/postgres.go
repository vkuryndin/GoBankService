package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Info struct {
	Name   string
	User   string
	Schema string
	Host   string
	Port   string
}

func Connect(databaseURL string) (*sql.DB, error) {
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database connection: %w", err)
	}

	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return database, nil
}

func GetInfo(ctx context.Context, database *sql.DB) (Info, error) {
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

	err := database.QueryRowContext(ctx, query).Scan(
		&databaseName,
		&databaseUser,
		&schemaName,
		&host,
		&port,
	)
	if err != nil {
		return Info{}, fmt.Errorf("get database info: %w", err)
	}

	dbHost := ""
	if host.Valid {
		dbHost = host.String
	}

	dbPort := ""
	if port.Valid {
		dbPort = fmt.Sprintf("%d", port.Int64)
	}

	return Info{
		Name:   databaseName,
		User:   databaseUser,
		Schema: schemaName,
		Host:   dbHost,
		Port:   dbPort,
	}, nil
}
