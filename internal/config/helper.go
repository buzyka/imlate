package config

import "github.com/golobby/container/v3"

func GetGlobalConfig() *Config {
	var cfg *Config
	container.MustResolve(container.Global, &cfg)
	return cfg
}

func ERPMainRegistrationPeriodType() string {
	return GetGlobalConfig().ERPMainRegistrationPeriodType
}

func ERPDefaultLessonAbsenceCodeName() string {
	return GetGlobalConfig().ERPDefaultLessonAbsenceCodeName
}
