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

// SpacePath returns the space-scoped API path. If spaceID is empty or "default",
// it returns the path as-is. Otherwise it prefixes with /s/{spaceID}.
func SpacePath(spaceID, path string) string {
	if spaceID == "" || spaceID == "default" {
		return path
	}
	return "/s/" + spaceID + "/" + strings.TrimLeft(path, "/")
}

// GetJSON performs a GET request and decodes the JSON response into dest.
func (c *KibanaClient) GetJSON(ctx context.Context, path string, dest any) error {
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
func (c *KibanaClient) PutJSON(ctx context.Context, path string, body any, dest any) error {
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
func (c *KibanaClient) PostJSON(ctx context.Context, path string, body any, dest any) error {
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
func (c *KibanaClient) Delete(ctx context.Context, path string) error {
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

// Exists checks if a resource exists via GET (Kibana doesn't support HEAD on all endpoints).
func (c *KibanaClient) Exists(ctx context.Context, path string) (bool, error) {
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

// DeleteWithBody performs a DELETE request with a JSON body.
func (c *KibanaClient) DeleteWithBody(ctx context.Context, path string, body any) error {
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
		return &APIError{StatusCode: resp.StatusCode, Path: path, Body: string(respBody)}
	}
	return nil
}

// PostRaw performs a POST request with raw bytes and a custom content type.
func (c *KibanaClient) PostRaw(ctx context.Context, path string, contentType string, body []byte, dest any) error {
	resp, err := c.Do(ctx, "POST", path, bytes.NewReader(body))
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

// PatchJSON performs a PATCH request with a JSON body.
func (c *KibanaClient) PatchJSON(ctx context.Context, path string, body any, dest any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.Do(ctx, "PATCH", path, bytes.NewReader(data))
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
