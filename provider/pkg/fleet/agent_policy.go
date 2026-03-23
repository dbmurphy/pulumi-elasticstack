package fleet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// AgentPolicy manages a Fleet agent policy via the /api/fleet/agent_policies API.
type AgentPolicy struct{}

// AgentPolicyInputs ...
type AgentPolicyInputs struct {
	Name                string  `pulumi:"name"`
	Namespace           *string `pulumi:"namespace,optional"`
	Description         *string `pulumi:"description,optional"`
	MonitorLogs         *bool   `pulumi:"monitorLogs,optional"`
	MonitorMetrics      *bool   `pulumi:"monitorMetrics,optional"`
	SkipDestroy         *bool   `pulumi:"skipDestroy,optional"`
	DataOutputID        *string `pulumi:"dataOutputId,optional"`
	MonitoringOutputID  *string `pulumi:"monitoringOutputId,optional"`
	FleetServerHostID   *string `pulumi:"fleetServerHostId,optional"`
	AgentFeatures       *string `pulumi:"agentFeatures,optional"`
	IsProtected         *bool   `pulumi:"isProtected,optional"`
	KeepMonitoringAlive *bool   `pulumi:"keepMonitoringAlive,optional"`
	GlobalDataTags      *string `pulumi:"globalDataTags,optional"`
	AdoptOnCreate       bool    `pulumi:"adoptOnCreate,optional"`
}

// AgentPolicyState ...
type AgentPolicyState struct {
	AgentPolicyInputs

	// Outputs
	PolicyID string `pulumi:"policyId"`
}

var (
	_ infer.CustomDelete[AgentPolicyState]                    = (*AgentPolicy)(nil)
	_ infer.CustomRead[AgentPolicyInputs, AgentPolicyState]   = (*AgentPolicy)(nil)
	_ infer.CustomUpdate[AgentPolicyInputs, AgentPolicyState] = (*AgentPolicy)(nil)
)

// Annotate ...
func (r *AgentPolicy) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Fleet agent policy.")
	a.SetToken("fleet", "AgentPolicy")
}

// Annotate ...
func (i *AgentPolicyInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the agent policy.")
	a.Describe(&i.Namespace, "The namespace for the agent policy. Defaults to 'default'.")
	a.Describe(&i.Description, "A description for the agent policy.")
	a.Describe(&i.MonitorLogs, "Enable log monitoring for the agent policy.")
	a.Describe(&i.MonitorMetrics, "Enable metrics monitoring for the agent policy.")
	a.Describe(&i.SkipDestroy, "If true, the policy will not be deleted when the resource is destroyed.")
	a.Describe(&i.DataOutputID, "The ID of the output to use for data.")
	a.Describe(&i.MonitoringOutputID, "The ID of the output to use for monitoring.")
	a.Describe(&i.FleetServerHostID, "The ID of the Fleet Server host.")
	a.Describe(&i.AgentFeatures, "JSON array of agent features.")
	a.Describe(&i.IsProtected, "Whether the agent policy is protected.")
	a.Describe(&i.KeepMonitoringAlive, "Whether to keep monitoring alive when the policy has no agents.")
	a.Describe(&i.GlobalDataTags, "JSON array of global data tags.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing agent policy into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *AgentPolicy) Create(
	ctx context.Context, req infer.CreateRequest[AgentPolicyInputs],
) (infer.CreateResponse[AgentPolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.CreateResponse[AgentPolicyState]{}, err
	}

	body, err := buildAgentPolicyBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[AgentPolicyState]{}, err
	}

	var result struct {
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}

	if err := fleetClient.PostJSON(ctx, "/api/fleet/agent_policies", body, &result); err != nil {
		return infer.CreateResponse[AgentPolicyState]{},
			fmt.Errorf("failed to create agent policy %s: %w", req.Inputs.Name, err)
	}

	return infer.CreateResponse[AgentPolicyState]{
		ID: result.Item.ID,
		Output: AgentPolicyState{
			AgentPolicyInputs: req.Inputs,
			PolicyID:          result.Item.ID,
		},
	}, nil
}

