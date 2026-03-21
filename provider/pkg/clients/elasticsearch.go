package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ElasticsearchClientConfig holds the configuration for creating an ES client.
type ElasticsearchClientConfig struct {
	Endpoints              []string
	Username               string
	Password               string
	APIKey                 string
	BearerToken            string
	ESClientAuthentication string
	Insecure               bool
	CAFile                 string
	CAData                 string
	CertFile               string
	CertData               string
	KeyFile                string
	KeyData                string
	Headers                map[string]string
}

// ElasticsearchClient provides methods to interact with the Elasticsearch API.
type ElasticsearchClient struct {
	httpClient   *http.Client
	endpoints    []string
	authMethod   AuthMethod
	username     string
	password     string
	apiKey       string
	bearerToken  string
	esClientAuth string
	headers      map[string]string
}

// NewElasticsearchClient creates a new Elasticsearch client from the given config.
func NewElasticsearchClient(cfg ElasticsearchClientConfig) (*ElasticsearchClient, error) {
	tlsConfig, err := buildTLSConfig(
		cfg.Insecure, cfg.CAFile, cfg.CAData,
		cfg.CertFile, cfg.CertData, cfg.KeyFile, cfg.KeyData,
	)
	if err != nil {
		return nil, fmt.Errorf("TLS configuration error: %w", err)
	}

	authMethod := determineAuthMethod(
		cfg.Username,
		cfg.Password,
		cfg.APIKey,
		cfg.BearerToken,
		cfg.ESClientAuthentication,
	)

	return &ElasticsearchClient{
		httpClient:   buildHTTPClient(tlsConfig),
		endpoints:    cfg.Endpoints,
		authMethod:   authMethod,
		username:     cfg.Username,
		password:     cfg.Password,
		apiKey:       cfg.APIKey,
		bearerToken:  cfg.BearerToken,
		esClientAuth: cfg.ESClientAuthentication,
		headers:      cfg.Headers,
	}, nil
}

// Do executes an HTTP request against the Elasticsearch cluster with retry logic.
// Retries continue until the context deadline expires, not based on a fixed retry count.
// Handles both transient errors (503/504) with exponential backoff and rate
// limiting (429) with Retry-After header parsing.
func (c *ElasticsearchClient) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
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
		setAuthHeaders(req, c.authMethod, c.username, c.password, c.apiKey, c.bearerToken, c.esClientAuth)

		for k, v := range c.headers {
			req.Header.Set(k, v)
		}

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

// ClusterInfo holds the response from GET /.
type ClusterInfo struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	ClusterUUID string `json:"cluster_uuid"`
	Version     struct {
		Number                           string `json:"number"`
		BuildFlavor                      string `json:"build_flavor"`
		BuildType                        string `json:"build_type"`
		BuildHash                        string `json:"build_hash"`
		BuildDate                        string `json:"build_date"`
		BuildSnapshot                    bool   `json:"build_snapshot"`
		LuceneVersion                    string `json:"lucene_version"`
		MinimumWireCompatibilityVersion  string `json:"minimum_wire_compatibility_version"`
		MinimumIndexCompatibilityVersion string `json:"minimum_index_compatibility_version"`
	} `json:"version"`
	Tagline string `json:"tagline"`
}

// GetClusterInfo calls GET / to retrieve cluster information.
func (c *ElasticsearchClient) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	resp, err := c.Do(ctx, "GET", "/", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var info ClusterInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode cluster info: %w", err)
	}

	return &info, nil
}

// Ping checks connectivity to the Elasticsearch cluster.
func (c *ElasticsearchClient) Ping(ctx context.Context) error {
	_, err := c.GetClusterInfo(ctx)
	return err
}
