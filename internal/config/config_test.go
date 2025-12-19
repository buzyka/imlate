package config

import (
	"os"
	"testing"

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

func loadTestEnvVariables(t *testing.T, env map[string]string) {
	t.Helper()
	for key, value := range env {
		err := os.Setenv(key, value)
		assert.NoError(t, err, "failed to set environment variable %s", key)
	}
}
