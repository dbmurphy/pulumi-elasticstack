package apm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// AgentConfiguration manages an APM agent configuration via the Kibana APM settings API.
type AgentConfiguration struct{}

// AgentConfigurationInputs ...
type AgentConfigurationInputs struct {
	ServiceName        string  `pulumi:"serviceName"`
	ServiceEnvironment *string `pulumi:"serviceEnvironment,optional"`
	AgentName          *string `pulumi:"agentName,optional"`
	Settings           string  `pulumi:"settings"`
}

// AgentConfigurationState ...
type AgentConfigurationState struct {
	AgentConfigurationInputs
}

var (
	_ infer.CustomDelete[AgentConfigurationState]                           = (*AgentConfiguration)(nil)
	_ infer.CustomRead[AgentConfigurationInputs, AgentConfigurationState]   = (*AgentConfiguration)(nil)
	_ infer.CustomUpdate[AgentConfigurationInputs, AgentConfigurationState] = (*AgentConfiguration)(nil)
)

// Annotate ...
func (r *AgentConfiguration) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an APM agent configuration.")
	a.SetToken("apm", "AgentConfiguration")
}

// Annotate ...
func (i *AgentConfigurationInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.ServiceName, "The name of the service to configure.")
	a.Describe(&i.ServiceEnvironment, "The environment of the service (e.g. production, staging).")
	a.Describe(&i.AgentName, "The APM agent name (e.g. java, nodejs, python).")
	a.Describe(&i.Settings, "JSON object of agent configuration key-value pairs.")
}

// Create ...
func (r *AgentConfiguration) Create(
	ctx context.Context,
	req infer.CreateRequest[AgentConfigurationInputs],
) (infer.CreateResponse[AgentConfigurationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[AgentConfigurationState]{}, err
	}

	body, err := buildAgentConfigBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[AgentConfigurationState]{},
			fmt.Errorf("failed to build agent configuration body: %w", err)
	}

	if err := kbClient.PutJSON(ctx, "/api/apm/settings/agent-configuration", body, nil); err != nil {
		return infer.CreateResponse[AgentConfigurationState]{},
			fmt.Errorf("failed to create APM agent configuration for service %s: %w",
				req.Inputs.ServiceName, err)
	}

	id := buildAgentConfigID(req.Inputs.ServiceName, req.Inputs.ServiceEnvironment)

	return infer.CreateResponse[AgentConfigurationState]{
		ID:     id,
		Output: AgentConfigurationState{AgentConfigurationInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *AgentConfiguration) Read(
	ctx context.Context,
	req infer.ReadRequest[AgentConfigurationInputs, AgentConfigurationState],
) (infer.ReadResponse[AgentConfigurationInputs, AgentConfigurationState], error) {
	// The APM agent configuration search API doesn't support exact lookup by
	// service name + environment. Return the stored state as-is.
	return infer.ReadResponse[AgentConfigurationInputs, AgentConfigurationState](req), nil
}

// Update ...
func (r *AgentConfiguration) Update(
	ctx context.Context,
	req infer.UpdateRequest[AgentConfigurationInputs, AgentConfigurationState],
) (infer.UpdateResponse[AgentConfigurationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[AgentConfigurationState]{}, err
	}

	body, err := buildAgentConfigBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[AgentConfigurationState]{},
			fmt.Errorf("failed to build agent configuration body: %w", err)
	}

	if err := kbClient.PutJSON(ctx, "/api/apm/settings/agent-configuration", body, nil); err != nil {
		return infer.UpdateResponse[AgentConfigurationState]{},
			fmt.Errorf("failed to update APM agent configuration for service %s: %w",
				req.Inputs.ServiceName, err)
	}

	return infer.UpdateResponse[AgentConfigurationState]{
		Output: AgentConfigurationState{AgentConfigurationInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *AgentConfiguration) Delete(
	ctx context.Context,
	req infer.DeleteRequest[AgentConfigurationState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	body := buildAgentConfigDeleteBody(req.State.ServiceName, req.State.ServiceEnvironment)
	if err := kbClient.DeleteWithBody(ctx, "/api/apm/settings/agent-configuration", body); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf(
			"failed to delete APM agent configuration for service %s: %w",
			req.State.ServiceName,
			err,
		)
	}

	return infer.DeleteResponse{}, nil
}

// buildAgentConfigID constructs the resource ID from service name and optional environment.
func buildAgentConfigID(serviceName string, serviceEnvironment *string) string {
	if serviceEnvironment != nil && *serviceEnvironment != "" {
		return serviceName + "/" + *serviceEnvironment
	}
	return serviceName
}

// buildAgentConfigBody constructs the PUT request body for creating/updating an APM agent configuration.
func buildAgentConfigBody(inputs AgentConfigurationInputs) (map[string]any, error) {
	service := map[string]any{
		"name": inputs.ServiceName,
	}
	if inputs.ServiceEnvironment != nil {
		service["environment"] = *inputs.ServiceEnvironment
	}

	var settings map[string]any
	if err := json.Unmarshal([]byte(inputs.Settings), &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
	}

	body := map[string]any{
		"service":  service,
		"settings": settings,
	}

	if inputs.AgentName != nil {
		body["agent_name"] = *inputs.AgentName
	}

	return body, nil
}

// buildAgentConfigDeleteBody constructs the DELETE request body for removing an APM agent configuration.
func buildAgentConfigDeleteBody(serviceName string, serviceEnvironment *string) map[string]any {
	service := map[string]any{
		"name": serviceName,
	}
	if serviceEnvironment != nil {
		service["environment"] = *serviceEnvironment
	}

	return map[string]any{
		"service": service,
	}
}
