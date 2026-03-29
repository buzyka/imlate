[Docs](../README.md) / API Reference

# API Reference

imlate exposes three API groups with different authentication requirements. The interactive Swagger UI is available at `/swagger/index.html` when the application is running.

## API Groups

### Public Endpoints

No authentication required.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/ping` | Health check, returns `{"message":"pong"}` |
| `GET` | `/` | Reader terminal UI (HTML) |
| `GET` | `/swagger/*any` | Swagger UI |
| `GET` | `/admin` | Admin SPA |
| `GET` | `/search/:id` | Look up a visitor by key |
| `POST` | `/login` | Admin JWT authentication |
| `POST` | `/refresh` | Refresh JWT token |
| `POST` | `/change-time` | Override current time (development only) |
| `POST` | `/register-terminal` | Register a new terminal device |

### Terminal-Authenticated Endpoints

Require `Terminal-Name` header and `Authorization: Bearer <token>`.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/track` | Record a visit by visitor ID and key |
| `POST` | `/find-and-track` | Look up visitor by key and record a visit |

### Admin API (JWT-Protected)

All under `/admin-api/*`. Require a valid JWT token with `admin` role.

**System**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin-api/dashboard` | Dashboard greeting |
| `GET` | `/admin-api/version` | Application version |
| `GET` | `/admin-api/current-user` | Currently authenticated user |

**User Management**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin-api/users` | List all users |
| `GET` | `/admin-api/users/:id` | Get user by ID |
| `POST` | `/admin-api/users` | Create a new user |
| `PUT` | `/admin-api/users/:id` | Update a user |
| `PUT` | `/admin-api/users/:id/password` | Update user password |
| `DELETE` | `/admin-api/users/:id` | Delete a user (soft delete) |

**Visitor Management**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin-api/visitors` | List all visitors |
| `GET` | `/admin-api/visitors/:id` | Get visitor by ID |
| `POST` | `/admin-api/visitors` | Create a new visitor |
| `PUT` | `/admin-api/visitors/:id` | Update a visitor |
| `POST` | `/admin-api/visitors/:id/image` | Upload visitor photo |
| `POST` | `/admin-api/visitors/:id/key` | Add a key to a visitor |
| `DELETE` | `/admin-api/visitors/:id/key/:key` | Remove a key from a visitor |
| `DELETE` | `/admin-api/visitors/:id` | Delete a visitor (soft delete) |

**Reports and Tracking**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin-api/reports/visits` | Paginated attendance report with filters |
| `POST` | `/admin-api/track/visit` | Manual sign-in/out on behalf of a visitor |

## Authentication

### Admin JWT Auth

Used for all `/admin-api/*` endpoints.

**Login:**

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your-password"}'
```

**Response:**

```json
{
  "code": 200,
  "expire": "2026-03-15T15:30:00Z",
  "token": "eyJhbG..."
}
```

**Using the token:**

```bash
curl http://localhost:8080/admin-api/dashboard \
  -H "Authorization: Bearer eyJhbG..."
```

Token lookup order:
1. `Authorization: Bearer <token>` header
2. `token` query parameter
3. `jwt` cookie

Token expires after 30 minutes. Use `POST /refresh` to get a new token.

### Terminal Auth

Used for `/track` and `/find-and-track` endpoints, called by physical terminal devices.

Requires two headers:
- `Terminal-Name` -- username of a user with `terminal` role
- `Authorization: Bearer <token>` -- the terminal user's password (validated via bcrypt)

```bash
curl -X POST http://localhost:8080/find-and-track \
  -H "Terminal-Name: terminal" \
  -H "Authorization: Bearer terminal-password" \
  -H "Content-Type: application/json" \
  -d '{"visit_key": "ABC123", "signed_in": true}'
```

The terminal's `admin_id` (UUID) is stored with each track record for audit purposes.

## Swagger Documentation

The API is documented with [swaggo/swag](https://github.com/swaggo/swag) annotations on handler functions. The generated spec is served at runtime and also available as static files.

### Regenerating Swagger

After modifying handler annotations:

```bash
make swag
```

This runs:

```bash
swag init -g cmd/app/main.go -o docs --parseDependency --parseInternal
```

Generated files:
- `docs/docs.go` -- embedded spec for runtime
- `docs/swagger.json` -- OpenAPI 2.0 (JSON)
- `docs/swagger.yaml` -- OpenAPI 2.0 (YAML)

### Adding Swagger Annotations

Template for a new handler:

```go
// HandlerName godoc
// @Summary      Short description
// @Description  Longer description of what this endpoint does.
// @Tags         tag-name
// @Accept       json
// @Produce      json
// @Param        request  body      RequestType   true  "Request body"
// @Param        id       path      int           true  "Resource ID"
// @Param        filter   query     string        false "Optional filter"
// @Success      200      {object}  ResponseType
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/resource [post]
```

## Error Response Format

All error responses follow a consistent JSON structure:

```json
{
  "error": "human-readable error message"
}
```

Authentication errors use a slightly different format:

```json
{
  "code": 401,
  "message": "incorrect Username or Password"
}
```

## Static Files

| Path | Serves |
|------|--------|
| `/assets/*` | Website static assets (CSS, JS, images) |
| `/storage/*` | Uploaded files (visitor photos, student images) |
| `/admin/assets/*` | Admin SPA assets |

Back to [Docs Index](../README.md)
