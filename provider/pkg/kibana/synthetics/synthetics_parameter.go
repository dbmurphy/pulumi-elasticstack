package synthetics

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Parameter manages a Kibana synthetics global parameter.
type Parameter struct{}

// ParameterInputs ...
type ParameterInputs struct {
	Key               string   `pulumi:"key"`
	Value             string   `pulumi:"value"`
	Description       *string  `pulumi:"description,optional"`
	Tags              []string `pulumi:"tags,optional"`
	ShareAcrossSpaces *bool    `pulumi:"shareAcrossSpaces,optional"`
	SpaceID           *string  `pulumi:"spaceId,optional"`
}

// ParameterState ...
type ParameterState struct {
	ParameterInputs

	// Outputs
	ParameterID string `pulumi:"parameterId"`
}

var (
	_ infer.CustomDelete[ParameterState]                  = (*Parameter)(nil)
	_ infer.CustomRead[ParameterInputs, ParameterState]   = (*Parameter)(nil)
	_ infer.CustomUpdate[ParameterInputs, ParameterState] = (*Parameter)(nil)
)

// Annotate ...
func (r *Parameter) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana synthetics global parameter for use in monitors.")
	a.SetToken("kibana", "Parameter")
}

// Annotate ...
func (i *ParameterInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Key, "The parameter key name.")
	a.Describe(&i.Value, "The parameter value.")
	a.Describe(&i.Description, "A description for the parameter.")
	a.Describe(&i.Tags, "Tags for the parameter.")
	a.Describe(&i.ShareAcrossSpaces, "Whether the parameter is shared across all spaces.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
}

// Create ...
func (r *Parameter) Create(
	ctx context.Context, req infer.CreateRequest[ParameterInputs],
) (infer.CreateResponse[ParameterState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[ParameterState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := buildParameterBody(req.Inputs)

	var result struct {
		ID string `json:"id"`
	}

	path := clients.SpacePath(spaceID, "/api/synthetics/params")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[ParameterState]{},
			fmt.Errorf("failed to create synthetics parameter %s: %w", req.Inputs.Key, err)
	}

	return infer.CreateResponse[ParameterState]{
		ID: result.ID,
		Output: ParameterState{
			ParameterInputs: req.Inputs,
			ParameterID:     result.ID,
		},
	}, nil
}

// Read ...
func (r *Parameter) Read(
	ctx context.Context,
	req infer.ReadRequest[ParameterInputs, ParameterState],
) (infer.ReadResponse[ParameterInputs, ParameterState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[ParameterInputs, ParameterState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/synthetics/params")

	// The params API uses a list endpoint; check if our ID still exists.
	var result []struct {
		ID string `json:"id"`
	}
	if err := kbClient.GetJSON(ctx, path, &result); err != nil {
		if clients.IsNotFound(err) {
			return infer.ReadResponse[ParameterInputs, ParameterState]{ID: ""}, nil
		}
		return infer.ReadResponse[ParameterInputs, ParameterState]{}, err
	}

	for _, p := range result {
		if p.ID == req.ID {
			return infer.ReadResponse[ParameterInputs, ParameterState](req), nil
		}
	}

	// Parameter not found in list
	return infer.ReadResponse[ParameterInputs, ParameterState]{ID: ""}, nil
}

// Update ...
func (r *Parameter) Update(
	ctx context.Context,
	req infer.UpdateRequest[ParameterInputs, ParameterState],
) (infer.UpdateResponse[ParameterState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[ParameterState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := buildParameterBody(req.Inputs)
	path := clients.SpacePath(spaceID, "/api/synthetics/params/"+req.ID)

	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[ParameterState]{},
			fmt.Errorf("failed to update synthetics parameter %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[ParameterState]{
		Output: ParameterState{
			ParameterInputs: req.Inputs,
			ParameterID:     req.ID,
		},
	}, nil
}

// Delete ...
func (r *Parameter) Delete(
	ctx context.Context, req infer.DeleteRequest[ParameterState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/synthetics/params")

	// The params API requires a DELETE with body containing the IDs to remove.
	body := map[string]any{
		"ids": []string{req.State.ParameterID},
	}

	if err := kbClient.DeleteWithBody(ctx, path, body); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildParameterBody(inputs ParameterInputs) map[string]any {
	body := map[string]any{
		"key":   inputs.Key,
		"value": inputs.Value,
	}

	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}
	if inputs.ShareAcrossSpaces != nil {
		body["share_across_spaces"] = *inputs.ShareAcrossSpaces
	}

	return body
}
