package isams

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/infrastructure/logging"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	TokenEndpoint = "/auth/connect/token"
)

const (
	StudentsEndpoint                 = "/api/students"
	StudentByIDEndpoint              = "/api/students/{schoolId}"
	RegisterEndpoint                 = "/api/registration/register/{registrationPeriodId}/students/{schoolId}"
	RegistrationPeriodsEndpoint      = "/api/registration/periods"
	RegistrationStatusEndpoint       = "/api/registration/register/{registrationPeriodId}/students/{schoolId}"
	AbsenceCodesEndpoint             = "/api/registration/absencecodes"
	RegistrationPresentCodesEndpoint = "/api/registration/presentcodes"
	YearGroupsDivisionsEndpoint      = "/api/school/yeargroups/{yearGroupId}/divisions"
	StudentCurrentPhotoEndpoint      = "/api/students/{schoolId}/photos/current"
)

const (
	defaultRegistrationPeriodID = "22947"
)

var tokenSource oauth2.TokenSource

type ClientFactory struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Logger       *zap.SugaredLogger
}

func (f *ClientFactory) NewClient(ctx context.Context) (*Client, error) {
	baseURL := strings.TrimRight(f.BaseURL, "/")
	tokenURL := baseURL + TokenEndpoint
	cfg := clientcredentials.Config{
		ClientID:     f.ClientID,
		ClientSecret: f.ClientSecret,
		TokenURL:     tokenURL,
	}

	ts := f.getTokenSource(ctx, cfg, baseURL)
	// Validate token source by fetching a token
	if _, err := ts.Token(); err != nil {
		return nil, err
	}

	oAuthClient := oauth2.NewClient(ctx, ts)

	return &Client{
		HTTPClient: oAuthClient,
		BaseURL:    baseURL,
		Logger:     f.Logger,
	}, nil
}

func (f *ClientFactory) getTokenSource(ctx context.Context, cfg clientcredentials.Config, baseURL string) oauth2.TokenSource {
	if tokenSource != nil {
		return tokenSource
	}

	baseTS := cfg.TokenSource(ctx)
	tokenSource = oauth2.ReuseTokenSource(nil, baseTS)

	return tokenSource
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Logger     *zap.SugaredLogger
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := c.HTTPClient.Do(req)
	c.logRequestResult(req, resp, err, time.Since(start))
	return resp, err
}

// logRequestResult logs how an iSAMS request ended. Only fields that are safe
// by construction are logged: headers are never dumped, so credentials added
// now or later (including the OAuth token the transport sets) cannot reach the
// logs; the query string and body are left out for the same reason.
func (c *Client) logRequestResult(req *http.Request, resp *http.Response, err error, duration time.Duration) {
	logger := c.Logger
	if logger == nil {
		logger = logging.Fallback()
	}

	fields := []any{
		"method", req.Method,
		"host", req.URL.Host,
		"path", req.URL.Path,
		"duration_ms", duration.Milliseconds(),
	}
	if err != nil {
		logger.Debugw("isams request failed", append(fields, "error", requestErrorMessage(err))...)
		return
	}
	logger.Debugw("isams request",
		append(fields, "status", resp.StatusCode, "content_length", resp.ContentLength)...)
}

// requestErrorMessage drops the request URL that *url.Error puts into its
// message, since it carries the query string.
func requestErrorMessage(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Op + ": " + urlErr.Err.Error()
	}
	return err.Error()
}
