package synthetics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// PrivateLocation manages a Kibana synthetics private location.
type PrivateLocation struct{}

// PrivateLocationInputs defines the input properties for a synthetics private location.
type PrivateLocationInputs struct {
	Label         string   `pulumi:"label"`
	AgentPolicyID string   `pulumi:"agentPolicyId"`
	Tags          []string `pulumi:"tags,optional"`
	Geo           *string  `pulumi:"geo,optional"`
	SpaceID       *string  `pulumi:"spaceId,optional"`
}

// PrivateLocationState defines the output state for a synthetics private location.
type PrivateLocationState struct {
	PrivateLocationInputs

	// Outputs
	LocationID string `pulumi:"locationId"`
}

var (
	_ infer.CustomDelete[PrivateLocationState]                        = (*PrivateLocation)(nil)
	_ infer.CustomRead[PrivateLocationInputs, PrivateLocationState]   = (*PrivateLocation)(nil)
	_ infer.CustomUpdate[PrivateLocationInputs, PrivateLocationState] = (*PrivateLocation)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *PrivateLocation) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana synthetics private location for running monitors on private infrastructure.")
	a.SetToken("kibana", "SyntheticsPrivateLocation")
}

// Annotate sets input property descriptions and defaults.
func (i *PrivateLocationInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Label, "The display label for the private location.")
	a.Describe(&i.AgentPolicyID, "The Fleet agent policy ID associated with this private location.")
	a.Describe(&i.Tags, "Tags for the private location.")
	a.Describe(&i.Geo, "Geographic coordinates as a JSON string with 'lat' and 'lon' fields.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
}

// Create provisions a new synthetics private location.
func (r *PrivateLocation) Create(
	ctx context.Context, req infer.CreateRequest[PrivateLocationInputs],
) (infer.CreateResponse[PrivateLocationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[PrivateLocationState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildPrivateLocationBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[PrivateLocationState]{}, err
	}

	var result struct {
		ID string `json:"id"`
	}

	path := clients.SpacePath(spaceID, "/api/synthetics/private_locations")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[PrivateLocationState]{},
			fmt.Errorf("failed to create synthetics private location %s: %w", req.Inputs.Label, err)
	}

	return infer.CreateResponse[PrivateLocationState]{
		ID: result.ID,
		Output: PrivateLocationState{
			PrivateLocationInputs: req.Inputs,
			LocationID:            result.ID,
		},
	}, nil
}

// Read fetches the current state of the synthetics private location.
func (r *PrivateLocation) Read(
	ctx context.Context,
	req infer.ReadRequest[PrivateLocationInputs, PrivateLocationState],
) (infer.ReadResponse[PrivateLocationInputs, PrivateLocationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[PrivateLocationInputs, PrivateLocationState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/synthetics/private_locations/"+req.ID)

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[PrivateLocationInputs, PrivateLocationState]{}, err
	}
	if !exists {
		return infer.ReadResponse[PrivateLocationInputs, PrivateLocationState]{ID: ""}, nil
	}

	return infer.ReadResponse[PrivateLocationInputs, PrivateLocationState](req), nil
}

// Update modifies an existing synthetics private location.
func (r *PrivateLocation) Update(
	ctx context.Context,
	req infer.UpdateRequest[PrivateLocationInputs, PrivateLocationState],
) (infer.UpdateResponse[PrivateLocationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[PrivateLocationState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildPrivateLocationBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[PrivateLocationState]{}, err
	}
	path := clients.SpacePath(spaceID, "/api/synthetics/private_locations/"+req.ID)

	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[PrivateLocationState]{},
			fmt.Errorf("failed to update synthetics private location %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[PrivateLocationState]{
		Output: PrivateLocationState{
			PrivateLocationInputs: req.Inputs,
			LocationID:            req.ID,
		},
	}, nil
}

// Delete removes the synthetics private location.
func (r *PrivateLocation) Delete(
	ctx context.Context, req infer.DeleteRequest[PrivateLocationState],
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
	path := clients.SpacePath(spaceID, "/api/synthetics/private_locations/"+req.State.LocationID)

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildPrivateLocationBody(inputs PrivateLocationInputs) (map[string]any, error) {
	body := map[string]any{
		"label":         inputs.Label,
		"agentPolicyId": inputs.AgentPolicyID,
	}

	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}
	if inputs.Geo != nil {
		var geo any
		if err := json.Unmarshal([]byte(*inputs.Geo), &geo); err != nil {
			return nil, fmt.Errorf("invalid geo JSON: %w", err)
		}
		body["geo"] = geo
	}

	return body, nil
}
