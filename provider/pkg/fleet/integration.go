package fleet

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Integration manages a Fleet integration package installation via the EPM API.
type Integration struct{}

// IntegrationInputs ...
type IntegrationInputs struct {
	Name          string `pulumi:"name"`
	Version       string `pulumi:"version"`
	Force         *bool  `pulumi:"force,optional"`
	SkipDestroy   *bool  `pulumi:"skipDestroy,optional"`
	AdoptOnCreate bool   `pulumi:"adoptOnCreate,optional"`
}

// IntegrationState ...
type IntegrationState struct {
	IntegrationInputs

	// Outputs
	InstalledVersion string `pulumi:"installedVersion"`
}

var (
	_ infer.CustomDelete[IntegrationState]                    = (*Integration)(nil)
	_ infer.CustomRead[IntegrationInputs, IntegrationState]   = (*Integration)(nil)
	_ infer.CustomUpdate[IntegrationInputs, IntegrationState] = (*Integration)(nil)
)

// Annotate ...
func (r *Integration) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Fleet integration package installation.")
	a.SetToken("fleet", "Integration")
}

// Annotate ...
func (i *IntegrationInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The integration package name.")
	a.Describe(&i.Version, "The version of the integration package to install.")
	a.Describe(&i.Force, "Force installation even if the package is already installed.")
	a.Describe(&i.SkipDestroy, "If true, the package will not be uninstalled when the resource is destroyed.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing integration into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Integration) Create(
	ctx context.Context, req infer.CreateRequest[IntegrationInputs],
) (infer.CreateResponse[IntegrationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.CreateResponse[IntegrationState]{}, err
	}

	path := fmt.Sprintf("/api/fleet/epm/packages/%s/%s", req.Inputs.Name, req.Inputs.Version)

	if req.Inputs.AdoptOnCreate {
		readPath := fmt.Sprintf("/api/fleet/epm/packages/%s", req.Inputs.Name)
		exists, err := fleetClient.Exists(ctx, readPath)
		if err != nil {
			return infer.CreateResponse[IntegrationState]{}, err
		}
		if exists {
			// Already installed — adopt it and ensure correct version
			if err := fleetClient.PostJSON(ctx, path, map[string]any{"force": true}, nil); err != nil {
				return infer.CreateResponse[IntegrationState]{},
					fmt.Errorf("failed to update adopted integration %s: %w", req.Inputs.Name, err)
			}
			return infer.CreateResponse[IntegrationState]{
				ID: req.Inputs.Name,
				Output: IntegrationState{
					IntegrationInputs: req.Inputs,
					InstalledVersion:  req.Inputs.Version,
				},
			}, nil
		}
	}

	body := map[string]any{}
	if req.Inputs.Force != nil && *req.Inputs.Force {
		body["force"] = true
	}

	if err := fleetClient.PostJSON(ctx, path, body, nil); err != nil {
		return infer.CreateResponse[IntegrationState]{},
			fmt.Errorf("failed to install integration %s@%s: %w", req.Inputs.Name, req.Inputs.Version, err)
	}

	return infer.CreateResponse[IntegrationState]{
		ID: req.Inputs.Name,
		Output: IntegrationState{
			IntegrationInputs: req.Inputs,
			InstalledVersion:  req.Inputs.Version,
		},
	}, nil
}

// Read ...
func (r *Integration) Read(
	ctx context.Context, req infer.ReadRequest[IntegrationInputs, IntegrationState],
) (infer.ReadResponse[IntegrationInputs, IntegrationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.ReadResponse[IntegrationInputs, IntegrationState]{}, err
	}

	path := fmt.Sprintf("/api/fleet/epm/packages/%s", req.ID)
	exists, err := fleetClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[IntegrationInputs, IntegrationState]{}, err
	}
	if !exists {
		return infer.ReadResponse[IntegrationInputs, IntegrationState]{ID: ""}, nil
	}

	return infer.ReadResponse[IntegrationInputs, IntegrationState](req), nil
}

// Update ...
func (r *Integration) Update(
	ctx context.Context, req infer.UpdateRequest[IntegrationInputs, IntegrationState],
) (infer.UpdateResponse[IntegrationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.UpdateResponse[IntegrationState]{}, err
	}

	// Install the new version (this upgrades the package)
	path := fmt.Sprintf("/api/fleet/epm/packages/%s/%s", req.Inputs.Name, req.Inputs.Version)
	body := map[string]any{}
	if req.Inputs.Force != nil && *req.Inputs.Force {
		body["force"] = true
	}

	if err := fleetClient.PostJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[IntegrationState]{},
			fmt.Errorf("failed to update integration %s to %s: %w", req.Inputs.Name, req.Inputs.Version, err)
	}

	return infer.UpdateResponse[IntegrationState]{
		Output: IntegrationState{
			IntegrationInputs: req.Inputs,
			InstalledVersion:  req.Inputs.Version,
		},
	}, nil
}

// Delete ...
func (r *Integration) Delete(
	ctx context.Context, req infer.DeleteRequest[IntegrationState],
) (infer.DeleteResponse, error) {
	if req.State.SkipDestroy != nil && *req.State.SkipDestroy {
		return infer.DeleteResponse{}, nil
	}

	cfg := infer.GetConfig[provider.Config](ctx)
	fleetClient, err := cfg.FleetClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	path := fmt.Sprintf("/api/fleet/epm/packages/%s/%s?force=true", req.State.Name, req.State.Version)
	if err := fleetClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}
