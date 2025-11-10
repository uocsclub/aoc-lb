package database

import (
	"sync"

	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseInst struct {
	db     *sql.DB
	dbLock sync.Mutex
}

func InitDatabase(filePath string, migrationDir string) (*DatabaseInst, error) {
	db, err := sql.Open("sqlite3", filePath)

	if err != nil {
		return nil, err
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, err
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationDir,
		"aoclb",
		driver,
	)

	if err != nil {
		return nil, err
	}

	err = migrator.Up()
	if err != nil && err != migrate.ErrNoChange {
		return nil, err
	}

	return &DatabaseInst{
		db:     db,
		dbLock: sync.Mutex{},
	}, nil
}
