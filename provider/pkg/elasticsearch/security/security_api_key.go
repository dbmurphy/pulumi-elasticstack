package security

import (
	"context"
	"encoding/json"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// ApiKey manages an Elasticsearch API key via POST /_security/api_key.
type ApiKey struct{}

// ApiKeyInputs ...
type ApiKeyInputs struct {
	Name               string  `pulumi:"name"`
	RoleDescriptors    *string `pulumi:"roleDescriptors,optional"`
	Expiration         *string `pulumi:"expiration,optional"`
	Metadata           *string `pulumi:"metadata,optional"`
	RegenerateOnChange bool    `pulumi:"regenerateOnChange,optional"`
}

// ApiKeyState ...
type ApiKeyState struct {
	ApiKeyInputs

	// Outputs
	KeyID       string `pulumi:"keyId"`
	ApiKeyValue string `pulumi:"apiKeyValue" provider:"secret"`
	Encoded     string `pulumi:"encoded"     provider:"secret"`
}

var (
	_ infer.CustomDiff[ApiKeyInputs, ApiKeyState] = (*ApiKey)(nil)
	_ infer.CustomDelete[ApiKeyState]             = (*ApiKey)(nil)
)

// Annotate ...
func (r *ApiKey) Annotate(a infer.Annotator) {
	a.Describe(r, "Creates an Elasticsearch API key. API keys are immutable — changes trigger replacement.")
	a.SetToken("elasticsearch", "ApiKey")
}

// Annotate ...
func (i *ApiKeyInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the API key.")
	a.Describe(&i.RoleDescriptors, "Role descriptors as JSON.")
	a.Describe(&i.Expiration, "Expiration time (e.g. '1d', '7d').")
	a.Describe(&i.Metadata, "API key metadata as JSON.")
	a.Describe(&i.RegenerateOnChange, "Create a new key when inputs change (keys are immutable).")
	a.SetDefault(&i.RegenerateOnChange, true)
}

// Create ...
func (r *ApiKey) Create(
	ctx context.Context,
	req infer.CreateRequest[ApiKeyInputs],
) (infer.CreateResponse[ApiKeyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[ApiKeyState]{}, err
	}

	body := buildApiKeyBody(req.Inputs)

	var result struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		ApiKey  string `json:"api_key"`
		Encoded string `json:"encoded"`
	}

	if err := esClient.PostJSON(ctx, "/_security/api_key", body, &result); err != nil {
		return infer.CreateResponse[ApiKeyState]{},
			fmt.Errorf("failed to create API key %s: %w", req.Inputs.Name, err)
	}

	return infer.CreateResponse[ApiKeyState]{
		ID: result.ID,
		Output: ApiKeyState{
			ApiKeyInputs: req.Inputs,
			KeyID:        result.ID,
			ApiKeyValue:  result.ApiKey,
			Encoded:      result.Encoded,
		},
	}, nil
}

// Diff ...
func (r *ApiKey) Diff(
	ctx context.Context,
	req infer.DiffRequest[ApiKeyInputs, ApiKeyState],
) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}

	// API keys are immutable — any input change forces replacement
	if req.Inputs.Name != req.State.Name {
		diff["name"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if ptrStringChanged(req.Inputs.RoleDescriptors, req.State.RoleDescriptors) {
		diff["roleDescriptors"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if ptrStringChanged(req.Inputs.Expiration, req.State.Expiration) {
		diff["expiration"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if ptrStringChanged(req.Inputs.Metadata, req.State.Metadata) {
		diff["metadata"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}

	return p.DiffResponse{
		HasChanges:          len(diff) > 0,
		DetailedDiff:        diff,
		DeleteBeforeReplace: true,
	}, nil
}

// Delete ...
func (r *ApiKey) Delete(
	ctx context.Context,
	req infer.DeleteRequest[ApiKeyState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	body := map[string]any{
		"ids": []string{req.State.KeyID},
	}
	if err := esClient.DeleteWithBody(ctx, "/_security/api_key", body); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to invalidate API key: %w", err)
	}

	return infer.DeleteResponse{}, nil
}

func buildApiKeyBody(inputs ApiKeyInputs) map[string]any {
	body := map[string]any{
		"name": inputs.Name,
	}

	if inputs.RoleDescriptors != nil {
		var rd any
		if err := json.Unmarshal([]byte(*inputs.RoleDescriptors), &rd); err == nil {
			body["role_descriptors"] = rd
		}
	}
	if inputs.Expiration != nil {
		body["expiration"] = *inputs.Expiration
	}
	if inputs.Metadata != nil {
		var meta any
		_ = json.Unmarshal([]byte(*inputs.Metadata), &meta)
		body["metadata"] = meta
	}

	return body
}

func ptrStringChanged(a, b *string) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}
