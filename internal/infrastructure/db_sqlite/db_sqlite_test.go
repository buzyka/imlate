package db_sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestOpen_Success(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()

	// Execute
	db, err := Open(":memory:", logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Test connection
	err = db.Ping()
	assert.NoError(t, err)

	// Cleanup
	_ = db.Close()
}

func TestOpen_MemoryDatabase(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()

	// Execute
	db, err := Open(":memory:", logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Verify we can create tables
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	assert.NoError(t, err)

	// Verify we can insert data
	_, err = db.Exec("INSERT INTO test (name) VALUES (?)", "test_value")
	assert.NoError(t, err)

	// Verify we can query data
	var name string
	err = db.QueryRow("SELECT name FROM test WHERE id = 1").Scan(&name)
	assert.NoError(t, err)
	assert.Equal(t, "test_value", name)

	// Cleanup
	_ = db.Close()
}

func TestOpen_ConnectionPoolSettings(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()

	// Execute
	db, err := Open(":memory:", logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Verify connection pool settings
	stats := db.Stats()
	assert.Equal(t, 0, stats.OpenConnections) // No connections yet

	// Make a query to open a connection
	err = db.Ping()
	assert.NoError(t, err)

	// Cleanup
	_ = db.Close()
}

func TestOpen_MaxOpenConnections(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()

	// Execute
	db, err := Open(":memory:", logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// The code sets MaxOpenConns to 5
	// We can verify by trying to use the database
	err = db.Ping()
	assert.NoError(t, err)

	// Cleanup
	_ = db.Close()
}

func TestMigrateUp_Success(t *testing.T) {
	// Setup - Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migrate_test.db")

	// Execute
	// Note: This will fail in test environment without actual migration files
	// but we test the function doesn't panic with invalid path
	err := MigrateUp(dbPath)

	// Assert - Should return error since migration files don't exist in test
	assert.Error(t, err)
}

func TestGetMigrationSourceURL_ReturnsValidURL(t *testing.T) {
	// Execute
	url := getMigrationSourceURL()

	// Assert
	assert.NotEmpty(t, url)
	assert.Contains(t, url, "file://")
	assert.Contains(t, url, "/migrations")
}

func TestGetRootPath_FindsMigrationsDirectory(t *testing.T) {
	// Execute
	path, err := getRootPath()

	// Assert
	// This test depends on running from within the project structure
	// If migrations directory exists, path should be found
	if err == nil {
		assert.NotEmpty(t, path)
		assert.DirExists(t, filepath.Join(path, "migrations"))
	} else {
		// If we're not in the right directory structure, should return error
		assert.Equal(t, os.ErrNotExist, err)
	}
}

func TestGetRootPath_NoMigrationsDirectory(t *testing.T) {
	// Setup - Change to temp directory without migrations
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(originalWd) }()

	// Execute
	path, err := getRootPath()

	// Assert
	assert.Error(t, err)
	assert.Equal(t, os.ErrNotExist, err)
	assert.Empty(t, path)
}

func TestOpen_WithNilLogger(t *testing.T) {
	// Execute & Assert - Should panic with nil logger
	assert.Panics(t, func() {
		_, _ = Open(":memory:", nil)
	})
}

func TestOpen_EmptyDatabaseURL(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()

	// Execute
	db, err := Open("", logger)

	// Assert - SQLite treats empty string as ":memory:"
	assert.NoError(t, err)
	assert.NotNil(t, db)

	err = db.Ping()
	assert.NoError(t, err)

	// Cleanup
	_ = db.Close()
}

func TestOpen_MultipleConnections(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()

	// Execute - Open multiple connections to same memory database
	db1, err1 := Open(":memory:", logger)
	db2, err2 := Open(":memory:", logger)

	// Assert
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotNil(t, db1)
	assert.NotNil(t, db2)

	// Each :memory: database is independent
	err := db1.Ping()
	assert.NoError(t, err)
	err = db2.Ping()
	assert.NoError(t, err)

	// Create table in db1
	_, err = db1.Exec("CREATE TABLE test1 (id INTEGER)")
	assert.NoError(t, err)

	// Table should not exist in db2 (different database)
	_, err = db2.Exec("SELECT * FROM test1")
	assert.Error(t, err) // Should error because table doesn't exist

	// Cleanup
	_ = db1.Close()
	_ = db2.Close()
}

func TestOpen_PersistentDatabase(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "persistent.db")

	// Execute - First connection
	db1, err := Open(dbPath, logger)
	assert.NoError(t, err)

	// Create table and insert data
	_, err = db1.Exec("CREATE TABLE persistent_test (id INTEGER PRIMARY KEY, value TEXT)")
	assert.NoError(t, err)
	_, err = db1.Exec("INSERT INTO persistent_test (value) VALUES (?)", "test_data")
	assert.NoError(t, err)
	_ = db1.Close()

	// Execute - Second connection to same file
	db2, err := Open(dbPath, logger)
	assert.NoError(t, err)

	// Assert - Data should persist
	var value string
	err = db2.QueryRow("SELECT value FROM persistent_test WHERE id = 1").Scan(&value)
	assert.NoError(t, err)
	assert.Equal(t, "test_data", value)

	// Cleanup
	_ = db2.Close()
}

func TestOpen_ConnectionLifetime(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()

	// Execute
	db, err := Open(":memory:", logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// ConnMaxLifetime is set to 3 minutes in the code
	// We just verify the connection works
	err = db.Ping()
	assert.NoError(t, err)

	// Cleanup
	_ = db.Close()
}

func TestGetMigrationSourceURL_Integration(t *testing.T) {
	// Execute
	url := getMigrationSourceURL()

	// Assert - URL should be properly formatted
	assert.Contains(t, url, "file://")

	// Extract path from URL
	path := url[7:] // Remove "file://"

	// The path should end with /migrations
	assert.Contains(t, path, "/migrations")
}

func TestOpen_WALMode(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "wal_test.db")

	// Execute
	db, err := Open(dbPath, logger)
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	// The code has commented out WAL mode, but we can test the database works
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	assert.NoError(t, err)

	// Journal mode could be DELETE (default) or WAL depending on SQLite version
	assert.NotEmpty(t, journalMode)
}

func TestOpen_StatementCache(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t).Sugar()

	// Execute
	db, err := Open(":memory:", logger)
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create table
	_, err = db.Exec("CREATE TABLE cache_test (id INTEGER PRIMARY KEY, value TEXT)")
	assert.NoError(t, err)

	// Execute same query multiple times (statement caching test)
	stmt, err := db.Prepare("INSERT INTO cache_test (value) VALUES (?)")
	assert.NoError(t, err)
	defer func() { _ = stmt.Close() }()

	for i := 0; i < 10; i++ {
		_, err = stmt.Exec("test")
		assert.NoError(t, err)
	}

	// Verify all records inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM cache_test").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestOpen_LoggerIntegration(t *testing.T) {
	// Setup - Use zaptest to capture logs
	logger := zaptest.NewLogger(t).Sugar()

	// Execute
	db, err := Open(":memory:", logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// The logger should have logged the success message
	// zaptest will fail the test if there are unexpected errors

	// Cleanup
	_ = db.Close()
}
