package clients

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestDetermineAuthMethod(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		password     string
		apiKey       string
		bearerToken  string
		esClientAuth string
		expected     AuthMethod
	}{
		{"no credentials", "", "", "", "", "", AuthNone},
		{"basic auth", "user", "pass", "", "", "", AuthBasic},
		{"api key", "", "", "key123", "", "", AuthAPIKey},
		{"bearer token", "", "", "", "token123", "", AuthBearer},
		{"bearer with client auth", "", "", "", "token123", "secret123", AuthBearerWithClientAuth},
		{"api key takes precedence over basic", "user", "pass", "key123", "", "", AuthAPIKey},
		{"bearer takes precedence over api key", "", "", "key123", "token123", "", AuthBearer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineAuthMethod(tt.username, tt.password, tt.apiKey, tt.bearerToken, tt.esClientAuth)
			if got != tt.expected {
				t.Errorf("determineAuthMethod() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSetAuthHeaders_Basic(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost", nil)
	setAuthHeaders(req, AuthBasic, "user", "pass", "", "", "")
	if req.Header.Get("Authorization") == "" {
		t.Error("expected Authorization header for basic auth")
	}
}

func TestSetAuthHeaders_APIKey(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost", nil)
	setAuthHeaders(req, AuthAPIKey, "", "", "mykey", "", "")
	auth := req.Header.Get("Authorization")
	if auth != "ApiKey mykey" {
		t.Errorf("expected 'ApiKey mykey', got '%s'", auth)
	}
}

func TestSetAuthHeaders_Bearer(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost", nil)
	setAuthHeaders(req, AuthBearer, "", "", "", "mytoken", "")
	auth := req.Header.Get("Authorization")
	if auth != "Bearer mytoken" {
		t.Errorf("expected 'Bearer mytoken', got '%s'", auth)
	}
}

func TestSetAuthHeaders_BearerWithClientAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost", nil)
	setAuthHeaders(req, AuthBearerWithClientAuth, "", "", "", "mytoken", "mysecret")
	auth := req.Header.Get("Authorization")
	if auth != "Bearer mytoken" {
		t.Errorf("expected 'Bearer mytoken', got '%s'", auth)
	}
	esAuth := req.Header.Get("ES-Client-Authentication")
	if esAuth != "SharedSecret mysecret" {
		t.Errorf("expected 'SharedSecret mysecret', got '%s'", esAuth)
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{404, false},
		{429, false}, // 429 is handled separately via isRateLimited
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		got := isRetryableStatusCode(tt.code)
		if got != tt.expected {
			t.Errorf("isRetryableStatusCode(%d) = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{200, false},
		{400, false},
		{429, true},
		{503, false},
	}

	for _, tt := range tests {
		got := isRateLimited(tt.code)
		if got != tt.expected {
			t.Errorf("isRateLimited(%d) = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestRetryBackoff(t *testing.T) {
	for attempt := 0; attempt < 10; attempt++ {
		backoff := retryBackoff(attempt)
		if backoff < 0 {
			t.Errorf("retryBackoff(%d) returned negative duration: %v", attempt, backoff)
		}
		if backoff > retryMaxInterval {
			t.Errorf("retryBackoff(%d) = %v, exceeds max %v", attempt, backoff, retryMaxInterval)
		}
	}
}

func TestRateLimitBackoff_WithRetryAfterHeader(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("Retry-After", "30")

	backoff := rateLimitBackoff(resp, 1)
	// Should be 30s + 0-2s jitter
	if backoff < 30*time.Second {
		t.Errorf("expected at least 30s, got %v", backoff)
	}
	if backoff > 32*time.Second {
		t.Errorf("expected at most 32s (30s + 2s jitter), got %v", backoff)
	}
}

func TestRateLimitBackoff_WithLargeRetryAfter(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("Retry-After", "999")

	backoff := rateLimitBackoff(resp, 1)
	// Should be capped at rateLimitMaxWait + jitter
	if backoff > rateLimitMaxWait+2*time.Second {
		t.Errorf("expected capped at %v + jitter, got %v", rateLimitMaxWait, backoff)
	}
}

func TestRateLimitBackoff_WithoutHeader(t *testing.T) {
	backoff := rateLimitBackoff(nil, 1)
	// Default wait (10s) + attempt scaling (5s) + jitter (0-5s) = 15-20s
	if backoff < rateLimitDefaultWait {
		t.Errorf("expected at least %v, got %v", rateLimitDefaultWait, backoff)
	}
	if backoff > rateLimitMaxWait+5*time.Second {
		t.Errorf("expected at most %v + jitter, got %v", rateLimitMaxWait, backoff)
	}
}

func TestRateLimitBackoff_IncreasingAttempts(t *testing.T) {
	// Higher attempt numbers should generally produce longer waits (modulo jitter)
	// Just verify they don't exceed the max
	for attempt := 1; attempt <= 10; attempt++ {
		backoff := rateLimitBackoff(nil, attempt)
		if backoff > rateLimitMaxWait+5*time.Second {
			t.Errorf("attempt %d: backoff %v exceeds max", attempt, backoff)
		}
	}
}

func TestBuildTLSConfig_Insecure(t *testing.T) {
	cfg, err := buildTLSConfig(true, "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestBuildTLSConfig_InvalidCAFile(t *testing.T) {
	_, err := buildTLSConfig(false, "/nonexistent/ca.pem", "", "", "", "", "")
	if err == nil {
		t.Error("expected error for nonexistent CA file")
	}
}

func TestBuildTLSConfig_InvalidCertFile(t *testing.T) {
	_, err := buildTLSConfig(false, "", "", "/nonexistent/cert.pem", "", "/nonexistent/key.pem", "")
	if err == nil {
		t.Error("expected error for nonexistent cert file")
	}
}

func TestEnsureDeadline_WithExistingDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	newCtx, newCancel := ensureDeadline(ctx)
	defer newCancel()

	deadline, ok := newCtx.Deadline()
	if !ok {
		t.Fatal("expected context to have deadline")
	}
	// Should use the existing deadline (5 min), not DefaultOperationTimeout (10 min)
	remaining := time.Until(deadline)
	if remaining > 5*time.Minute+time.Second {
		t.Errorf("expected deadline ~5min, got %v", remaining)
	}
}

func TestEnsureDeadline_WithoutDeadline(t *testing.T) {
	ctx := context.Background()

	newCtx, cancel := ensureDeadline(ctx)
	defer cancel()

	deadline, ok := newCtx.Deadline()
	if !ok {
		t.Fatal("expected context to have deadline after ensureDeadline")
	}
	remaining := time.Until(deadline)
	if remaining < DefaultOperationTimeout-time.Second || remaining > DefaultOperationTimeout+time.Second {
		t.Errorf("expected deadline ~%v, got %v", DefaultOperationTimeout, remaining)
	}
}

func TestSleepWithContext_NormalSleep(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	err := sleepWithContext(ctx, 50*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("expected at least 40ms sleep, got %v", elapsed)
	}
}

func TestSleepWithContext_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := sleepWithContext(ctx, 10*time.Second)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}
