package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/buzyka/imlate/internal/infrastructure/util"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

var dbEngine string = "mysql"

// List of supported engines
var supportedEngines = []string{"mysql", "sqlite3"}

func Open(engine string, dataSourceName string, logger *zap.SugaredLogger) (*sql.DB, error) {
	if !util.InArray(engine, supportedEngines) {
		panic(fmt.Sprintf("Unsupported database engine: %s", engine))
	}
	dbEngine = engine
	db, err := sql.Open(dbEngine, dataSourceName)
	if err != nil {
		panic(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetConnMaxIdleTime(time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	if err := db.Ping(); err != nil {
		logger.Errorf("Error opening a connection to the database: %s;  with error message: %s", dataSourceName, err.Error())
		return db, err
	}
	if err := MigrateUp(dataSourceName); err != nil && err.Error() != "no change" {
		logger.Errorf("Error running migrations: %s", err.Error())
		return db, err
	}
	logger.Info("[DB] Connection established; Migrations applied")
	return db, nil
}

func MigrateUp(dataSourceName string) error {
	m, err := migrate.New(getMigrationSourceURL(), dbEngine+"://"+dataSourceName)
	if err != nil {
		return err
	}
	return m.Up()
}

func MigrateDown(dataSourceName string) error {
	m, err := migrate.New(getMigrationSourceURL(), dbEngine+"://"+dataSourceName)
	if err != nil {
		return err
	}
	return m.Down()
}

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
