package main

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

// migrationsFS embeds the SQL migration files so they ship inside the binary and
// need no separate deployment step.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// runMigrations applies any pending goose migrations at startup. start.sql seeds
// a fresh database with the baseline schema; every schema change after that
// baseline is an additive migration under migrations/, so both fresh and
// long-lived databases converge to the same schema instead of silently drifting.
func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}
