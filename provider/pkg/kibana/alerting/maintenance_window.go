package alerting

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// MaintenanceWindow manages a Kibana maintenance window via the internal Alerting API.
type MaintenanceWindow struct{}

// MaintenanceWindowInputs ...
type MaintenanceWindowInputs struct {
	Title         string  `pulumi:"title"`
	Enabled       *bool   `pulumi:"enabled,optional"`
	Schedule      string  `pulumi:"schedule"`
	ScopedQuery   *string `pulumi:"scopedQuery,optional"`
	SpaceID       *string `pulumi:"spaceId,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// MaintenanceWindowState ...
type MaintenanceWindowState struct {
	MaintenanceWindowInputs

	// Outputs
	WindowID string `pulumi:"windowId"`
}

var (
	_ infer.CustomDelete[MaintenanceWindowState]                          = (*MaintenanceWindow)(nil)
	_ infer.CustomRead[MaintenanceWindowInputs, MaintenanceWindowState]   = (*MaintenanceWindow)(nil)
	_ infer.CustomUpdate[MaintenanceWindowInputs, MaintenanceWindowState] = (*MaintenanceWindow)(nil)
)

// Annotate ...
func (r *MaintenanceWindow) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana maintenance window that suppresses alerting notifications.")
	a.SetToken("kibana", "MaintenanceWindow")
}

// Annotate ...
func (i *MaintenanceWindowInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Title, "The title of the maintenance window.")
	a.Describe(&i.Enabled, "Whether the maintenance window is enabled.")
	a.Describe(&i.Schedule, "The maintenance window schedule as a JSON RRule object.")
	a.Describe(&i.ScopedQuery, "Scoped query filters as a JSON array of filter objects.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing maintenance window into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *MaintenanceWindow) Create(
	ctx context.Context,
	req infer.CreateRequest[MaintenanceWindowInputs],
) (infer.CreateResponse[MaintenanceWindowState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[MaintenanceWindowState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildMaintenanceWindowBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[MaintenanceWindowState]{}, err
	}

	var result struct {
		ID string `json:"id"`
	}

	path := clients.SpacePath(spaceID, "/internal/alerting/rules/maintenance_window")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[MaintenanceWindowState]{}, fmt.Errorf(
			"failed to create maintenance window %s: %w",
			req.Inputs.Title,
			err,
		)
	}

	return infer.CreateResponse[MaintenanceWindowState]{
		ID: result.ID,
		Output: MaintenanceWindowState{
			MaintenanceWindowInputs: req.Inputs,
			WindowID:                result.ID,
		},
	}, nil
}

// Read ...
func (r *MaintenanceWindow) Read(
	ctx context.Context,
	req infer.ReadRequest[MaintenanceWindowInputs, MaintenanceWindowState],
) (infer.ReadResponse[MaintenanceWindowInputs, MaintenanceWindowState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[MaintenanceWindowInputs, MaintenanceWindowState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/internal/alerting/rules/maintenance_window/"+req.ID)

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[MaintenanceWindowInputs, MaintenanceWindowState]{}, err
	}
	if !exists {
		return infer.ReadResponse[MaintenanceWindowInputs, MaintenanceWindowState]{ID: ""}, nil
	}

	return infer.ReadResponse[MaintenanceWindowInputs, MaintenanceWindowState](req), nil
}

// Update ...
func (r *MaintenanceWindow) Update(
	ctx context.Context,
	req infer.UpdateRequest[MaintenanceWindowInputs, MaintenanceWindowState],
) (infer.UpdateResponse[MaintenanceWindowState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[MaintenanceWindowState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildMaintenanceWindowBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[MaintenanceWindowState]{}, err
	}
	path := clients.SpacePath(spaceID, "/internal/alerting/rules/maintenance_window/"+req.ID)

	// Maintenance window update uses POST, not PUT.
	if err := kbClient.PostJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[MaintenanceWindowState]{}, fmt.Errorf(
			"failed to update maintenance window %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[MaintenanceWindowState]{
		Output: MaintenanceWindowState{
			MaintenanceWindowInputs: req.Inputs,
			WindowID:                req.ID,
		},
	}, nil
}

// Delete ...
func (r *MaintenanceWindow) Delete(
	ctx context.Context,
	req infer.DeleteRequest[MaintenanceWindowState],
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
	path := clients.SpacePath(spaceID, "/internal/alerting/rules/maintenance_window/"+req.State.WindowID)

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildMaintenanceWindowBody(inputs MaintenanceWindowInputs) (map[string]any, error) {
	body := map[string]any{
		"title": inputs.Title,
	}

	if inputs.Enabled != nil {
		body["enabled"] = *inputs.Enabled
	}

	var schedule any
	if err := json.Unmarshal([]byte(inputs.Schedule), &schedule); err != nil {
		return nil, fmt.Errorf("invalid schedule JSON: %w", err)
	}
	body["schedule"] = schedule

	if inputs.ScopedQuery != nil {
		var scopedQuery any
		if err := json.Unmarshal([]byte(*inputs.ScopedQuery), &scopedQuery); err != nil {
			return nil, fmt.Errorf("invalid scopedQuery JSON: %w", err)
		}
		body["scoped_query"] = scopedQuery
	}

	return body, nil
}
