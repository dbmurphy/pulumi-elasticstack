package fleet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// IntegrationPolicy manages a Fleet package policy (integration policy) via the
// /api/fleet/package_policies API.
type IntegrationPolicy struct{}

// IntegrationPolicyInputs ...
type IntegrationPolicyInputs struct {
	Name               string  `pulumi:"name"`
	Namespace          *string `pulumi:"namespace,optional"`
	Description        *string `pulumi:"description,optional"`
	AgentPolicyID      string  `pulumi:"agentPolicyId"`
	IntegrationName    string  `pulumi:"integrationName"`
	IntegrationVersion string  `pulumi:"integrationVersion"`
	Input              *string `pulumi:"input,optional"`
	Vars               *string `pulumi:"vars,optional"`
	Force              *bool   `pulumi:"force,optional"`
	AdoptOnCreate      bool    `pulumi:"adoptOnCreate,optional"`
}

// IntegrationPolicyState ...
type IntegrationPolicyState struct {
	IntegrationPolicyInputs

	// Outputs
	PolicyID string `pulumi:"policyId"`
}

var (
	_ infer.CustomDelete[IntegrationPolicyState]                          = (*IntegrationPolicy)(nil)
	_ infer.CustomRead[IntegrationPolicyInputs, IntegrationPolicyState]   = (*IntegrationPolicy)(nil)
	_ infer.CustomUpdate[IntegrationPolicyInputs, IntegrationPolicyState] = (*IntegrationPolicy)(nil)
)

// Annotate ...
func (r *IntegrationPolicy) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Fleet integration policy (package policy).")
	a.SetToken("fleet", "IntegrationPolicy")
}

// Annotate ...
func (i *IntegrationPolicyInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the integration policy.")
	a.Describe(&i.Namespace, "The namespace for the integration policy.")
	a.Describe(&i.Description, "A description for the integration policy.")
	a.Describe(&i.AgentPolicyID, "The ID of the agent policy to attach this integration to.")
	a.Describe(&i.IntegrationName, "The integration package name.")
	a.Describe(&i.IntegrationVersion, "The integration package version.")
	a.Describe(&i.Input, "JSON array of input configurations for the integration.")
	a.Describe(&i.Vars, "JSON object of package-level variables.")
	a.Describe(&i.Force, "Force the operation even if it would be destructive.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing integration policy into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *IntegrationPolicy) Create(
	ctx context.Context, req infer.CreateRequest[IntegrationPolicyInputs],
) (infer.CreateResponse[IntegrationPolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.CreateResponse[IntegrationPolicyState]{}, err
	}

	body, err := buildIntegrationPolicyBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[IntegrationPolicyState]{}, err
	}

	var result struct {
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}

	if err := fleetClient.PostJSON(ctx, "/api/fleet/package_policies", body, &result); err != nil {
		return infer.CreateResponse[IntegrationPolicyState]{},
			fmt.Errorf("failed to create integration policy %s: %w", req.Inputs.Name, err)
	}

	return infer.CreateResponse[IntegrationPolicyState]{
		ID: result.Item.ID,
		Output: IntegrationPolicyState{
			IntegrationPolicyInputs: req.Inputs,
			PolicyID:                result.Item.ID,
		},
	}, nil
}

// Read ...
func (r *IntegrationPolicy) Read(
	ctx context.Context,
	req infer.ReadRequest[IntegrationPolicyInputs, IntegrationPolicyState],
) (infer.ReadResponse[IntegrationPolicyInputs, IntegrationPolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.ReadResponse[IntegrationPolicyInputs, IntegrationPolicyState]{}, err
	}

	exists, err := fleetClient.Exists(ctx, "/api/fleet/package_policies/"+req.ID)
	if err != nil {
		return infer.ReadResponse[IntegrationPolicyInputs, IntegrationPolicyState]{}, err
	}
	if !exists {
		return infer.ReadResponse[IntegrationPolicyInputs, IntegrationPolicyState]{ID: ""}, nil
	}

	return infer.ReadResponse[IntegrationPolicyInputs, IntegrationPolicyState](req), nil
}

// Update ...
func (r *IntegrationPolicy) Update(
	ctx context.Context,
	req infer.UpdateRequest[IntegrationPolicyInputs, IntegrationPolicyState],
) (infer.UpdateResponse[IntegrationPolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.UpdateResponse[IntegrationPolicyState]{}, err
	}

	body, err := buildIntegrationPolicyBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[IntegrationPolicyState]{}, err
	}
	if req.Inputs.Force != nil && *req.Inputs.Force {
		body["force"] = true
	}

	if err := fleetClient.PutJSON(ctx, "/api/fleet/package_policies/"+req.ID, body, nil); err != nil {
		return infer.UpdateResponse[IntegrationPolicyState]{},
			fmt.Errorf("failed to update integration policy %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[IntegrationPolicyState]{
		Output: IntegrationPolicyState{
			IntegrationPolicyInputs: req.Inputs,
			PolicyID:                req.ID,
		},
	}, nil
}

// Delete ...
func (r *IntegrationPolicy) Delete(
	ctx context.Context, req infer.DeleteRequest[IntegrationPolicyState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	path := "/api/fleet/package_policies/" + req.State.PolicyID
	if req.State.Force != nil && *req.State.Force {
		path += "?force=true"
	}

	if err := fleetClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildIntegrationPolicyBody(inputs IntegrationPolicyInputs) (map[string]any, error) {
	body := map[string]any{
		"name":      inputs.Name,
		"policy_id": inputs.AgentPolicyID,
		"package": map[string]any{
			"name":    inputs.IntegrationName,
			"version": inputs.IntegrationVersion,
		},
	}

	if inputs.Namespace != nil {
		body["namespace"] = *inputs.Namespace
	}
	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if inputs.Input != nil {
		var inputCfg any
		if err := json.Unmarshal([]byte(*inputs.Input), &inputCfg); err != nil {
			return nil, fmt.Errorf("invalid input JSON: %w", err)
		}
		body["inputs"] = inputCfg
	}
	if inputs.Vars != nil {
		var vars any
		if err := json.Unmarshal([]byte(*inputs.Vars), &vars); err != nil {
			return nil, fmt.Errorf("invalid vars JSON: %w", err)
		}
		body["vars"] = vars
	}
	if inputs.Force != nil && *inputs.Force {
		body["force"] = true
	}

	return body, nil
}
