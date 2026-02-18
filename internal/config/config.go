package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
)

var parseEnv = env.Parse

type Config struct {
	Debug          bool   `env:"DEBUG" envDefault:"false"`
	Environment    string `env:"ENVIRONMENT" envDefault:"production"` // possible values: development, staging, production.
	DatabaseEngine string `env:"DATABASE_ENGINE" envDefault:"mysql"`
	DatabaseURL    string `env:"DATABASE_URL" envDefault:"trackme:trackme@/tracker?parseTime=true"`

	ISAMSBaseURL         string `env:"ISAMS_BASE_URL"`
	ISAMSAPIClientID     string `env:"ISAMS_API_CLIENT_ID"`
	ISAMSAPIClientSecret string `env:"ISAMS_API_CLIENT_SECRET"`

	ERPTimeZone string `env:"ERP_LOCAL_TIMEZONE"`
	APPTimeZone string `env:"APP_LOCAL_TIMEZONE" envDefault:"UTC"`

	ERPMainRegistrationPeriodType   string `env:"ERP_MAIN_REGISTRATION_PERIOD_TYPE" envDefault:"AM"`
	ERPDefaultLessonAbsenceCodeName string `env:"ERP_DEFAULT_LESSON_ABSENCE_CODE_NAME" envDefault:"O"`

	StudentsImagePhotoDir       string `env:"STUDENTS_IMAGE_PHOTO_DIR" envDefault:"website/assets/img/students"`
	StudentsImagePhotoURLPrefix string `env:"STUDENTS_IMAGE_PHOTO_URL_PREFIX" envDefault:"/assets/img/students"`

	AutoRegistrationYearGroups []int32 `env:"AUTO_REGISTRATION_YEAR_GROUPS" envSeparator:","`

	ForceERPSyncOnStart bool `env:"FORCE_ERP_SYNC_ON_START" envDefault:"false"`

	CronStudentSync            string `env:"CRON_STUDENT_SYNC" envDefault:"0 7-17/2 * * 1-5"`
	CronPhotoSync              string `env:"CRON_PHOTO_SYNC" envDefault:"0 5 * * 1-5"`
	CronRegistrationCodesSync  string `env:"CRON_REGISTRATION_CODES_SYNC" envDefault:"0 7-17/1 * * 1-5"`
	CronMarkAbsent             string `env:"CRON_MARK_ABSENT" envDefault:"10 8-12/1 * * 1-5"`

	AuthTokenSecret string `env:"AUTH_TOKEN_SECRET" envDefault:""`

	erpLocation *time.Location
	appLocation *time.Location
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
	err := parseEnv(&cfg)
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

	cfg.erpLocation = time.Local
	if cfg.ERPTimeZone != "" {
		if erpLoc, err := time.LoadLocation(cfg.ERPTimeZone); err == nil {
			cfg.erpLocation = erpLoc
		}
	}

	cfg.appLocation = time.Local
	if cfg.APPTimeZone != "" {
		if appLoc, err := time.LoadLocation(cfg.APPTimeZone); err == nil {
			cfg.appLocation = appLoc
		}
	}

	return cfg, nil
}

func getDatabaseURLForSqliteFromEnv() (url string, ok bool) {
	cfg := &SqliteDBConfig{}
	if err := parseEnv(cfg); err != nil {
		return url, false
	}
	if cfg.DatabasePath == "" {
		return url, false
	}
	return cfg.DatabasePath, true
}

func getDatabaseURLForMysqlFromEnv() (url string, ok bool) {
	cfg := &MysqlDBConfig{}
	if err := parseEnv(cfg); err != nil {
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

func (c *Config) ERPTimeLocation() *time.Location {
	return c.erpLocation
}

func (c *Config) APPTimeLocation() *time.Location {
	return c.appLocation
}

func (c *Config) IsERPIntegrated() bool {
	return c.ISAMSBaseURL != "" && c.ISAMSAPIClientID != "" && c.ISAMSAPIClientSecret != ""
}
