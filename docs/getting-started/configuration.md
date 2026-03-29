[Docs](../README.md) / [Getting Started](README.md) / Configuration

# Configuration Reference

All configuration is loaded from environment variables using the [caarlos0/env](https://github.com/caarlos0/env) library. The application loads `.env` files from the current directory and the project root on startup via `gotenv`.

Source: [`internal/config/config.go`](../../internal/config/config.go)

Example file: [`docker/.env.docker.example`](../../docker/.env.docker.example)

## Core / Application

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_PORT` | HTTP server listen port | `8080` | No |
| `ENVIRONMENT` | Runtime environment (`development`, `staging`, `production`) | `production` | No |
| `DEBUG` | Enable debug mode | `false` | No |
| `APP_LOCAL_TIMEZONE` | Application timezone (IANA format, e.g. `Europe/London`) | `UTC` | No |

## Database

The application supports MySQL (primary) and SQLite engines. MySQL is the default and recommended engine.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_ENGINE` | Database driver (`mysql` or `sqlite`) | `mysql` | No |
| `DATABASE_URL` | Full DSN (used if individual vars below are not set) | `trackme:trackme@/tracker?parseTime=true` | No |
| `DATABASE_HOST` | MySQL host | -- | For MySQL |
| `DATABASE_PORT` | MySQL port | `3306` (implicit) | No |
| `DATABASE_USERNAME` | MySQL user | -- | For MySQL |
| `DATABASE_PASSWORD` | MySQL password | -- | For MySQL |
| `DATABASE_NAME` | MySQL database name | -- | For MySQL |
| `DATABASE_PATH` | SQLite file path | -- | For SQLite |

> When `DATABASE_HOST`, `DATABASE_USERNAME`, `DATABASE_PASSWORD`, and `DATABASE_NAME` are all set, they take precedence over `DATABASE_URL` for MySQL. The connection string is built as `user:pass@tcp(host:port)/dbname?parseTime=true`.

## Authentication

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `AUTH_TOKEN_SECRET` | JWT signing secret for admin auth (HS256) | -- | **Yes** (min 32 chars) |

The application validates this on startup and panics if it is missing or too short.

## ERP / iSAMS Integration

These variables control the optional iSAMS integration. When disabled, cron jobs and ERP-dependent features are skipped entirely.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ERP_INTEGRATION_ENABLED` | Master toggle for iSAMS integration | `false` | No |
| `ISAMS_BASE_URL` | iSAMS API base URL | -- | When ERP enabled |
| `ISAMS_API_CLIENT_ID` | OAuth2 client ID | -- | When ERP enabled |
| `ISAMS_API_CLIENT_SECRET` | OAuth2 client secret | -- | When ERP enabled |
| `ERP_LOCAL_TIMEZONE` | Timezone for ERP date calculations | System local | No |
| `ERP_MAIN_REGISTRATION_PERIOD_TYPE` | Registration period type code | `AM` | No |
| `ERP_DEFAULT_LESSON_ABSENCE_CODE_NAME` | Default absence code name | `O` | No |
| `AUTO_REGISTRATION_YEAR_GROUPS` | Comma-separated year groups for auto-registration | -- | No |
| `FORCE_ERP_SYNC_ON_START` | Force a full sync on application start | `false` | No |

> ERP integration is only active when all three conditions are met: `ERP_INTEGRATION_ENABLED=true`, `ISAMS_BASE_URL` is non-empty, and both `ISAMS_API_CLIENT_ID` and `ISAMS_API_CLIENT_SECRET` are set. See `Config.IsERPIntegrated()`.

## Cron Schedules

These cron expressions are only used when ERP integration is enabled.

| Variable | Description | Default |
|----------|-------------|---------|
| `CRON_STUDENT_SYNC` | Student data sync from iSAMS | `0 7-17/2 * * 1-5` |
| `CRON_PHOTO_SYNC` | Student photo sync | `0 5 * * 1-5` |
| `CRON_REGISTRATION_CODES_SYNC` | Registration code dictionary sync | `0 7-17/1 * * 1-5` |
| `CRON_MARK_ABSENT` | Mark unregistered students as absent | `10 8-12/1 * * 1-5` |

## Storage / Images

| Variable | Description | Default |
|----------|-------------|---------|
| `STUDENTS_IMAGE_PHOTO_DIR` | Filesystem path for student photos | `storage/img/students` |
| `STUDENTS_IMAGE_PHOTO_URL_PREFIX` | URL prefix for student photo serving | `/storage/img/students` |
| `VISITOR_IMAGE_DIR` | Filesystem path for visitor images | `storage/img/visitors` |
| `VISITOR_IMAGE_URL_PREFIX` | URL prefix for visitor image serving | `/storage/img/visitors` |

Back to [Getting Started](README.md) | [Docs Index](../README.md)
