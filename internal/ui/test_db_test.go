package ui

import (
	"database/sql"
	"testing"

	"github.com/dhth/hours/internal/persistence"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func newMigratedTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	require.NoError(t, persistence.InitDB(db))
	require.NoError(t, persistence.UpgradeDB(db, 1))

	return db
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return newMigratedTestDB(t)
}
