package dashboard

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Dashboard manages a Kibana dashboard via the experimental Dashboards API.
type Dashboard struct{}

// Inputs ...
type Inputs struct {
	Body          string  `pulumi:"body"`
	SpaceID       *string `pulumi:"spaceID,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// State ...
type State struct {
	Inputs

	// Computed outputs
	DashboardID string `pulumi:"dashboardId"`
}

var (
	_ infer.CustomDelete[State]         = (*Dashboard)(nil)
	_ infer.CustomRead[Inputs, State]   = (*Dashboard)(nil)
	_ infer.CustomUpdate[Inputs, State] = (*Dashboard)(nil)
)

// Annotate ...
func (r *Dashboard) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana dashboard (experimental API).")
	a.SetToken("kibana", "Dashboard")
}

// Annotate ...
func (i *Inputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Body, "The dashboard definition as JSON.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.AdoptOnCreate, "If true and the dashboard already exists, adopt it into state instead of failing.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Dashboard) Create(
	ctx context.Context,
	req infer.CreateRequest[Inputs],
) (infer.CreateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	spaceID := derefString(req.Inputs.SpaceID)

	// Parse the body to extract an ID if present
	var bodyMap map[string]any
	if err := json.Unmarshal([]byte(req.Inputs.Body), &bodyMap); err != nil {
		return infer.CreateResponse[State]{}, fmt.Errorf("failed to parse dashboard body JSON: %w", err)
	}

	// Determine the dashboard ID from the body, if provided
	dashboardID := ""
	if id, ok := bodyMap["id"].(string); ok && id != "" {
		dashboardID = id
	}

	var path string
	if dashboardID != "" {
		path = clients.SpacePath(spaceID, "/api/dashboards/dashboard/"+dashboardID)
	} else {
		path = clients.SpacePath(spaceID, "/api/dashboards/dashboard")
	}

	// adoptOnCreate: check if the dashboard already exists
	if req.Inputs.AdoptOnCreate && dashboardID != "" {
		existsPath := clients.SpacePath(spaceID, "/api/dashboards/dashboard/"+dashboardID)
		exists, err := kbClient.Exists(ctx, existsPath)
		if err != nil {
			return infer.CreateResponse[State]{}, fmt.Errorf("failed to check if dashboard exists: %w", err)
		}
		if exists {
			return infer.CreateResponse[State]{
				ID: dashboardID,
				Output: State{
					Inputs:      req.Inputs,
					DashboardID: dashboardID,
				},
			}, nil
		}
	}

	var result struct {
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}

	if err := kbClient.PostJSON(ctx, path, bodyMap, &result); err != nil {
		return infer.CreateResponse[State]{}, fmt.Errorf("failed to create dashboard: %w", err)
	}

	resultID := result.Item.ID
	if resultID == "" {
		resultID = dashboardID
	}

	return infer.CreateResponse[State]{
		ID: resultID,
		Output: State{
			Inputs:      req.Inputs,
			DashboardID: resultID,
		},
	}, nil
}

// Read ...
func (r *Dashboard) Read(
	ctx context.Context,
	req infer.ReadRequest[Inputs, State],
) (infer.ReadResponse[Inputs, State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}

	spaceID := derefString(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/dashboards/dashboard/"+req.ID)

	var result map[string]json.RawMessage
	if err := kbClient.GetJSON(ctx, path, &result); err != nil {
		if clients.IsNotFound(err) {
			return infer.ReadResponse[Inputs, State]{ID: ""}, nil
		}
		return infer.ReadResponse[Inputs, State]{}, err
	}

	// Re-serialize the full response as the body for state
	bodyBytes, err := json.Marshal(result)
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, fmt.Errorf(
			"failed to serialize dashboard response: %w",
			err,
		)
	}

	inputs := req.Inputs
	inputs.Body = string(bodyBytes)

	dashboardID := req.ID
	if raw, ok := result["item"]; ok {
		var item struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &item) == nil && item.ID != "" {
			dashboardID = item.ID
		}
	}

	state := State{
		Inputs:      inputs,
		DashboardID: dashboardID,
	}

	return infer.ReadResponse[Inputs, State]{
		ID:     req.ID,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update ...
func (r *Dashboard) Update(
	ctx context.Context,
	req infer.UpdateRequest[Inputs, State],
) (infer.UpdateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	spaceID := derefString(req.Inputs.SpaceID)
	path := clients.SpacePath(spaceID, "/api/dashboards/dashboard/"+req.ID)

	var bodyMap map[string]any
	if err := json.Unmarshal([]byte(req.Inputs.Body), &bodyMap); err != nil {
		return infer.UpdateResponse[State]{}, fmt.Errorf("failed to parse dashboard body JSON: %w", err)
	}

	if err := kbClient.PostJSON(ctx, path, bodyMap, nil); err != nil {
		return infer.UpdateResponse[State]{}, fmt.Errorf("failed to update dashboard %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[State]{
		Output: State{
			Inputs:      req.Inputs,
			DashboardID: req.ID,
		},
	}, nil
}

// Delete ...
func (r *Dashboard) Delete(ctx context.Context, req infer.DeleteRequest[State]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := derefString(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/dashboards/dashboard/"+req.State.DashboardID)

	if err := kbClient.Delete(ctx, path); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete dashboard %s: %w", req.State.DashboardID, err)
	}

	return infer.DeleteResponse{}, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
