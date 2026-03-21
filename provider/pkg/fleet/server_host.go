package fleet

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// ServerHost manages a Fleet Server host via the /api/fleet/fleet_server_hosts API.
type ServerHost struct{}

// ServerHostInputs ...
type ServerHostInputs struct {
	Name          string   `pulumi:"name"`
	Hosts         []string `pulumi:"hosts"`
	IsDefault     *bool    `pulumi:"isDefault,optional"`
	ProxyID       *string  `pulumi:"proxyId,optional"`
	AdoptOnCreate bool     `pulumi:"adoptOnCreate,optional"`
}

// ServerHostState ...
type ServerHostState struct {
	ServerHostInputs

	// Outputs
	HostID string `pulumi:"hostId"`
}

var (
	_ infer.CustomDelete[ServerHostState]                   = (*ServerHost)(nil)
	_ infer.CustomRead[ServerHostInputs, ServerHostState]   = (*ServerHost)(nil)
	_ infer.CustomUpdate[ServerHostInputs, ServerHostState] = (*ServerHost)(nil)
)

// Annotate ...
func (r *ServerHost) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Fleet Server host configuration.")
	a.SetToken("fleet", "ServerHost")
}

// Annotate ...
func (i *ServerHostInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the Fleet Server host.")
	a.Describe(&i.Hosts, "List of host URLs for the Fleet Server.")
	a.Describe(&i.IsDefault, "Whether this is the default Fleet Server host.")
	a.Describe(&i.ProxyID, "The ID of the proxy to use for this Fleet Server host.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing Fleet Server host into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *ServerHost) Create(
	ctx context.Context, req infer.CreateRequest[ServerHostInputs],
) (infer.CreateResponse[ServerHostState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.CreateResponse[ServerHostState]{}, err
	}

	body := buildServerHostBody(req.Inputs)

	var result struct {
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}

	if err := fleetClient.PostJSON(ctx, "/api/fleet/fleet_server_hosts", body, &result); err != nil {
		return infer.CreateResponse[ServerHostState]{},
			fmt.Errorf("failed to create fleet server host %s: %w", req.Inputs.Name, err)
	}

	return infer.CreateResponse[ServerHostState]{
		ID: result.Item.ID,
		Output: ServerHostState{
			ServerHostInputs: req.Inputs,
			HostID:           result.Item.ID,
		},
	}, nil
}

// Read ...
func (r *ServerHost) Read(
	ctx context.Context, req infer.ReadRequest[ServerHostInputs, ServerHostState],
) (infer.ReadResponse[ServerHostInputs, ServerHostState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.ReadResponse[ServerHostInputs, ServerHostState]{}, err
	}

	exists, err := fleetClient.Exists(ctx, "/api/fleet/fleet_server_hosts/"+req.ID)
	if err != nil {
		return infer.ReadResponse[ServerHostInputs, ServerHostState]{}, err
	}
	if !exists {
		return infer.ReadResponse[ServerHostInputs, ServerHostState]{ID: ""}, nil
	}

	return infer.ReadResponse[ServerHostInputs, ServerHostState](req), nil
}

// Update ...
func (r *ServerHost) Update(
	ctx context.Context, req infer.UpdateRequest[ServerHostInputs, ServerHostState],
) (infer.UpdateResponse[ServerHostState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.UpdateResponse[ServerHostState]{}, err
	}

	body := buildServerHostBody(req.Inputs)
	if err := fleetClient.PutJSON(ctx, "/api/fleet/fleet_server_hosts/"+req.ID, body, nil); err != nil {
		return infer.UpdateResponse[ServerHostState]{}, fmt.Errorf(
			"failed to update fleet server host %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[ServerHostState]{
		Output: ServerHostState{
			ServerHostInputs: req.Inputs,
			HostID:           req.ID,
		},
	}, nil
}

// Delete ...
func (r *ServerHost) Delete(
	ctx context.Context, req infer.DeleteRequest[ServerHostState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := fleetClient.Delete(ctx, "/api/fleet/fleet_server_hosts/"+req.State.HostID); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildServerHostBody(inputs ServerHostInputs) map[string]any {
	body := map[string]any{
		"name":      inputs.Name,
		"host_urls": inputs.Hosts,
	}

	if inputs.IsDefault != nil {
		body["is_default"] = *inputs.IsDefault
	}
	if inputs.ProxyID != nil {
		body["proxy_id"] = *inputs.ProxyID
	}

	return body
}
