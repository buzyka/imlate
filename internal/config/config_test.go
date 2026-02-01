package config

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/stretchr/testify/assert"
)

func TestGetDatabaseURLFromEnvVariableWillSetCorrectConnectionString(t *testing.T) {
	var tests = []struct {
		name        string
		newEnv      map[string]string
		expectedURL string
	}{
		{
			name: "declared all parameters",
			newEnv: map[string]string{
				"DATABASE_HOST":     "host.com",
				"DATABASE_PORT":     "3307",
				"DATABASE_USERNAME": "user1",
				"DATABASE_PASSWORD": "pwd",
				"DATABASE_NAME":     "my_db",
			},
			expectedURL: "user1:pwd@tcp(host.com:3307)/my_db?parseTime=true",
		},
		{
			name: "db port omitted used default one",
			newEnv: map[string]string{
				"DATABASE_HOST":     "host.com",
				"DATABASE_PORT":     "",
				"DATABASE_USERNAME": "user1",
				"DATABASE_PASSWORD": "pwd",
				"DATABASE_NAME":     "my_db",
			},
			expectedURL: "user1:pwd@tcp(host.com:3306)/my_db?parseTime=true",
		},
		{
			name: "db host and port omitted used default one",
			newEnv: map[string]string{
				"DATABASE_HOST":     "",
				"DATABASE_PORT":     "",
				"DATABASE_USERNAME": "user1",
				"DATABASE_PASSWORD": "pwd",
				"DATABASE_NAME":     "my_db",
			},
			expectedURL: "user1:pwd@/my_db?parseTime=true",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			loadTestEnvVariables(t, tc.newEnv)

			cfg, err := NewFromEnv()

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedURL, cfg.DatabaseURL)
		})
	}
}

func TestGetDatabaseURLFromEnvVariableWithNotCorrectConfigurationWillSetDefaultDataConnectionString(t *testing.T) {
	var tests = []struct {
		name   string
		newEnv map[string]string
	}{
		{
			name: "db user not declared",
			newEnv: map[string]string{
				"DATABASE_USERNAME": "",
				"DATABASE_PASSWORD": "pwd",
				"DATABASE_NAME":     "my_db",
			},
		},
		{
			name: "db password not declared",
			newEnv: map[string]string{
				"DATABASE_USERNAME": "user1",
				"DATABASE_PASSWORD": "",
				"DATABASE_NAME":     "my_db",
			},
		},
		{
			name: "db name not declared",
			newEnv: map[string]string{
				"DATABASE_USERNAME": "user1",
				"DATABASE_PASSWORD": "pwd",
				"DATABASE_NAME":     "",
			},
		},
		{
			name: "all credentials not declared",
			newEnv: map[string]string{
				"DATABASE_USERNAME": "",
				"DATABASE_PASSWORD": "",
				"DATABASE_NAME":     "",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.newEnv["DATABASE_HOST"] = ""
			tc.newEnv["DATABASE_PORT"] = ""
			loadTestEnvVariables(t, tc.newEnv)

			cfg, err := NewFromEnv()

			assert.Nil(t, err)
			assert.Equal(t, "trackme:trackme@/tracker?parseTime=true", cfg.DatabaseURL)
		})
	}
}

func TestGetTimeLocationFromEnvVariableWillSetCorrectLocation(t *testing.T) {
	var tests = []struct {
		name           string
		newEnv         map[string]string
		expectedERPLoc string
		expectedAPPLoc string
	}{
		{
			name: "valid timezone locations",
			newEnv: map[string]string{
				"ERP_LOCAL_TIMEZONE": "Europe/London",
				"APP_LOCAL_TIMEZONE": "Europe/Paris",
			},
			expectedERPLoc: "Europe/London",
			expectedAPPLoc: "Europe/Paris",
		},
		{
			name: "empty timezone locations",
			newEnv: map[string]string{
				"ERP_LOCAL_TIMEZONE": "",
				"APP_LOCAL_TIMEZONE": "",
			},
			expectedERPLoc: time.Now().Location().String(),
			expectedAPPLoc: time.Now().Location().String(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			loadTestEnvVariables(t, tc.newEnv)

			cfg, err := NewFromEnv()
			assert.Nil(t, err)

			exERPLocation, err := time.LoadLocation(tc.expectedERPLoc)
			assert.Nil(t, err)

			assert.Equal(t, exERPLocation, cfg.ERPTimeLocation())

			exAPPLocation, err := time.LoadLocation(tc.expectedAPPLoc)
			assert.Nil(t, err)

			assert.Equal(t, exAPPLocation, cfg.APPTimeLocation())
		})
	}
}

