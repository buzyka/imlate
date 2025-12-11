package gocontainer

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/buzyka/imlate/internal/config"
	"github.com/golobby/container/v3"
	"github.com/stretchr/testify/assert"
)

func TestBuild_InvalidDatabaseEngine(t *testing.T) {
	// Setup
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	defer func() {
		container.Global = oldGlobal
	}()

	cfg := &config.Config{
		DatabaseEngine: "invalid_engine",
		DatabaseURL:    "invalid_url",
	}

	// Execute & Assert - Should panic due to invalid database configuration
	assert.Panics(t, func() {
		Build(cfg)
	})
}

func TestBuild_InvalidDatabaseURL(t *testing.T) {
	// Setup
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	defer func() {
		container.Global = oldGlobal
	}()

	cfg := &config.Config{
		DatabaseEngine: "sqlite3",
		DatabaseURL:    "/invalid/path/that/does/not/exist/and/cannot/be/created/db.sqlite",
	}

	// Execute & Assert - Should panic due to invalid database path
	assert.Panics(t, func() {
		Build(cfg)
	})
}

// MockDB is a helper to simulate database connection failures
type MockDB struct{}

func (m *MockDB) Open(driverName, dataSourceName string) (*sql.DB, error) {
	return nil, errors.New("mock database connection error")
}

func TestBuild_NilConfig(t *testing.T) {
	// Setup
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	defer func() {
		container.Global = oldGlobal
	}()

	// Execute & Assert - Should panic with nil config
	assert.Panics(t, func() {
		Build(nil)
	})
}
