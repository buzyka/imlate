package db_sqlite

import (
	"database/sql"
	"time"

	"os"
	"path/filepath"

	// "time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

const dbEngine = "sqlite3"

func Open(dataSourceName string, logger *zap.SugaredLogger) (*sql.DB, error) {
	db, err := sql.Open(dbEngine, dataSourceName)
	if err != nil {
		panic(err.Error())
	}
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(time.Minute * 3)
	// if err := MigrateUp(dataSourceName); err != nil && err.Error() != "no change" {
	// 	logger.Errorf("Error running migrations: %s", err.Error())
	// 	return db, err
	// }
	// initMe(db)
	logger.Info("[DB] Connection established; Migrations applied")
	return db, nil
}

// func initMe(connection *sql.DB) {
// 	if _, err := connection.Exec("PRAGMA journal_mode=WAL;"); err != nil {
// 		panic("Pragma error: " + err.Error())
// 	}
// 	_, err := connection.Exec(`
// 		CREATE TABLE IF NOT EXISTS visitors (
// 			id TEXT PRIMARY KEY,
// 			name TEXT NOT NULL,
// 			surname TEXT NOT NULL,
// 			grade INTEGER NULL,
// 			image TEXT NULL
// 			);`)
// 	if err != nil {
// 		panic("Migration error: " + err.Error())	
// 	}
// }

func MigrateUp(dataSourceName string) error {
	m, err := migrate.New(getMigrationSourceURL(), dbEngine+"://"+dataSourceName)
	if err != nil {
		return err
	}
	return m.Up()
}

// func MigrateDown(dataSourceName string) error {
// 	m, err := migrate.New(getMigrationSourceURL(), dbEngine+"://"+dataSourceName)
// 	if err != nil {
// 		return err
// 	}
// 	return m.Down()
// }

func getMigrationSourceURL() string {
	migrationSourceURL := "file://"

	if path, err := getRootPath(); err == nil {
		migrationSourceURL += path + "/migrations"
	}

	return migrationSourceURL
}

func getRootPath() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		for {
			if info, errDir := os.Stat(cwd + "/migrations"); errDir == nil && info.IsDir() {
				return cwd, nil
			}
			parent := filepath.Dir(cwd)
			if parent == cwd {
				break
			}
			cwd = parent
		}
	}
	return "", os.ErrNotExist
}