func TestNewFromEnv_InvalidTimezoneFallsBackToLocal(t *testing.T) {
	loadTestEnvVariables(t, map[string]string{
		"ERP_LOCAL_TIMEZONE": "Invalid/Zone",
		"APP_LOCAL_TIMEZONE": "Invalid/Zone",
	})

	cfg, err := NewFromEnv()
	assert.NoError(t, err)

	assert.Equal(t, time.Local, cfg.ERPTimeLocation())
	assert.Equal(t, time.Local, cfg.APPTimeLocation())
}

func TestNewFromEnv_SqliteDatabasePath(t *testing.T) {
	loadTestEnvVariables(t, map[string]string{
		"DATABASE_ENGINE": "sqlite",
		"DATABASE_PATH":   "/tmp/test.db",
	})

	cfg, err := NewFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/test.db", cfg.DatabaseURL)
}

func TestNewFromEnv_SqliteMissingPathUsesDefault(t *testing.T) {
	loadTestEnvVariables(t, map[string]string{
		"DATABASE_ENGINE": "sqlite",
	})

	cfg, err := NewFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "trackme:trackme@/tracker?parseTime=true", cfg.DatabaseURL)
}

func TestNewFromEnv_UnknownEngineKeepsDatabaseURL(t *testing.T) {
	loadTestEnvVariables(t, map[string]string{
		"DATABASE_ENGINE": "postgres",
		"DATABASE_URL":    "custom-url",
	})

	cfg, err := NewFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "custom-url", cfg.DatabaseURL)
}

func TestNewFromEnv_ParseError(t *testing.T) {
	oldParse := parseEnv
	parseEnv = func(_ interface{}, _ ...env.Options) error {
		return errors.New("parse error")
	}
	t.Cleanup(func() {
		parseEnv = oldParse
	})

	_, err := NewFromEnv()
	assert.Error(t, err)
}

func TestGetDatabaseURLForSqliteFromEnv_ParseError(t *testing.T) {
	oldParse := parseEnv
	parseEnv = func(_ interface{}, _ ...env.Options) error {
		return errors.New("parse error")
	}
	t.Cleanup(func() {
		parseEnv = oldParse
	})

	_, ok := getDatabaseURLForSqliteFromEnv()
	assert.False(t, ok)
}

func TestGetDatabaseURLForMysqlFromEnv_ParseError(t *testing.T) {
	oldParse := parseEnv
	parseEnv = func(_ interface{}, _ ...env.Options) error {
		return errors.New("parse error")
	}
	t.Cleanup(func() {
		parseEnv = oldParse
	})

	_, ok := getDatabaseURLForMysqlFromEnv()
	assert.False(t, ok)
}

func loadTestEnvVariables(t *testing.T, env map[string]string) {
	t.Helper()
	resetConfigEnv(t)
	for key, value := range env {
		err := os.Setenv(key, value)
		assert.NoError(t, err, "failed to set environment variable %s", key)
	}
}

func resetConfigEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"DEBUG",
		"ENVIRONMENT",
		"DATABASE_ENGINE",
		"DATABASE_URL",
		"ISAMS_BASE_URL",
		"ISAMS_API_CLIENT_ID",
		"ISAMS_API_CLIENT_SECRET",
		"ERP_LOCAL_TIMEZONE",
		"APP_LOCAL_TIMEZONE",
		"ERP_MAIN_REGISTRATION_PERIOD_TYPE",
		"ERP_DEFAULT_PRESENT_CODE_NAME",
		"ERP_DEFAULT_LESSON_ABSENCE_CODE_NAME",
		"STUDENTS_IMAGE_PHOTO_DIR",
		"STUDENTS_IMAGE_PHOTO_URL_PREFIX",
		"DATABASE_HOST",
		"DATABASE_PORT",
		"DATABASE_USERNAME",
		"DATABASE_PASSWORD",
		"DATABASE_NAME",
		"DATABASE_PATH",
	}

	previous := map[string]*string{}
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			val := value
			previous[key] = &val
		} else {
			previous[key] = nil
		}
		err := os.Unsetenv(key)
		assert.NoError(t, err, "failed to unset environment variable %s", key)
	}

	t.Cleanup(func() {
		for key, value := range previous {
			if value == nil {
				_ = os.Unsetenv(key)
				continue
			}
			_ = os.Setenv(key, *value)
		}
	})
}
