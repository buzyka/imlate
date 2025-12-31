package config

import "github.com/golobby/container/v3"

func GetGlobalConfig() *Config {
	var cfg *Config
	container.MustResolve(container.Global, &cfg)
	return cfg
}

func ERPFirstRegistrationPeriodName() string {
	return GetGlobalConfig().ERPFirstRegistrationPeriodName
}

func ERPDefaultPresentCodeName() string {
	return GetGlobalConfig().ERPDefaultPresentCodeName
}

func ERPDefaultLateCodeName() string {
	return GetGlobalConfig().ERPDefaultLateCodeName
}
