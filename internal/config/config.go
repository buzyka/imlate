package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Debug          bool   `env:"DEBUG" envDefault:"false"`
	Environment    string `env:"ENVIRONMENT" envDefault:"production"` // possible values: development, staging, production.
	DatabaseEngine string `env:"DATABASE_ENGINE" envDefault:"mysql"`
	DatabaseURL    string `env:"DATABASE_URL" envDefault:"trackme:trackme@/tracker?parseTime=true"`

	ISAMSBaseURL         string `env:"ISAMS_BASE_URL"`
	ISAMSAPIClientID     string `env:"ISAMS_API_CLIENT_ID"`
	ISAMSAPIClientSecret string `env:"ISAMS_API_CLIENT_SECRET"`
}

type MysqlDBConfig struct {
	Host         string `env:"DATABASE_HOST"`
	Port         string `env:"DATABASE_PORT"`
	User         string `env:"DATABASE_USERNAME"`
	Password     string `env:"DATABASE_PASSWORD"`
	DatabaseName string `env:"DATABASE_NAME"`
}

type SqliteDBConfig struct {
	DatabasePath string `env:"DATABASE_PATH"`
}

func NewFromEnv() (Config, error) {
	cfg := Config{}
	err := env.Parse(&cfg)
	if err != nil {
		return cfg, err
	}
	switch cfg.DatabaseEngine {
	case "mysql":
		if url, ok := getDatabaseURLForMysqlFromEnv(); ok {
			cfg.DatabaseURL = url
		}
	case "sqlite":
		if url, ok := getDatabaseURLForSqliteFromEnv(); ok {
			cfg.DatabaseURL = url
		}
	}

	return cfg, nil
}

func getDatabaseURLForSqliteFromEnv() (url string, ok bool) {
	cfg := &SqliteDBConfig{}
	if err := env.Parse(cfg); err != nil {
		return url, false
	}
	if cfg.DatabasePath == "" {
		return url, false
	}
	return cfg.DatabasePath, true
}

func getDatabaseURLForMysqlFromEnv() (url string, ok bool) {
	cfg := &MysqlDBConfig{}
	if err := env.Parse(cfg); err != nil {
		return url, false
	}
	if cfg.User == "" || cfg.Password == "" || cfg.DatabaseName == "" {
		return url, false
	}
	url = fmt.Sprintf("%s:%s@", cfg.User, cfg.Password)
	if cfg.Host != "" {
		if cfg.Port != "" {
			url += fmt.Sprintf("tcp(%s:%s)", cfg.Host, cfg.Port)
		} else {
			url += fmt.Sprintf("tcp(%s:3306)", cfg.Host)
		}
	}
	url += "/" + cfg.DatabaseName + "?parseTime=true"
	return url, true
}
