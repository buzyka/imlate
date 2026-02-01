package config

import (
	"testing"

	"github.com/golobby/container/v3"
	"github.com/stretchr/testify/assert"
)

func TestHelperFunctionsUseGlobalConfig(t *testing.T) {
	cfg := &Config{
		ERPMainRegistrationPeriodType:   "MAIN",
		ERPDefaultPresentCodeName:       "/",
		ERPDefaultLessonAbsenceCodeName: "O",
	}
	err := container.Singleton(func() *Config {
		return cfg
	})
	assert.NoError(t, err)

	assert.Equal(t, cfg, GetGlobalConfig())
	assert.Equal(t, "MAIN", ERPMainRegistrationPeriodType())
	assert.Equal(t, "/", ERPDefaultPresentCodeName())
	assert.Equal(t, "O", ERPDefaultLessonAbsenceCodeName())
}
