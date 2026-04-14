# imlate

A visitor and student tracking system built with Go. Tracks sign-in/sign-out events via physical terminal devices or manual admin actions, provides an admin panel for visitor and user management, generates attendance reports, and optionally synchronizes student data with iSAMS (school management ERP).

## Features

- **Visitor tracking** -- sign-in/out via physical key readers or manual admin entry.
- **Admin panel** -- web-based SPA for managing visitors, users, keys, and viewing reports.
- **Attendance reports** -- paginated, filterable reports aggregated per visitor per day.
- **Terminal authentication** -- dedicated auth scheme for physical tracking devices.
- **iSAMS integration** (optional) -- syncs students, photos, registration codes, and writes back attendance/absence data.
- **REST API** -- fully documented with Swagger/OpenAPI.

## Getting Started

The quickest way to run imlate is with the pre-built Docker image — no Go toolchain or repository clone required.

```bash
docker pull ghcr.io/buzyka/imlate:latest
```

```bash
docker run -d \
  --name imlate \
  -p 8080:8080 \
  -e AUTH_TOKEN_SECRET="replace-with-at-least-32-character-secret" \
  -e DATABASE_HOST="<mysql-host>" \
  -e DATABASE_USERNAME="<mysql-user>" \
  -e DATABASE_PASSWORD="<mysql-password>" \
  -e DATABASE_NAME="tracker" \
  ghcr.io/buzyka/imlate:latest
```

Verify it is running:

```bash
curl http://localhost:8080/ping
# {"message":"pong"}
```

The application requires an external **MySQL 8** database. Migrations are applied automatically on every start.

For a full setup guide including a Docker Compose example, all environment variables, and persistent storage, see [docs/getting-started/docker-image.md](docs/getting-started/docker-image.md).

## License

MIT — see [LICENSE](LICENSE) for details.

---

Want to contribute or run imlate locally for development? See [CONTRIBUTING.md](CONTRIBUTING.md).
