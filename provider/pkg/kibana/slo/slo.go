package slo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Slo manages a Kibana SLO (Service Level Objective) via the Observability SLOs API.
type Slo struct{}

// Inputs ...
type Inputs struct {
	Name            string   `pulumi:"name"`
	Description     *string  `pulumi:"description,optional"`
	Indicator       string   `pulumi:"indicator"`
	TimeWindow      string   `pulumi:"timeWindow"`
	BudgetingMethod string   `pulumi:"budgetingMethod"`
	Objective       string   `pulumi:"objective"`
	Settings        *string  `pulumi:"settings,optional"`
	Tags            []string `pulumi:"tags,optional"`
	GroupBy         *string  `pulumi:"groupBy,optional"`
	SpaceID         *string  `pulumi:"spaceID,optional"`
	AdoptOnCreate   bool     `pulumi:"adoptOnCreate,optional"`
}

// State ...
type State struct {
	Inputs

	// Computed outputs
	SloID string `pulumi:"sloId"`
}

var (
	_ infer.CustomDelete[State]         = (*Slo)(nil)
	_ infer.CustomRead[Inputs, State]   = (*Slo)(nil)
	_ infer.CustomUpdate[Inputs, State] = (*Slo)(nil)
)

// Annotate ...
func (r *Slo) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana SLO (Service Level Objective).")
	a.SetToken("kibana", "Slo")
}

// Annotate ...
func (i *Inputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the SLO.")
	a.Describe(&i.Description, "A description of the SLO.")
	a.Describe(&i.Indicator, "The SLI indicator definition as JSON.")
	a.Describe(&i.TimeWindow, "The time window configuration as JSON.")
	a.Describe(&i.BudgetingMethod, "The budgeting method: 'occurrences' or 'timeslices'.")
	a.Describe(&i.Objective, "The SLO objective/target as JSON.")
	a.Describe(&i.Settings, "Additional SLO settings as JSON.")
	a.Describe(&i.Tags, "Tags for the SLO.")
	a.Describe(&i.GroupBy, "Group-by field for the SLO.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.AdoptOnCreate, "If true and the SLO already exists, adopt it into state instead of failing.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Slo) Create(ctx context.Context, req infer.CreateRequest[Inputs]) (infer.CreateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	spaceID := derefString(req.Inputs.SpaceID)
	path := clients.SpacePath(spaceID, "/api/observability/slos")

	body, err := buildSloBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[State]{}, fmt.Errorf("failed to create SLO: %w", err)
	}

	return infer.CreateResponse[State]{
		ID: result.ID,
		Output: State{
			Inputs: req.Inputs,
			SloID:  result.ID,
		},
	}, nil
}

// Read ...
func (r *Slo) Read(
	ctx context.Context, req infer.ReadRequest[Inputs, State],
) (infer.ReadResponse[Inputs, State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}

	spaceID := derefString(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/observability/slos/"+req.ID)

	var result map[string]json.RawMessage
	if err := kbClient.GetJSON(ctx, path, &result); err != nil {
		if clients.IsNotFound(err) {
			return infer.ReadResponse[Inputs, State]{ID: ""}, nil
		}
		return infer.ReadResponse[Inputs, State]{}, err
	}

	inputs := req.Inputs

	if raw, ok := result["name"]; ok {
		var name string
		if json.Unmarshal(raw, &name) == nil {
			inputs.Name = name
		}
	}
	if raw, ok := result["description"]; ok {
		var desc string
		if json.Unmarshal(raw, &desc) == nil {
			inputs.Description = &desc
		}
	}
	if raw, ok := result["indicator"]; ok {
		inputs.Indicator = string(raw)
	}
	if raw, ok := result["timeWindow"]; ok {
		inputs.TimeWindow = string(raw)
	}
	if raw, ok := result["budgetingMethod"]; ok {
		var bm string
		if json.Unmarshal(raw, &bm) == nil {
			inputs.BudgetingMethod = bm
		}
	}
	if raw, ok := result["objective"]; ok {
		inputs.Objective = string(raw)
	}
	if raw, ok := result["settings"]; ok && string(raw) != "null" {
		s := string(raw)
		inputs.Settings = &s
	}
	if raw, ok := result["tags"]; ok {
		var tags []string
		if json.Unmarshal(raw, &tags) == nil {
			inputs.Tags = tags
		}
	}
	if raw, ok := result["groupBy"]; ok {
		var gb string
		if json.Unmarshal(raw, &gb) == nil {
			inputs.GroupBy = &gb
		}
	}

	sloID := req.ID
	if raw, ok := result["id"]; ok {
		var id string
		if json.Unmarshal(raw, &id) == nil {
			sloID = id
		}
	}

	state := State{
		Inputs: inputs,
		SloID:  sloID,
	}

	return infer.ReadResponse[Inputs, State]{
		ID:     req.ID,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update ...
func (r *Slo) Update(
	ctx context.Context, req infer.UpdateRequest[Inputs, State],
) (infer.UpdateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	spaceID := derefString(req.Inputs.SpaceID)
	path := clients.SpacePath(spaceID, "/api/observability/slos/"+req.ID)

	body, err := buildSloBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[State]{}, fmt.Errorf("failed to update SLO %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[State]{
		Output: State{
			Inputs: req.Inputs,
			SloID:  req.ID,
		},
	}, nil
}

// Delete ...
func (r *Slo) Delete(ctx context.Context, req infer.DeleteRequest[State]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := derefString(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/observability/slos/"+req.State.SloID)

	if err := kbClient.Delete(ctx, path); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete SLO %s: %w", req.State.SloID, err)
	}

	return infer.DeleteResponse{}, nil
}

func buildSloBody(inputs Inputs) (map[string]any, error) {
	body := map[string]any{
		"name":            inputs.Name,
		"budgetingMethod": inputs.BudgetingMethod,
	}

	// Indicator is required JSON
	var indicator any
	if err := json.Unmarshal([]byte(inputs.Indicator), &indicator); err != nil {
		return nil, fmt.Errorf("invalid indicator JSON: %w", err)
	}
	body["indicator"] = indicator

	// TimeWindow is required JSON
	var timeWindow any
	if err := json.Unmarshal([]byte(inputs.TimeWindow), &timeWindow); err != nil {
		return nil, fmt.Errorf("invalid timeWindow JSON: %w", err)
	}
	body["timeWindow"] = timeWindow

	// Objective is required JSON
	var objective any
	if err := json.Unmarshal([]byte(inputs.Objective), &objective); err != nil {
		return nil, fmt.Errorf("invalid objective JSON: %w", err)
	}
	body["objective"] = objective

	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if inputs.Settings != nil {
		var settings any
		if err := json.Unmarshal([]byte(*inputs.Settings), &settings); err != nil {
			return nil, fmt.Errorf("invalid settings JSON: %w", err)
		}
		body["settings"] = settings
	}
	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}
	if inputs.GroupBy != nil {
		body["groupBy"] = *inputs.GroupBy
	}

	return body, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
