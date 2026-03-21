package fleet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Output manages a Fleet output via the /api/fleet/outputs API.
type Output struct{}

// OutputInputs ...
type OutputInputs struct {
	Name                string   `pulumi:"name"`
	OutputType          string   `pulumi:"outputType"`
	DefaultIntegrations *bool    `pulumi:"defaultIntegrations,optional"`
	DefaultMonitoring   *bool    `pulumi:"defaultMonitoring,optional"`
	Hosts               []string `pulumi:"hosts,optional"`
	ConfigYaml          *string  `pulumi:"configYaml,optional"`
	Ssl                 *string  `pulumi:"ssl,optional"`
	AdoptOnCreate       bool     `pulumi:"adoptOnCreate,optional"`
}

// OutputState ...
type OutputState struct {
	OutputInputs

	// Outputs
	OutputID string `pulumi:"outputId"`
}

var (
	_ infer.CustomDelete[OutputState]               = (*Output)(nil)
	_ infer.CustomRead[OutputInputs, OutputState]   = (*Output)(nil)
	_ infer.CustomUpdate[OutputInputs, OutputState] = (*Output)(nil)
)

// Annotate ...
func (r *Output) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Fleet output configuration.")
	a.SetToken("fleet", "Output")
}

// Annotate ...
func (i *OutputInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the output.")
	a.Describe(&i.OutputType, "The output type: elasticsearch, logstash, kafka, or remote_elasticsearch.")
	a.Describe(&i.DefaultIntegrations, "Whether this output is the default for agent integrations.")
	a.Describe(&i.DefaultMonitoring, "Whether this output is the default for agent monitoring.")
	a.Describe(&i.Hosts, "List of hosts for the output.")
	a.Describe(&i.ConfigYaml, "Additional YAML configuration for the output.")
	a.Describe(&i.Ssl, "JSON object of TLS/SSL configuration for the output.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing output into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Output) Create(
	ctx context.Context, req infer.CreateRequest[OutputInputs],
) (infer.CreateResponse[OutputState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.CreateResponse[OutputState]{}, err
	}

	body, err := buildOutputBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[OutputState]{}, err
	}

	var result struct {
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}

	if err := fleetClient.PostJSON(ctx, "/api/fleet/outputs", body, &result); err != nil {
		return infer.CreateResponse[OutputState]{}, fmt.Errorf("failed to create output %s: %w", req.Inputs.Name, err)
	}

	return infer.CreateResponse[OutputState]{
		ID: result.Item.ID,
		Output: OutputState{
			OutputInputs: req.Inputs,
			OutputID:     result.Item.ID,
		},
	}, nil
}

// Read ...
func (r *Output) Read(
	ctx context.Context, req infer.ReadRequest[OutputInputs, OutputState],
) (infer.ReadResponse[OutputInputs, OutputState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.ReadResponse[OutputInputs, OutputState]{}, err
	}

	exists, err := fleetClient.Exists(ctx, "/api/fleet/outputs/"+req.ID)
	if err != nil {
		return infer.ReadResponse[OutputInputs, OutputState]{}, err
	}
	if !exists {
		return infer.ReadResponse[OutputInputs, OutputState]{ID: ""}, nil
	}

	return infer.ReadResponse[OutputInputs, OutputState](req), nil
}

// Update ...
func (r *Output) Update(
	ctx context.Context, req infer.UpdateRequest[OutputInputs, OutputState],
) (infer.UpdateResponse[OutputState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.UpdateResponse[OutputState]{}, err
	}

	body, err := buildOutputBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[OutputState]{}, err
	}
	if err := fleetClient.PutJSON(ctx, "/api/fleet/outputs/"+req.ID, body, nil); err != nil {
		return infer.UpdateResponse[OutputState]{}, fmt.Errorf("failed to update output %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[OutputState]{
		Output: OutputState{
			OutputInputs: req.Inputs,
			OutputID:     req.ID,
		},
	}, nil
}

// Delete ...
func (r *Output) Delete(ctx context.Context, req infer.DeleteRequest[OutputState]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := fleetClient.Delete(ctx, "/api/fleet/outputs/"+req.State.OutputID); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildOutputBody(inputs OutputInputs) (map[string]any, error) {
	body := map[string]any{
		"name": inputs.Name,
		"type": inputs.OutputType,
	}

	if inputs.DefaultIntegrations != nil {
		body["is_default"] = *inputs.DefaultIntegrations
	}
	if inputs.DefaultMonitoring != nil {
		body["is_default_monitoring"] = *inputs.DefaultMonitoring
	}
	if len(inputs.Hosts) > 0 {
		body["hosts"] = inputs.Hosts
	}
	if inputs.ConfigYaml != nil {
		body["config_yaml"] = *inputs.ConfigYaml
	}
	if inputs.Ssl != nil {
		var ssl any
		if err := json.Unmarshal([]byte(*inputs.Ssl), &ssl); err != nil {
			return nil, fmt.Errorf("invalid ssl JSON: %w", err)
		}
		body["ssl"] = ssl
	}

	return body, nil
}
