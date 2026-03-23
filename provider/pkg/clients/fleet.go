package clients

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FleetClientConfig holds the configuration for creating a Fleet client.
type FleetClientConfig struct {
	Endpoint    string
	Username    string
	Password    string
	APIKey      string
	BearerToken string
	CACerts     []string
	Insecure    bool
}

// FleetClient provides methods to interact with the Fleet API.
type FleetClient struct {
	httpClient  *http.Client
	endpoint    string
	authMethod  AuthMethod
	username    string
	password    string
	apiKey      string
	bearerToken string
}

// NewFleetClient creates a new Fleet client from the given config.
func NewFleetClient(cfg FleetClientConfig) (*FleetClient, error) {
	var caData string
	for _, certPath := range cfg.CACerts {
		data, err := readFileIfExists(certPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read Fleet CA cert %s: %w", certPath, err)
		}
		caData += string(data) + "\n"
	}

	tlsConfig, err := buildTLSConfig(cfg.Insecure, "", caData, "", "", "", "")
	if err != nil {
		return nil, fmt.Errorf("TLS configuration error: %w", err)
	}

	authMethod := determineAuthMethod(cfg.Username, cfg.Password, cfg.APIKey, cfg.BearerToken, "")

	return &FleetClient{
		httpClient:  buildHTTPClient(tlsConfig),
		endpoint:    cfg.Endpoint,
		authMethod:  authMethod,
		username:    cfg.Username,
		password:    cfg.Password,
		apiKey:      cfg.APIKey,
		bearerToken: cfg.BearerToken,
	}, nil
}

// Do executes an HTTP request against the Fleet API with retry logic.
// Retries continue until the context deadline expires.
// Handles 429 rate limiting with Retry-After parsing and 503/504 with backoff.
func (c *FleetClient) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
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

		url := strings.TrimRight(c.endpoint, "/") + "/" + strings.TrimLeft(path, "/")

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

// Ping checks connectivity to the Fleet API.
func (c *FleetClient) Ping(ctx context.Context) error {
	resp, err := c.Do(ctx, "GET", "/api/fleet/settings", nil)
	if err != nil {
		return err
	}
	drainAndClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fleet health check returned status %d", resp.StatusCode)
	}
	return nil
}