// Read ...
func (r *AgentPolicy) Read(
	ctx context.Context, req infer.ReadRequest[AgentPolicyInputs, AgentPolicyState],
) (infer.ReadResponse[AgentPolicyInputs, AgentPolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.ReadResponse[AgentPolicyInputs, AgentPolicyState]{}, err
	}

	exists, err := fleetClient.Exists(ctx, "/api/fleet/agent_policies/"+req.ID)
	if err != nil {
		return infer.ReadResponse[AgentPolicyInputs, AgentPolicyState]{}, err
	}
	if !exists {
		return infer.ReadResponse[AgentPolicyInputs, AgentPolicyState]{ID: ""}, nil
	}

	return infer.ReadResponse[AgentPolicyInputs, AgentPolicyState](req), nil
}

// Update ...
func (r *AgentPolicy) Update(
	ctx context.Context, req infer.UpdateRequest[AgentPolicyInputs, AgentPolicyState],
) (infer.UpdateResponse[AgentPolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.UpdateResponse[AgentPolicyState]{}, err
	}

	body, err := buildAgentPolicyBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[AgentPolicyState]{}, err
	}
	if err := fleetClient.PutJSON(ctx, "/api/fleet/agent_policies/"+req.ID, body, nil); err != nil {
		return infer.UpdateResponse[AgentPolicyState]{}, fmt.Errorf("failed to update agent policy %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[AgentPolicyState]{
		Output: AgentPolicyState{
			AgentPolicyInputs: req.Inputs,
			PolicyID:          req.ID,
		},
	}, nil
}

// Delete ...
func (r *AgentPolicy) Delete(
	ctx context.Context, req infer.DeleteRequest[AgentPolicyState],
) (infer.DeleteResponse, error) {
	if req.State.SkipDestroy != nil && *req.State.SkipDestroy {
		return infer.DeleteResponse{}, nil
	}

	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	deleteBody := map[string]any{
		"agentPolicyId": req.State.PolicyID,
	}
	if err := fleetClient.PostJSON(ctx, "/api/fleet/agent_policies/delete", deleteBody, nil); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildAgentPolicyBody(inputs AgentPolicyInputs) (map[string]any, error) {
	body := map[string]any{
		"name": inputs.Name,
	}

	if inputs.Namespace != nil {
		body["namespace"] = *inputs.Namespace
	}
	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if inputs.MonitorLogs != nil {
		body["monitoring_enabled"] = buildMonitoringEnabled(inputs.MonitorLogs, inputs.MonitorMetrics)
	} else if inputs.MonitorMetrics != nil {
		body["monitoring_enabled"] = buildMonitoringEnabled(inputs.MonitorLogs, inputs.MonitorMetrics)
	}
	if inputs.DataOutputID != nil {
		body["data_output_id"] = *inputs.DataOutputID
	}
	if inputs.MonitoringOutputID != nil {
		body["monitoring_output_id"] = *inputs.MonitoringOutputID
	}
	if inputs.FleetServerHostID != nil {
		body["fleet_server_host_id"] = *inputs.FleetServerHostID
	}
	if inputs.AgentFeatures != nil {
		var features any
		if err := json.Unmarshal([]byte(*inputs.AgentFeatures), &features); err != nil {
			return nil, fmt.Errorf("invalid agent_features JSON: %w", err)
		}
		body["agent_features"] = features
	}
	if inputs.IsProtected != nil {
		body["is_protected"] = *inputs.IsProtected
	}
	if inputs.KeepMonitoringAlive != nil {
		body["keep_monitoring_alive"] = *inputs.KeepMonitoringAlive
	}
	if inputs.GlobalDataTags != nil {
		var tags any
		if err := json.Unmarshal([]byte(*inputs.GlobalDataTags), &tags); err != nil {
			return nil, fmt.Errorf("invalid global_data_tags JSON: %w", err)
		}
		body["global_data_tags"] = tags
	}

	return body, nil
}

func buildMonitoringEnabled(logs, metrics *bool) []string {
	var enabled []string
	if logs != nil && *logs {
		enabled = append(enabled, "logs")
	}
	if metrics != nil && *metrics {
		enabled = append(enabled, "metrics")
	}
	return enabled
}
