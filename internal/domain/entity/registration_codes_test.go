package entity

import (
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/golobby/container/v3"
	"github.com/stretchr/testify/assert"
)

func TestRegistrationCodeDictionary_GetCodeByCodeName(t *testing.T) {
	rc1 := &RegistrationCode{ID: 1, Code: "/", Name: "Present", IsAbsenceCode: false}
	rc2 := &RegistrationCode{ID: 2, Code: "O", Name: "Absent", IsAbsenceCode: true}
	dict := &RegistrationCodeDictionary{
		Codes: map[int32]*RegistrationCode{
			rc1.ID: rc1,
			rc2.ID: rc2,
		},
		UploadedAt: time.Now(),
	}

	t.Run("Found", func(t *testing.T) {
		code, ok := dict.GetCodeByCodeName("O")
		assert.True(t, ok)
		assert.Equal(t, rc2, code)
	})

	t.Run("NotFound", func(t *testing.T) {
		code, ok := dict.GetCodeByCodeName("X")
		assert.False(t, ok)
		assert.Nil(t, code)
	})

	t.Run("EmptyDictionary", func(t *testing.T) {
		empty := &RegistrationCodeDictionary{}
		code, ok := empty.GetCodeByCodeName("O")
		assert.False(t, ok)
		assert.Nil(t, code)
	})
}

func TestRegistrationCodeDictionary_SetGet(t *testing.T) {
	oldPresents := GetPresentsCodeDictionary()
	oldAbsence := GetAbsenceCodeDictionary()
	defer func() {
		SetPresentsCodeDictionary(oldPresents)
		SetAbsenceCodeDictionary(oldAbsence)
	}()

	presentDict := &RegistrationCodeDictionary{
		Codes: map[int32]*RegistrationCode{
			1: {ID: 1, Code: "/", Name: "Present"},
		},
		UploadedAt: time.Now(),
	}
	absenceDict := &RegistrationCodeDictionary{
		Codes: map[int32]*RegistrationCode{
			11: {ID: 11, Code: "O", Name: "Absent", IsAbsenceCode: true},
		},
		UploadedAt: time.Now(),
	}

	SetPresentsCodeDictionary(presentDict)
	SetAbsenceCodeDictionary(absenceDict)

	assert.Equal(t, presentDict, GetPresentsCodeDictionary())
	assert.Equal(t, absenceDict, GetAbsenceCodeDictionary())
}

func TestGetDefaultLessonAbsenceCode(t *testing.T) {
	oldAbsence := GetAbsenceCodeDictionary()
	defer SetAbsenceCodeDictionary(oldAbsence)

	setTestConfig(t, &config.Config{
		ERPDefaultLessonAbsenceCodeName: "O",
	})

	t.Run("DictionaryNil", func(t *testing.T) {
		SetAbsenceCodeDictionary(nil)
		code, ok := GetDefaultLessonAbsenceCode()
		assert.False(t, ok)
		assert.Nil(t, code)
	})

	t.Run("Found", func(t *testing.T) {
		absenceDict := &RegistrationCodeDictionary{
			Codes: map[int32]*RegistrationCode{
				11: {ID: 11, Code: "O", Name: "Absent", IsAbsenceCode: true},
			},
			UploadedAt: time.Now(),
		}
		SetAbsenceCodeDictionary(absenceDict)

		code, ok := GetDefaultLessonAbsenceCode()
		assert.True(t, ok)
		assert.Equal(t, absenceDict.Codes[11], code)
	})
}

func setTestConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	err := container.Singleton(func() *config.Config {
		return cfg
	})
	assert.NoError(t, err)
}
