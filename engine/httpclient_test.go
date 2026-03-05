package engine

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRandomUserAgent(t *testing.T) {
	ua := randomUserAgent()
	if ua == "" {
		t.Fatal("randomUserAgent returned empty string")
	}

	found := false
	for _, poolUA := range userAgentPool {
		if ua == poolUA {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("randomUserAgent returned %q, which is not in the pool", ua)
	}
}

func TestSetBrowserHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	setBrowserHeaders(req)

	checks := map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"DNT":                       "1",
		"Connection":                "keep-alive",
		"Upgrade-Insecure-Requests": "1",
	}

	for header, want := range checks {
		got := req.Header.Get(header)
		if got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}

	ua := req.Header.Get("User-Agent")
	if ua == "" {
		t.Error("User-Agent header is empty")
	}
}

// mockTransport is a configurable RoundTripper for testing.
type mockTransport struct {
	responses []*http.Response
	errors    []error
	calls     atomic.Int32
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	i := int(m.calls.Add(1)) - 1
	if i < len(m.errors) && m.errors[i] != nil {
		return nil, m.errors[i]
	}
	if i < len(m.responses) {
		return m.responses[i], nil
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func makeResp(status int) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}
}

func TestRetryTransport_RetriesOn429ThenSucceeds(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{makeResp(429), makeResp(200)},
	}
	rt := NewRetryTransport(mock, 2, time.Millisecond)
	req, _ := http.NewRequest("GET", "https://example.com", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if mock.calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", mock.calls.Load())
	}
}

func TestRetryTransport_NoRetryOn200(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{makeResp(200)},
	}
	rt := NewRetryTransport(mock, 2, time.Millisecond)
	req, _ := http.NewRequest("GET", "https://example.com", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if mock.calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", mock.calls.Load())
	}
}

func TestRetryTransport_NoRetryOn400(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{makeResp(400)},
	}
	rt := NewRetryTransport(mock, 2, time.Millisecond)
	req, _ := http.NewRequest("GET", "https://example.com", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if mock.calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", mock.calls.Load())
	}
}

func TestRetryTransport_ExhaustsRetries(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{makeResp(503), makeResp(503), makeResp(503)},
	}
	rt := NewRetryTransport(mock, 2, time.Millisecond)
	req, _ := http.NewRequest("GET", "https://example.com", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 503 {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
	if mock.calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", mock.calls.Load())
	}
}

func TestRetryTransport_RetriesNetworkError(t *testing.T) {
	netErr := errors.New("connection refused")
	mock := &mockTransport{
		responses: []*http.Response{nil, makeResp(200)},
		errors:    []error{netErr, nil},
	}
	rt := NewRetryTransport(mock, 2, time.Millisecond)
	req, _ := http.NewRequest("GET", "https://example.com", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if mock.calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", mock.calls.Load())
	}
}

func TestRetryTransport_RespectsContextCancellation(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{makeResp(503), makeResp(200)},
	}
	rt := NewRetryTransport(mock, 2, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := rt.RoundTrip(req)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
