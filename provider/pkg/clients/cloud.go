package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CloudClientConfig holds the configuration for creating an Elastic Cloud client.
type CloudClientConfig struct {
	Endpoint string
	APIKey   string
}

// CloudClient provides methods to interact with the Elastic Cloud API.
type CloudClient struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
}

// NewCloudClient creates a new Elastic Cloud client.
func NewCloudClient(cfg CloudClientConfig) (*CloudClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("elastic cloud API key is required")
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.elastic-cloud.com"
	}

	return &CloudClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoint:   strings.TrimRight(endpoint, "/"),
		apiKey:     cfg.APIKey,
	}, nil
}

// Do executes an HTTP request against the Elastic Cloud API with retry logic.
// Retries continue until the context deadline expires.
// Handles 429 rate limiting with Retry-After parsing and 503/504 with backoff.
func (c *CloudClient) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
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

		url := c.endpoint + "/api/v1" + path

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "ApiKey "+c.apiKey)

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

// GetJSON performs a GET request and decodes the JSON response into dest.
func (c *CloudClient) GetJSON(ctx context.Context, path string, dest any) error {
	resp, err := c.Do(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{Path: path}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Path: path, Body: string(body)}
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

// PostJSON performs a POST request with a JSON body and optionally decodes the response.
func (c *CloudClient) PostJSON(ctx context.Context, path string, body any, dest any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Path: path, Body: string(respBody)}
	}

	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

// PutJSON performs a PUT request with a JSON body and optionally decodes the response.
func (c *CloudClient) PutJSON(ctx context.Context, path string, body any, dest any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Path: path, Body: string(respBody)}
	}

	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

// Delete performs a DELETE request.
func (c *CloudClient) Delete(ctx context.Context, path string) error {
	resp, err := c.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Path: path, Body: string(body)}
	}
	return nil
}
