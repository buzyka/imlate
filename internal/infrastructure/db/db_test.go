package db

import (
	"bufio"
	"bytes"
	"os"
	"testing"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/assert"
	"github.com/subosito/gotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestOpenWithIncorrectURLWillPanic(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	assert.Panics(t, func() {
		Open("mysql", "notURL", logger) //nolint:errcheck
	})
}

func TestOpenWithIncorrectConnectionWillReturnError(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logBuffer),
		zapcore.DebugLevel,
	)).Sugar()

	_, err := Open("mysql", "user:password@/db_name", logger)

	assert.NotNil(t, err)
	assert.Contains(t, logBuffer.String(), "Error opening a connection to the database")
}

func TestOpen(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logBuffer),
		zapcore.DebugLevel,
	)).Sugar()
	cfg, _ := prepareTestEnv(t)

	db, err := Open("mysql", cfg.DatabaseURL, logger)

	assert.Nil(t, err)
	assert.NotNil(t, db)
	assert.Contains(t, logBuffer.String(), "[DB] Connection established; Migrations applied")
}

func TestMigrateUpWillApplyMigrationsOnlyOnce(t *testing.T) {
	cfg, _ := prepareTestEnv(t)
	MigrateDown(cfg.DatabaseURL) //nolint:errcheck

	err := MigrateUp(cfg.DatabaseURL)
	assert.Nil(t, err)

	err = MigrateUp(cfg.DatabaseURL)
	assert.NotNil(t, err)
	assert.Equal(t, "no change", err.Error())
}

func TestMigrationWithIncorrectURLWillReturnError(t *testing.T) {
	databaseURL := "shopware:shopware@/db_name"
	err := MigrateUp(databaseURL)
	assert.NotNil(t, err)
	assert.NotEqual(t, "no change", err.Error())

	err = MigrateDown(databaseURL)
	assert.NotNil(t, err)
	assert.NotEqual(t, "no change", err.Error())
}

func TestOpenWillApplyMigrations(t *testing.T) {
	cfg, logger := prepareTestEnv(t)
	MigrateDown(cfg.DatabaseURL) //nolint:errcheck

	_, err := Open("mysql", cfg.DatabaseURL, logger)
	assert.Nil(t, err)
	err = MigrateUp(cfg.DatabaseURL)
	assert.NotNil(t, err)
	assert.Equal(t, "no change", err.Error())
}

func TestOpenWithFailedMigrationWillReturnError(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logBuffer),
		zapcore.DebugLevel,
	)).Sugar()
	cfg, _ := prepareTestEnv(t)

	testCase := setUp(t, cfg.DatabaseURL)
	defer testCase.tearDown()

	testCase.createTestMigration(`CREATE TABLE ...`)

	_, err := Open("mysql", cfg.DatabaseURL, logger)

	assert.Error(t, err)
	assert.Contains(t, logBuffer.String(), "Error running migrations")
}

func prepareTestEnv(t *testing.T) (*config.Config, *zap.SugaredLogger) {
	t.Helper()
	_ = gotenv.Load("../../../.env")
	logger := zaptest.NewLogger(t).Sugar()
	cfg, cfgErr := config.NewFromEnv()
	assert.Nil(t, cfgErr)
	cfg.DatabaseURL += "_test"
	return &cfg, logger
}

type MigrationTestCase struct {
	t                 *testing.T
	migration         *migrate.Migrate
	stableVersion     uint
	testMigrationFile string
}

func setUp(t *testing.T, dataSourceName string) *MigrationTestCase {
	t.Helper()
	migrationObj, err := migrate.New(getMigrationSourceURL(), "mysql://"+dataSourceName)
	assert.Nil(t, err)
	stableVersion, _, _ := migrationObj.Version()

	return &MigrationTestCase{
		t:                 t,
		migration:         migrationObj,
		stableVersion:     stableVersion,
		testMigrationFile: "100000_incorrect_sql.up.sql",
	}
}

func (tc *MigrationTestCase) tearDown() {
	tc.removeTestMigration()
	err := tc.migration.Force(int(tc.stableVersion))
	assert.Nil(tc.t, err)
}

func (tc *MigrationTestCase) removeTestMigration() {
	rootPath, er := util.GetRootPath()
	assert.Nil(tc.t, er)
	migrationPath := rootPath + "/migrations/" + tc.testMigrationFile
	if util.FileExists(migrationPath) {
		err := os.Remove(migrationPath)
		assert.Nil(tc.t, err)
	}
}

func (tc *MigrationTestCase) createTestMigration(content string) {
	rootPath, er := util.GetRootPath()
	assert.Nil(tc.t, er)
	migrationPath := rootPath + "/migrations/" + tc.testMigrationFile
	file, err := os.Create(migrationPath)
	assert.Nil(tc.t, err)
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(content)
	assert.Nil(tc.t, err)
	err = writer.Flush()
	assert.Nil(tc.t, err)
}
