package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GetJSON performs a GET request and decodes the JSON response into dest.
func (c *ElasticsearchClient) GetJSON(ctx context.Context, path string, dest any) error {
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
		return fmt.Errorf("GET %s returned status %d: %s", path, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

// PutJSON performs a PUT request with a JSON body and decodes the response.
func (c *ElasticsearchClient) PutJSON(ctx context.Context, path string, body any, dest any) error {
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
		return &APIError{
			StatusCode: resp.StatusCode,
			Path:       path,
			Body:       string(respBody),
		}
	}

	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

// PostJSON performs a POST request with a JSON body and decodes the response.
func (c *ElasticsearchClient) PostJSON(ctx context.Context, path string, body any, dest any) error {
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
		return &APIError{
			StatusCode: resp.StatusCode,
			Path:       path,
			Body:       string(respBody),
		}
	}

	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

// Delete performs a DELETE request.
func (c *ElasticsearchClient) Delete(ctx context.Context, path string) error {
	resp, err := c.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Path:       path,
			Body:       string(body),
		}
	}
	return nil
}

// DeleteWithBody performs a DELETE request with a JSON body (e.g. API key invalidation).
func (c *ElasticsearchClient) DeleteWithBody(ctx context.Context, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.Do(ctx, "DELETE", path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Path:       path,
			Body:       string(respBody),
		}
	}
	return nil
}

// Exists checks if a resource exists via HEAD request.
func (c *ElasticsearchClient) Exists(ctx context.Context, path string) (bool, error) {
	resp, err := c.Do(ctx, "HEAD", path, nil)
	if err != nil {
		// Connection errors should bubble up
		if strings.Contains(err.Error(), "expired") {
			return false, err
		}
		return false, nil
	}
	drainAndClose(resp.Body)
	return resp.StatusCode == http.StatusOK, nil
}

// NotFoundError indicates a resource was not found.
type NotFoundError struct {
	Path string
}

// Error ...
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("resource not found: %s", e.Path)
}

// IsNotFound returns true if the error is a NotFoundError.
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Path       string
	Body       string
}

// Error ...
func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d on %s: %s", e.StatusCode, e.Path, e.Body)
}
