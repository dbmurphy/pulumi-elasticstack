package clients

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	defaultTimeout = 30 * time.Second

	// DefaultOperationTimeout is applied when a context has no deadline.
	// This gives long-running operations (like enrich policy execution)
	// plenty of time to complete without giving up prematurely.
	DefaultOperationTimeout = 10 * time.Minute

	// Backoff settings for non-429 errors (503, 504, network errors).
	retryBaseInterval = 1 * time.Second
	retryMaxInterval  = 30 * time.Second

	// Rate limit (429) backoff settings.
	rateLimitDefaultWait = 10 * time.Second
	rateLimitMaxWait     = 120 * time.Second
)

// AuthMethod represents the authentication method used.
type AuthMethod int

// AuthNone ...
const (
	AuthNone AuthMethod = iota
	AuthBasic
	AuthAPIKey
	AuthBearer
	AuthBearerWithClientAuth
)

// ensureDeadline returns a context with a deadline. If the parent context
// already has a deadline, it is returned as-is. Otherwise, a new context
// with DefaultOperationTimeout is created.
func ensureDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, DefaultOperationTimeout)
}

// retryBackoff computes the wait time for a retry attempt using exponential
// backoff with full jitter, capped at retryMaxInterval.
func retryBackoff(attempt int) time.Duration {
	backoff := retryBaseInterval * time.Duration(1<<attempt)
	if backoff > retryMaxInterval {
		backoff = retryMaxInterval
	}
	// Full jitter: random duration in [0, backoff]
	return time.Duration(rand.Int63n(int64(backoff))) //nolint:gosec
}

// rateLimitBackoff determines how long to wait after a 429 response.
// If the server sends a Retry-After header (seconds), we use that value
// (capped at rateLimitMaxWait). Otherwise we use rateLimitDefaultWait with
// jitter. The attempt number adds incremental backoff on repeated 429s.
func rateLimitBackoff(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, err := strconv.Atoi(ra); err == nil && seconds > 0 {
				wait := time.Duration(seconds) * time.Second
				if wait > rateLimitMaxWait {
					wait = rateLimitMaxWait
				}
				// Add small jitter (0–2s) to prevent thundering herd
				jitter := time.Duration(rand.Int63n(int64(2 * time.Second))) //nolint:gosec
				return wait + jitter
			}
		}
	}

	// No Retry-After header — use default with increasing backoff
	base := rateLimitDefaultWait + time.Duration(attempt)*5*time.Second
	if base > rateLimitMaxWait {
		base = rateLimitMaxWait
	}
	jitter := time.Duration(rand.Int63n(int64(5 * time.Second))) //nolint:gosec
	return base + jitter
}

// sleepWithContext sleeps for the given duration, returning early if the
// context is cancelled. Returns ctx.Err() if cancelled, nil otherwise.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// buildTLSConfig creates a TLS configuration from the provided parameters.
func buildTLSConfig(insecure bool, caFile, caData, certFile, certData, keyFile, keyData string) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure, //nolint:gosec // User-configurable for self-signed certs
	}

	// Load CA certificates
	if caFile != "" || caData != "" {
		caCertPool := x509.NewCertPool()

		if caFile != "" {
			caCert, err := os.ReadFile(caFile) // #nosec G304 -- file path comes from user-provided TLS config
			if err != nil {
				return nil, fmt.Errorf("failed to read CA file %s: %w", caFile, err)
			}
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate from %s", caFile)
			}
		}

		if caData != "" {
			if !caCertPool.AppendCertsFromPEM([]byte(caData)) {
				return nil, fmt.Errorf("failed to parse CA certificate data")
			}
		}

		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate for mTLS
	if (certFile != "" || certData != "") && (keyFile != "" || keyData != "") {
		var cert tls.Certificate
		var err error

		if certFile != "" && keyFile != "" {
			cert, err = tls.LoadX509KeyPair(certFile, keyFile)
		} else {
			certPEM := []byte(certData)
			keyPEM := []byte(keyData)
			if certFile != "" {
				certPEM, err = os.ReadFile(certFile) // #nosec G304 -- file path comes from user-provided TLS config
				if err != nil {
					return nil, fmt.Errorf("failed to read cert file %s: %w", certFile, err)
				}
			}
			if keyFile != "" {
				keyPEM, err = os.ReadFile(keyFile) // #nosec G304 -- file path comes from user-provided TLS config
				if err != nil {
					return nil, fmt.Errorf("failed to read key file %s: %w", keyFile, err)
				}
			}
			cert, err = tls.X509KeyPair(certPEM, keyPEM)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// buildHTTPClient creates an HTTP client with the given TLS configuration.
func buildHTTPClient(tlsConfig *tls.Config) *http.Client {
	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   defaultTimeout,
	}
}

// setAuthHeaders applies authentication headers to an HTTP request.
func setAuthHeaders(
	req *http.Request, method AuthMethod,
	username, password, apiKey, bearerToken, esClientAuth string,
) {
	switch method {
	case AuthBasic:
		req.SetBasicAuth(username, password)
	case AuthAPIKey:
		req.Header.Set("Authorization", "ApiKey "+apiKey)
	case AuthBearer:
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	case AuthBearerWithClientAuth:
		req.Header.Set("Authorization", "Bearer "+bearerToken)
		req.Header.Set("ES-Client-Authentication", "SharedSecret "+esClientAuth)
	}
}

// determineAuthMethod returns the auth method based on which credentials are provided.
func determineAuthMethod(username, password, apiKey, bearerToken, esClientAuth string) AuthMethod {
	if bearerToken != "" && esClientAuth != "" {
		return AuthBearerWithClientAuth
	}
	if bearerToken != "" {
		return AuthBearer
	}
	if apiKey != "" {
		return AuthAPIKey
	}
	if username != "" && password != "" {
		return AuthBasic
	}
	return AuthNone
}

// isRetryableStatusCode returns true if the HTTP status code is retryable
// (excluding 429 which gets its own handling path).
func isRetryableStatusCode(statusCode int) bool {
	return statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

// isRateLimited returns true if the response indicates rate limiting.
func isRateLimited(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests
}

// drainAndClose reads remaining body content and closes it.
func drainAndClose(body io.ReadCloser) {
	if body != nil {
		_, _ = io.Copy(io.Discard, body) //nolint:errcheck,gosec
		_ = body.Close()                 //nolint:errcheck,gosec
	}
}
