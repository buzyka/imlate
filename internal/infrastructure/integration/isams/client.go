package isams

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	TokenEndpoint = "/auth/connect/token"
)

const (
	StudentsEndpoint = "/api/students"
	StudentByIDEndpoint = "/api/students/{schoolId}"
	RegisterEndpoint = "/api/registration/register/{registrationPeriodId}/students/{schoolId}"
	RegistrationPeriodsEndpoint = "/api/registration/periods"
	RegistrationStatusEndpoint = "/api/registration/register/{registrationPeriodId}/students/{schoolId}"
	AbsenceCodesEndpoint = "/api/registration/absencecodes"
)

const (
	defaultRegistrationPeriodID = "22947"
)

var tokenSource oauth2.TokenSource

type ClientFactory struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
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
}


func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if resp, err :=httputil.DumpRequestOut(req, true); err == nil {
		fmt.Printf("Request: %s\n", string(resp))
	}
	return c.HTTPClient.Do(req)
}
