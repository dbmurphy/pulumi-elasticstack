package synthetics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Monitor manages a Kibana synthetics monitor via the Synthetics API.
type Monitor struct{}

// MonitorInputs ...
type MonitorInputs struct {
	Name             string   `pulumi:"name"`
	MonitorType      string   `pulumi:"monitorType"`
	Schedule         int      `pulumi:"schedule"`
	SpaceID          *string  `pulumi:"spaceId,optional"`
	Locations        []string `pulumi:"locations,optional"`
	PrivateLocations []string `pulumi:"privateLocations,optional"`
	Enabled          *bool    `pulumi:"enabled,optional"`
	Tags             []string `pulumi:"tags,optional"`
	Alert            *string  `pulumi:"alert,optional"`
	RetestOnFailure  *bool    `pulumi:"retestOnFailure,optional"`
	Config           *string  `pulumi:"config,optional"`
	AdoptOnCreate    bool     `pulumi:"adoptOnCreate,optional"`
}

// MonitorState ...
type MonitorState struct {
	MonitorInputs

	// Outputs
	MonitorID string `pulumi:"monitorId"`
}

var (
	_ infer.CustomDelete[MonitorState]                = (*Monitor)(nil)
	_ infer.CustomRead[MonitorInputs, MonitorState]   = (*Monitor)(nil)
	_ infer.CustomUpdate[MonitorInputs, MonitorState] = (*Monitor)(nil)
)

// Annotate ...
func (r *Monitor) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana synthetics monitor for uptime monitoring.")
	a.SetToken("kibana", "Monitor")
}

// Annotate ...
func (i *MonitorInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the monitor.")
	a.Describe(&i.MonitorType, "The monitor type: 'http', 'tcp', 'icmp', or 'browser'.")
	a.Describe(&i.Schedule, "The monitor check interval in minutes.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.Locations, "List of public location IDs where the monitor runs.")
	a.Describe(&i.PrivateLocations, "List of private location names where the monitor runs.")
	a.Describe(&i.Enabled, "Whether the monitor is enabled.")
	a.Describe(&i.Tags, "Tags for the monitor.")
	a.Describe(&i.Alert, "Alert configuration as a JSON string.")
	a.Describe(&i.RetestOnFailure, "Whether to retest on failure before triggering an alert.")
	a.Describe(&i.Config, "Type-specific monitor configuration as a JSON string (http/tcp/icmp/browser settings).")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing monitor into Pulumi state on create.")
	a.SetDefault(&i.Enabled, true)
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Monitor) Create(
	ctx context.Context, req infer.CreateRequest[MonitorInputs],
) (infer.CreateResponse[MonitorState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[MonitorState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildMonitorBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[MonitorState]{}, err
	}

	var result struct {
		ID string `json:"id"`
	}

	path := clients.SpacePath(spaceID, "/api/synthetics/monitors")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[MonitorState]{},
			fmt.Errorf("failed to create synthetics monitor %s: %w", req.Inputs.Name, err)
	}

	return infer.CreateResponse[MonitorState]{
		ID: result.ID,
		Output: MonitorState{
			MonitorInputs: req.Inputs,
			MonitorID:     result.ID,
		},
	}, nil
}

// Read ...
func (r *Monitor) Read(
	ctx context.Context,
	req infer.ReadRequest[MonitorInputs, MonitorState],
) (infer.ReadResponse[MonitorInputs, MonitorState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[MonitorInputs, MonitorState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/synthetics/monitors/"+req.ID)

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[MonitorInputs, MonitorState]{}, err
	}
	if !exists {
		return infer.ReadResponse[MonitorInputs, MonitorState]{ID: ""}, nil
	}

	return infer.ReadResponse[MonitorInputs, MonitorState](req), nil
}

// Update ...
func (r *Monitor) Update(
	ctx context.Context,
	req infer.UpdateRequest[MonitorInputs, MonitorState],
) (infer.UpdateResponse[MonitorState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[MonitorState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildMonitorBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[MonitorState]{}, err
	}
	path := clients.SpacePath(spaceID, "/api/synthetics/monitors/"+req.ID)

	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[MonitorState]{},
			fmt.Errorf("failed to update synthetics monitor %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[MonitorState]{
		Output: MonitorState{
			MonitorInputs: req.Inputs,
			MonitorID:     req.ID,
		},
	}, nil
}

// Delete ...
func (r *Monitor) Delete(
	ctx context.Context, req infer.DeleteRequest[MonitorState],
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
	path := clients.SpacePath(spaceID, "/api/synthetics/monitors/"+req.State.MonitorID)

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildMonitorBody(inputs MonitorInputs) (map[string]any, error) {
	body := map[string]any{
		"type": inputs.MonitorType,
		"name": inputs.Name,
		"schedule": map[string]any{
			"number": fmt.Sprintf("%d", inputs.Schedule),
			"unit":   "m",
		},
	}

	if len(inputs.Locations) > 0 {
		body["locations"] = inputs.Locations
	}
	if len(inputs.PrivateLocations) > 0 {
		body["private_locations"] = inputs.PrivateLocations
	}
	if inputs.Enabled != nil {
		body["enabled"] = *inputs.Enabled
	}
	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}
	if inputs.Alert != nil {
		var alert any
		if err := json.Unmarshal([]byte(*inputs.Alert), &alert); err != nil {
			return nil, fmt.Errorf("invalid alert JSON: %w", err)
		}
		body["alert"] = alert
	}
	if inputs.RetestOnFailure != nil {
		body["retest_on_failure"] = *inputs.RetestOnFailure
	}

	// Merge type-specific config into the body
	if inputs.Config != nil {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(*inputs.Config), &cfg); err == nil {
			for k, v := range cfg {
				body[k] = v
			}
		}
	}

	return body, nil
}

func resolveSpaceID(spaceID *string) string {
	if spaceID == nil || *spaceID == "" {
		return "default"
	}
	return *spaceID
}
