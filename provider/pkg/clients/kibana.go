package clients

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// KibanaClientConfig holds the configuration for creating a Kibana client.
type KibanaClientConfig struct {
	Endpoints   []string
	Username    string
	Password    string
	APIKey      string
	BearerToken string
	CACerts     []string
	Insecure    bool
}

// KibanaClient provides methods to interact with the Kibana API.
type KibanaClient struct {
	httpClient  *http.Client
	endpoints   []string
	authMethod  AuthMethod
	username    string
	password    string
	apiKey      string
	bearerToken string
}

// NewKibanaClient creates a new Kibana client from the given config.
func NewKibanaClient(cfg KibanaClientConfig) (*KibanaClient, error) {
	// Build CA data from cert file paths
	var caData string
	for _, certPath := range cfg.CACerts {
		data, err := readFileIfExists(certPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read Kibana CA cert %s: %w", certPath, err)
		}
		caData += string(data) + "\n"
	}

	tlsConfig, err := buildTLSConfig(cfg.Insecure, "", caData, "", "", "", "")
	if err != nil {
		return nil, fmt.Errorf("TLS configuration error: %w", err)
	}

	authMethod := determineAuthMethod(cfg.Username, cfg.Password, cfg.APIKey, cfg.BearerToken, "")

	return &KibanaClient{
		httpClient:  buildHTTPClient(tlsConfig),
		endpoints:   cfg.Endpoints,
		authMethod:  authMethod,
		username:    cfg.Username,
		password:    cfg.Password,
		apiKey:      cfg.APIKey,
		bearerToken: cfg.BearerToken,
	}, nil
}

// Do executes an HTTP request against the Kibana API with retry logic.
// Retries continue until the context deadline expires.
// Handles 429 rate limiting with Retry-After parsing and 503/504 with backoff.
func (c *KibanaClient) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	ctx, cancel := ensureDeadline(ctx)
	defer cancel()

	var lastErr error
	attempt := 0

	for {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return nil, fmt.Errorf("context expired after retries: %w", lastErr)
			}
			return nil, fmt.Errorf("context expired: %w", err)
		}

		endpoint := c.endpoints[attempt%len(c.endpoints)]
		url := strings.TrimRight(endpoint, "/") + "/" + strings.TrimLeft(path, "/")

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("kbn-xsrf", "true")
		setAuthHeaders(req, c.authMethod, c.username, c.password, c.apiKey, c.bearerToken, "")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if sleepWithContext(ctx, retryBackoff(attempt)) != nil {
				return nil, fmt.Errorf("context expired after request error: %w", lastErr)
			}
			attempt++
			continue
		}

		if isRateLimited(resp.StatusCode) {
			wait := rateLimitBackoff(resp, attempt)
			drainAndClose(resp.Body)
			lastErr = fmt.Errorf("rate limited (429) on %s %s, waiting %s", method, url, wait)
			if sleepWithContext(ctx, wait) != nil {
				return nil, fmt.Errorf("context expired while rate limited: %w", lastErr)
			}
			attempt++
			continue
		}

		if isRetryableStatusCode(resp.StatusCode) {
			drainAndClose(resp.Body)
			lastErr = fmt.Errorf("received retryable status %d from %s", resp.StatusCode, url)
			if sleepWithContext(ctx, retryBackoff(attempt)) != nil {
				return nil, fmt.Errorf("context expired after retryable error: %w", lastErr)
			}
			attempt++
			continue
		}

		return resp, nil
	}
}

// Ping checks connectivity to the Kibana API.
func (c *KibanaClient) Ping(ctx context.Context) error {
	resp, err := c.Do(ctx, "GET", "/api/status", nil)
	if err != nil {
		return err
	}
	drainAndClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("kibana health check returned status %d", resp.StatusCode)
	}
	return nil
}
