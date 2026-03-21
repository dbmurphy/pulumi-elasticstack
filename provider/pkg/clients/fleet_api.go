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
func (c *FleetClient) GetJSON(ctx context.Context, path string, dest any) error {
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

// PutJSON performs a PUT request with a JSON body and optionally decodes the response.
func (c *FleetClient) PutJSON(ctx context.Context, path string, body any, dest any) error {
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

// PostJSON performs a POST request with a JSON body and optionally decodes the response.
func (c *FleetClient) PostJSON(ctx context.Context, path string, body any, dest any) error {
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

// Delete performs a DELETE request.
func (c *FleetClient) Delete(ctx context.Context, path string) error {
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
		return &APIError{StatusCode: resp.StatusCode, Path: path, Body: string(body)}
	}
	return nil
}

// Exists checks if a resource exists via GET.
func (c *FleetClient) Exists(ctx context.Context, path string) (bool, error) {
	resp, err := c.Do(ctx, "GET", path, nil)
	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return false, err
		}
		return false, nil
	}
	drainAndClose(resp.Body)
	return resp.StatusCode == http.StatusOK, nil
}
