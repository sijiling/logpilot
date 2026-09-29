// Package migrations exposes versioned database migrations.
package migrations

import _ "embed"

// Initial contains the first LogPilot SQLite schema migration.
//
//go:embed 001_init.sql
var Initial string
