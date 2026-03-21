// Package snapshot implements Elasticsearch snapshot lifecycle and repository management.
package snapshot

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Lifecycle manages an SLM policy via PUT /_slm/policy/<name>.
type Lifecycle struct{}

// LifecycleInputs defines the input properties for a snapshot lifecycle policy.
type LifecycleInputs struct {
	Name               string   `pulumi:"name"`
	Schedule           string   `pulumi:"schedule"`
	SnapshotName       string   `pulumi:"snapshotName"`
	Repository         string   `pulumi:"repository"`
	Indices            []string `pulumi:"indices,optional"`
	ExpireAfter        *string  `pulumi:"expireAfter,optional"`
	MaxCount           *int     `pulumi:"maxCount,optional"`
	MinCount           *int     `pulumi:"minCount,optional"`
	IgnoreUnavailable  *bool    `pulumi:"ignoreUnavailable,optional"`
	IncludeGlobalState *bool    `pulumi:"includeGlobalState,optional"`
	Partial            *bool    `pulumi:"partial,optional"`
	FeatureStates      []string `pulumi:"featureStates,optional"`
	AdoptOnCreate      bool     `pulumi:"adoptOnCreate,optional"`
}

// LifecycleState defines the output state for a snapshot lifecycle policy.
type LifecycleState struct {
	LifecycleInputs
}

var (
	_ infer.CustomDelete[LifecycleState]                  = (*Lifecycle)(nil)
	_ infer.CustomUpdate[LifecycleInputs, LifecycleState] = (*Lifecycle)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *Lifecycle) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch Snapshot Lifecycle Management (SLM) policy.")
	a.SetToken("elasticsearch", "SnapshotLifecycle")
}

// Annotate sets input property descriptions and defaults.
func (i *LifecycleInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The SLM policy name.")
	a.Describe(&i.Schedule, "Cron schedule for snapshot creation.")
	a.Describe(&i.SnapshotName, "Snapshot name template.")
	a.Describe(&i.Repository, "Snapshot repository name.")
	a.Describe(&i.Indices, "Indices to include in snapshots.")
	a.Describe(&i.ExpireAfter, "Time after which snapshots expire.")
	a.Describe(&i.MaxCount, "Maximum number of snapshots to retain.")
	a.Describe(&i.MinCount, "Minimum number of snapshots to retain.")
	a.Describe(&i.IgnoreUnavailable, "Ignore unavailable indices.")
	a.Describe(&i.IncludeGlobalState, "Include global state in snapshots.")
	a.Describe(&i.Partial, "Allow partial snapshots.")
	a.Describe(&i.FeatureStates, "Feature states to include.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing SLM policy into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create provisions a new snapshot lifecycle policy.
func (r *Lifecycle) Create(
	ctx context.Context,
	req infer.CreateRequest[LifecycleInputs],
) (infer.CreateResponse[LifecycleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[LifecycleState]{}, err
	}

	name := req.Inputs.Name
	body := buildSLMBody(req.Inputs)

	if err := esClient.PutJSON(ctx, "/_slm/policy/"+name, body, nil); err != nil {
		return infer.CreateResponse[LifecycleState]{}, fmt.Errorf("failed to create SLM policy %s: %w", name, err)
	}

	return infer.CreateResponse[LifecycleState]{
		ID:     name,
		Output: LifecycleState{LifecycleInputs: req.Inputs},
	}, nil
}

// Update modifies an existing snapshot lifecycle policy.
func (r *Lifecycle) Update(
	ctx context.Context,
	req infer.UpdateRequest[LifecycleInputs, LifecycleState],
) (infer.UpdateResponse[LifecycleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[LifecycleState]{}, err
	}

	body := buildSLMBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_slm/policy/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[LifecycleState]{}, fmt.Errorf(
			"failed to update SLM policy %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	return infer.UpdateResponse[LifecycleState]{
		Output: LifecycleState{LifecycleInputs: req.Inputs},
	}, nil
}

// Delete removes the snapshot lifecycle policy.
func (r *Lifecycle) Delete(ctx context.Context, req infer.DeleteRequest[LifecycleState]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_slm/policy/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildSLMBody(inputs LifecycleInputs) map[string]any {
	body := map[string]any{
		"schedule":   inputs.Schedule,
		"name":       inputs.SnapshotName,
		"repository": inputs.Repository,
	}

	config := map[string]any{}
	if len(inputs.Indices) > 0 {
		config["indices"] = inputs.Indices
	}
	if inputs.IgnoreUnavailable != nil {
		config["ignore_unavailable"] = *inputs.IgnoreUnavailable
	}
	if inputs.IncludeGlobalState != nil {
		config["include_global_state"] = *inputs.IncludeGlobalState
	}
	if inputs.Partial != nil {
		config["partial"] = *inputs.Partial
	}
	if len(inputs.FeatureStates) > 0 {
		config["feature_states"] = inputs.FeatureStates
	}
	if len(config) > 0 {
		body["config"] = config
	}

	retention := map[string]any{}
	if inputs.ExpireAfter != nil {
		retention["expire_after"] = *inputs.ExpireAfter
	}
	if inputs.MaxCount != nil {
		retention["max_count"] = *inputs.MaxCount
	}
	if inputs.MinCount != nil {
		retention["min_count"] = *inputs.MinCount
	}
	if len(retention) > 0 {
		body["retention"] = retention
	}

	return body
}
