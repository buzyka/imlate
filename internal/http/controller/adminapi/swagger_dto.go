package adminapi

import usecase "github.com/buzyka/imlate/internal/usecase/adminapi"

// ErrorResponse is a shared error payload for admin handlers.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse is a shared success payload for simple message responses.
type MessageResponse struct {
	Message string `json:"message"`
}

// DashboardResponse is the success payload for admin dashboard endpoint.
type DashboardResponse struct {
	Message string `json:"message"`
}

// AuthErrorResponse matches auth middleware unauthorized response payload.
type AuthErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// AuthTokenResponse matches gin-jwt login/refresh token response payload.
type AuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// LoginRequest defines credentials payload for /login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest defines refresh token payload for /refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// VersionResponse is the success payload for the version endpoint.
type VersionResponse struct {
	Version string `json:"version"`
}

// VisitorResponse documents admin visitor payloads.
type VisitorResponse = usecase.VisitorResponse

// VisitorListResponse documents visitor list payloads.
type VisitorListResponse = []usecase.VisitorResponse

// PostReportsVisitsRequest is a Swagger-visible alias for the POST reports request body.
type PostReportsVisitsRequest = usecase.PostReportsVisitsRequest

// PostReportsVisitsResponse is a Swagger-visible alias for the POST reports response.
type PostReportsVisitsResponse = usecase.PostReportsVisitsResponse
