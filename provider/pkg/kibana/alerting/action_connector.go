package alerting

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// ActionConnector manages a Kibana action connector via the Actions API.
type ActionConnector struct{}

// ActionConnectorInputs ...
type ActionConnectorInputs struct {
	Name            string  `pulumi:"name"`
	ConnectorTypeID string  `pulumi:"connectorTypeId"`
	Config          *string `pulumi:"config,optional"`
	Secrets         *string `pulumi:"secrets,optional"       provider:"secret"`
	SpaceID         *string `pulumi:"spaceId,optional"`
	AdoptOnCreate   bool    `pulumi:"adoptOnCreate,optional"`
}

// ActionConnectorState ...
type ActionConnectorState struct {
	ActionConnectorInputs

	// Outputs
	ConnectorID string `pulumi:"connectorId"`
}

var (
	_ infer.CustomDelete[ActionConnectorState]                        = (*ActionConnector)(nil)
	_ infer.CustomRead[ActionConnectorInputs, ActionConnectorState]   = (*ActionConnector)(nil)
	_ infer.CustomUpdate[ActionConnectorInputs, ActionConnectorState] = (*ActionConnector)(nil)
)

// Annotate ...
func (r *ActionConnector) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana action connector for use with alerting rules.")
	a.SetToken("kibana", "ActionConnector")
}

// Annotate ...
func (i *ActionConnectorInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the connector.")
	a.Describe(&i.ConnectorTypeID, "The connector type ID (e.g. '.slack', '.email', '.webhook').")
	a.Describe(&i.Config, "The connector configuration as a JSON string.")
	a.Describe(&i.Secrets, "The connector secrets as a JSON string.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing connector into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *ActionConnector) Create(
	ctx context.Context,
	req infer.CreateRequest[ActionConnectorInputs],
) (infer.CreateResponse[ActionConnectorState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[ActionConnectorState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := buildConnectorBody(req.Inputs)

	// AdoptOnCreate not supported for connectors (server-generated IDs).

	var result struct {
		ID string `json:"id"`
	}

	path := clients.SpacePath(spaceID, "/api/actions/connector")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[ActionConnectorState]{}, fmt.Errorf(
			"failed to create action connector %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	return infer.CreateResponse[ActionConnectorState]{
		ID: result.ID,
		Output: ActionConnectorState{
			ActionConnectorInputs: req.Inputs,
			ConnectorID:           result.ID,
		},
	}, nil
}

// Read ...
func (r *ActionConnector) Read(
	ctx context.Context,
	req infer.ReadRequest[ActionConnectorInputs, ActionConnectorState],
) (infer.ReadResponse[ActionConnectorInputs, ActionConnectorState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[ActionConnectorInputs, ActionConnectorState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/actions/connector/"+req.ID)

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[ActionConnectorInputs, ActionConnectorState]{}, err
	}
	if !exists {
		return infer.ReadResponse[ActionConnectorInputs, ActionConnectorState]{ID: ""}, nil
	}

	return infer.ReadResponse[ActionConnectorInputs, ActionConnectorState](req), nil
}

// Update ...
func (r *ActionConnector) Update(
	ctx context.Context,
	req infer.UpdateRequest[ActionConnectorInputs, ActionConnectorState],
) (infer.UpdateResponse[ActionConnectorState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[ActionConnectorState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := buildConnectorUpdateBody(req.Inputs)
	path := clients.SpacePath(spaceID, "/api/actions/connector/"+req.ID)

	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[ActionConnectorState]{}, fmt.Errorf(
			"failed to update action connector %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[ActionConnectorState]{
		Output: ActionConnectorState{
			ActionConnectorInputs: req.Inputs,
			ConnectorID:           req.ID,
		},
	}, nil
}

// Delete ...
func (r *ActionConnector) Delete(
	ctx context.Context,
	req infer.DeleteRequest[ActionConnectorState],
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
	path := clients.SpacePath(spaceID, "/api/actions/connector/"+req.State.ConnectorID)

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildConnectorBody(inputs ActionConnectorInputs) map[string]any {
	body := map[string]any{
		"name":              inputs.Name,
		"connector_type_id": inputs.ConnectorTypeID,
	}

	if inputs.Config != nil {
		var cfg any
		if err := json.Unmarshal([]byte(*inputs.Config), &cfg); err == nil {
			body["config"] = cfg
		}
	}
	if inputs.Secrets != nil {
		var secrets any
		if err := json.Unmarshal([]byte(*inputs.Secrets), &secrets); err == nil {
			body["secrets"] = secrets
		}
	}

	return body
}

func buildConnectorUpdateBody(inputs ActionConnectorInputs) map[string]any {
	body := map[string]any{
		"name": inputs.Name,
	}

	if inputs.Config != nil {
		var cfg any
		if err := json.Unmarshal([]byte(*inputs.Config), &cfg); err == nil {
			body["config"] = cfg
		}
	}
	if inputs.Secrets != nil {
		var secrets any
		if err := json.Unmarshal([]byte(*inputs.Secrets), &secrets); err == nil {
			body["secrets"] = secrets
		}
	}

	return body
}

func resolveSpaceID(spaceID *string) string {
	if spaceID == nil || *spaceID == "" {
		return "default"
	}
	return *spaceID
}
