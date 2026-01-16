package isams

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type testResponseOption func(*http.Response)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func testResponse(t *testing.T, status int, opts ...testResponseOption) *http.Response {
	t.Helper()
	resp := &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
	}

	for _, opt := range opts {
		opt(resp)
	}

	return resp
}

func WithBodyString(s string) testResponseOption {
	return func(r *http.Response) {
		r.Body = io.NopCloser(strings.NewReader(s))
		// ContentLength is optional but can be useful
		r.ContentLength = int64(len(s))
	}
}

func WithBodyBinary(b []byte) testResponseOption {
	return func(r *http.Response) {
		// copy to avoid surprises if caller mutates slice later
		cp := append([]byte(nil), b...)
		r.Body = io.NopCloser(bytes.NewReader(cp))
		r.ContentLength = int64(len(cp))
	}
}

func WithHeader(key, value string) testResponseOption {
	return func(r *http.Response) {
		r.Header.Set(key, value)
	}
}

