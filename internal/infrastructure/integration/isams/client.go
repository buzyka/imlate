package isams

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"

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
	c.logOutgoingRequest(req)

	return c.HTTPClient.Do(req)
}

func (c *Client) logOutgoingRequest(req *http.Request) {
	logger := c.Logger
	if logger == nil {
		logger = logging.Fallback()
	}

	requestForLog := req.Clone(req.Context())
	requestForLog.Header = req.Header.Clone()

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			logger.Infow("isams outgoing request", "method", req.Method, "url", req.URL.String(), "dumpError", err.Error())
			return
		}

		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		requestForLog.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		requestForLog.ContentLength = int64(len(bodyBytes))
	}

	for _, headerName := range []string{"Authorization", "Cookie", "X-API-Key"} {
		if requestForLog.Header.Get(headerName) != "" {
			requestForLog.Header.Set(headerName, "REDACTED")
		}
	}

	reqDump, err := httputil.DumpRequestOut(requestForLog, true)
	if err != nil {
		logger.Infow("isams outgoing request", "method", req.Method, "url", req.URL.String(), "dumpError", err.Error())
	} else {
		logger.Infow("isams outgoing request", "method", req.Method, "url", req.URL.String(), "request", string(reqDump))
	}
}
