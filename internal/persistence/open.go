package persistence

import (
	"database/sql"

	// Register the SQLite driver for binaries that only depend on persistence.
	_ "modernc.org/sqlite"
)

func GetDB(dbpath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbpath)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, err
}
