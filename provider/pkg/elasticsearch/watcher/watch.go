package watcher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

const queryInactive = "?active=false"

// Watch manages an Elasticsearch watcher watch via PUT /_watcher/watch/<watch_id>.
type Watch struct{}

// WatchInputs ...
type WatchInputs struct {
	WatchId        string  `pulumi:"watchId"`
	Active         *bool   `pulumi:"active,optional"`
	Trigger        string  `pulumi:"trigger"`
	Input          string  `pulumi:"input"`
	Condition      *string `pulumi:"condition,optional"`
	Actions        string  `pulumi:"actions"`
	Transform      *string `pulumi:"transform,optional"`
	ThrottlePeriod *string `pulumi:"throttlePeriod,optional"`
	Metadata       *string `pulumi:"metadata,optional"`
	AdoptOnCreate  bool    `pulumi:"adoptOnCreate,optional"`
}

// WatchState ...
type WatchState struct {
	WatchInputs
}

var (
	_ infer.CustomDelete[WatchState]              = (*Watch)(nil)
	_ infer.CustomRead[WatchInputs, WatchState]   = (*Watch)(nil)
	_ infer.CustomUpdate[WatchInputs, WatchState] = (*Watch)(nil)
)

// Annotate ...
func (r *Watch) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch Watcher watch.")
	a.SetToken("elasticsearch", "Watch")
}

// Annotate ...
func (i *WatchInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.WatchId, "The watch ID.")
	a.Describe(&i.Active, "Whether the watch is active. Defaults to true.")
	a.SetDefault(&i.Active, true)
	a.Describe(&i.Trigger, "The trigger configuration as JSON.")
	a.Describe(&i.Input, "The input configuration as JSON.")
	a.Describe(&i.Condition, "The condition configuration as JSON.")
	a.Describe(&i.Actions, "The actions configuration as JSON.")
	a.Describe(&i.Transform, "The transform configuration as JSON.")
	a.Describe(&i.ThrottlePeriod, "The minimum time between actions being run (e.g. '5m').")
	a.Describe(&i.Metadata, "Watch metadata as JSON.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing watch into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Watch) Create(
	ctx context.Context,
	req infer.CreateRequest[WatchInputs],
) (infer.CreateResponse[WatchState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[WatchState]{}, err
	}

	watchId := req.Inputs.WatchId

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_watcher/watch/"+watchId)
		if err != nil {
			return infer.CreateResponse[WatchState]{}, err
		}
		if exists {
			body, err := buildWatchBody(req.Inputs)
			if err != nil {
				return infer.CreateResponse[WatchState]{}, err
			}
			path := "/_watcher/watch/" + watchId
			if req.Inputs.Active != nil && !*req.Inputs.Active {
				path += queryInactive
			}
			if err := esClient.PutJSON(ctx, path, body, nil); err != nil {
				return infer.CreateResponse[WatchState]{}, fmt.Errorf(
					"failed to update adopted watch %s: %w",
					watchId,
					err,
				)
			}
			return infer.CreateResponse[WatchState]{
				ID:     watchId,
				Output: WatchState{WatchInputs: req.Inputs},
			}, nil
		}
	}

	body, err := buildWatchBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[WatchState]{}, err
	}
	path := "/_watcher/watch/" + watchId
	if req.Inputs.Active != nil && !*req.Inputs.Active {
		path += queryInactive
	}
	if err := esClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.CreateResponse[WatchState]{}, fmt.Errorf("failed to create watch %s: %w", watchId, err)
	}

	return infer.CreateResponse[WatchState]{
		ID:     watchId,
		Output: WatchState{WatchInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *Watch) Read(
	ctx context.Context,
	req infer.ReadRequest[WatchInputs, WatchState],
) (infer.ReadResponse[WatchInputs, WatchState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[WatchInputs, WatchState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_watcher/watch/"+req.ID)
	if err != nil {
		return infer.ReadResponse[WatchInputs, WatchState]{}, err
	}
	if !exists {
		return infer.ReadResponse[WatchInputs, WatchState]{ID: ""}, nil
	}

	return infer.ReadResponse[WatchInputs, WatchState](req), nil
}

// Update ...
func (r *Watch) Update(
	ctx context.Context,
	req infer.UpdateRequest[WatchInputs, WatchState],
) (infer.UpdateResponse[WatchState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[WatchState]{}, err
	}

	watchId := req.Inputs.WatchId
	body, err := buildWatchBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[WatchState]{}, err
	}
	path := "/_watcher/watch/" + watchId
	if req.Inputs.Active != nil && !*req.Inputs.Active {
		path += queryInactive
	}
	if err := esClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[WatchState]{}, fmt.Errorf("failed to update watch %s: %w", watchId, err)
	}

	return infer.UpdateResponse[WatchState]{
		Output: WatchState{WatchInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *Watch) Delete(ctx context.Context, req infer.DeleteRequest[WatchState]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_watcher/watch/"+req.State.WatchId); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildWatchBody(inputs WatchInputs) (map[string]any, error) {
	body := map[string]any{}

	var trigger any
	if err := json.Unmarshal([]byte(inputs.Trigger), &trigger); err != nil {
		return nil, fmt.Errorf("invalid trigger JSON: %w", err)
	}
	body["trigger"] = trigger

	var input any
	if err := json.Unmarshal([]byte(inputs.Input), &input); err != nil {
		return nil, fmt.Errorf("invalid input JSON: %w", err)
	}
	body["input"] = input

	var actions any
	if err := json.Unmarshal([]byte(inputs.Actions), &actions); err != nil {
		return nil, fmt.Errorf("invalid actions JSON: %w", err)
	}
	body["actions"] = actions

	if inputs.Condition != nil {
		var condition any
		if err := json.Unmarshal([]byte(*inputs.Condition), &condition); err != nil {
			return nil, fmt.Errorf("invalid condition JSON: %w", err)
		}
		body["condition"] = condition
	}
	if inputs.Transform != nil {
		var transform any
		if err := json.Unmarshal([]byte(*inputs.Transform), &transform); err != nil {
			return nil, fmt.Errorf("invalid transform JSON: %w", err)
		}
		body["transform"] = transform
	}
	if inputs.ThrottlePeriod != nil {
		body["throttle_period"] = *inputs.ThrottlePeriod
	}
	if inputs.Metadata != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.Metadata), &meta); err != nil {
			return nil, fmt.Errorf("invalid metadata JSON: %w", err)
		}
		body["metadata"] = meta
	}

	return body, nil
}
