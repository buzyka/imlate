package entity

import (
	"time"

	"github.com/buzyka/imlate/internal/config"
)

type RegistrationCodeDictionary struct {
	Codes map[int32]*RegistrationCode
	UploadedAt time.Time
}

type RegistrationCode struct {
	ID   int32
	Code string  
	Name string
	IsAbsenceCode bool
}

var presentsCodeDict *RegistrationCodeDictionary
var absenceCodeDict *RegistrationCodeDictionary

func GetPresentsCodeDictionary() *RegistrationCodeDictionary {
	return presentsCodeDict
}

func SetPresentsCodeDictionary(dict *RegistrationCodeDictionary) {
	presentsCodeDict = dict
}

func GetAbsenceCodeDictionary() *RegistrationCodeDictionary {
	return absenceCodeDict
}

func SetAbsenceCodeDictionary(dict *RegistrationCodeDictionary) {
	absenceCodeDict = dict
}

func GetDefaultLessonAbsenceCode() (*RegistrationCode, bool) {
	dlac := config.ERPDefaultLessonAbsenceCodeName()
	dic := GetAbsenceCodeDictionary()	
	if dic == nil {
		return nil, false
	}
	return dic.GetCodeByCodeName(dlac)
}

func (rcd *RegistrationCodeDictionary) GetCodeByCodeName(code string) (*RegistrationCode, bool) {
	for _, c := range rcd.Codes {
		if c.Code == code {
			return c, true
		}
	}
	return nil, false
}
