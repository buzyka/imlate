package entity

import "time"

type RegistrationCodeDictionary struct {
	Codes map[int32]*RegistrationCode
	UploadedAt time.Time
}

type RegistrationCode struct {
	ID   int32  
	Name string
	IsAbsenceCode bool
}

var registrationCodeDict *RegistrationCodeDictionary

func GetRegistrationCodeDictionary() *RegistrationCodeDictionary {
	return registrationCodeDict
}

func SetRegistrationCodeDictionary(dict *RegistrationCodeDictionary) {
	registrationCodeDict = dict
}
