package cluster

import (
	"context"
	"encoding/json"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Settings manages Elasticsearch cluster settings via PUT /_cluster/settings.
type Settings struct{}

// SettingsInputs ...
type SettingsInputs struct {
	Persistent    *string `pulumi:"persistent,optional"`
	Transient     *string `pulumi:"transient,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// SettingsState ...
type SettingsState struct {
	SettingsInputs

	// Outputs: the settings as applied by the cluster
	AppliedPersistent string `pulumi:"appliedPersistent"`
	AppliedTransient  string `pulumi:"appliedTransient"`
}

var (
	_ infer.CustomDelete[SettingsState]                 = (*Settings)(nil)
	_ infer.CustomRead[SettingsInputs, SettingsState]   = (*Settings)(nil)
	_ infer.CustomUpdate[SettingsInputs, SettingsState] = (*Settings)(nil)
	_ infer.CustomDiff[SettingsInputs, SettingsState]   = (*Settings)(nil)
)

// Annotate ...
func (r *Settings) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages Elasticsearch cluster-level settings (persistent and transient).")
	a.SetToken("elasticsearch", "Settings")
}

// Annotate ...
func (i *SettingsInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Persistent, "JSON string of persistent cluster settings.")
	a.Describe(&i.Transient, "JSON string of transient cluster settings.")
	a.Describe(&i.AdoptOnCreate, "If true, adopt existing settings into state instead of erroring.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Settings) Create(
	ctx context.Context, req infer.CreateRequest[SettingsInputs],
) (infer.CreateResponse[SettingsState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[SettingsState]{}, err
	}

	state, err := applySettings(ctx, esClient, req.Inputs)
	if err != nil {
		return infer.CreateResponse[SettingsState]{}, err
	}

	return infer.CreateResponse[SettingsState]{
		ID:     "cluster-settings",
		Output: state,
	}, nil
}

// Read ...
func (r *Settings) Read(
	ctx context.Context, req infer.ReadRequest[SettingsInputs, SettingsState],
) (infer.ReadResponse[SettingsInputs, SettingsState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[SettingsInputs, SettingsState]{}, err
	}

	var result map[string]json.RawMessage
	if err := esClient.GetJSON(ctx, "/_cluster/settings", &result); err != nil {
		return infer.ReadResponse[SettingsInputs, SettingsState]{}, err
	}

	state := req.State
	if p, ok := result["persistent"]; ok {
		state.AppliedPersistent = string(p)
	}
	if t, ok := result["transient"]; ok {
		state.AppliedTransient = string(t)
	}

	return infer.ReadResponse[SettingsInputs, SettingsState]{
		ID:     "cluster-settings",
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

// Update ...
func (r *Settings) Update(
	ctx context.Context, req infer.UpdateRequest[SettingsInputs, SettingsState],
) (infer.UpdateResponse[SettingsState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[SettingsState]{}, err
	}

	state, err := applySettings(ctx, esClient, req.Inputs)
	if err != nil {
		return infer.UpdateResponse[SettingsState]{}, err
	}

	return infer.UpdateResponse[SettingsState]{
		Output: state,
	}, nil
}

// Delete ...
func (r *Settings) Delete(
	ctx context.Context, req infer.DeleteRequest[SettingsState],
) (infer.DeleteResponse, error) {
	// Cluster settings can't be "deleted" — we reset them to empty
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	body := map[string]any{
		"persistent": map[string]any{"*": nil},
		"transient":  map[string]any{"*": nil},
	}

	if err := esClient.PutJSON(ctx, "/_cluster/settings", body, nil); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to reset cluster settings: %w", err)
	}

	return infer.DeleteResponse{}, nil
}

// Diff ...
func (r *Settings) Diff(
	ctx context.Context, req infer.DiffRequest[SettingsInputs, SettingsState],
) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}

	if !jsonStringsEqual(req.Inputs.Persistent, ptrOf(req.State.AppliedPersistent)) {
		diff["persistent"] = p.PropertyDiff{Kind: p.Update}
	}
	if !jsonStringsEqual(req.Inputs.Transient, ptrOf(req.State.AppliedTransient)) {
		diff["transient"] = p.PropertyDiff{Kind: p.Update}
	}

	return p.DiffResponse{
		HasChanges:   len(diff) > 0,
		DetailedDiff: diff,
	}, nil
}

func applySettings(ctx context.Context, esClient interface {
	PutJSON(ctx context.Context, path string, body any, dest any) error
}, inputs SettingsInputs,
) (SettingsState, error) {
	body := map[string]any{}

	if inputs.Persistent != nil {
		var persistent any
		if err := json.Unmarshal([]byte(*inputs.Persistent), &persistent); err != nil {
			return SettingsState{}, fmt.Errorf("failed to parse persistent settings JSON: %w", err)
		}
		body["persistent"] = persistent
	}

	if inputs.Transient != nil {
		var transient any
		if err := json.Unmarshal([]byte(*inputs.Transient), &transient); err != nil {
			return SettingsState{}, fmt.Errorf("failed to parse transient settings JSON: %w", err)
		}
		body["transient"] = transient
	}

	var result map[string]json.RawMessage
	if err := esClient.PutJSON(ctx, "/_cluster/settings", body, &result); err != nil {
		return SettingsState{}, fmt.Errorf("failed to update cluster settings: %w", err)
	}

	state := SettingsState{
		SettingsInputs: inputs,
	}
	if p, ok := result["persistent"]; ok {
		state.AppliedPersistent = string(p)
	}
	if t, ok := result["transient"]; ok {
		state.AppliedTransient = string(t)
	}

	return state, nil
}

func jsonStringsEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	var aVal, bVal any
	if err := json.Unmarshal([]byte(*a), &aVal); err != nil {
		return *a == *b
	}
	if err := json.Unmarshal([]byte(*b), &bVal); err != nil {
		return *a == *b
	}
	aJSON, _ := json.Marshal(aVal)
	bJSON, _ := json.Marshal(bVal)
	return string(aJSON) == string(bJSON)
}

func ptrOf(s string) *string {
	return &s
}
