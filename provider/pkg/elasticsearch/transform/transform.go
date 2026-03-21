package transform

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Transform manages an Elasticsearch transform via PUT /_transform/<transform_id>.
type Transform struct{}

// Inputs ...
type Inputs struct {
	Name            string  `pulumi:"name"`
	Source          *string `pulumi:"source,optional"`
	Destination     *string `pulumi:"destination,optional"`
	Pivot           *string `pulumi:"pivot,optional"`
	Latest          *string `pulumi:"latest,optional"`
	Frequency       *string `pulumi:"frequency,optional"`
	Sync            *string `pulumi:"sync,optional"`
	RetentionPolicy *string `pulumi:"retentionPolicy,optional"`
	Description     *string `pulumi:"description,optional"`
	Metadata        *string `pulumi:"metadata,optional"`
	Enabled         *bool   `pulumi:"enabled,optional"`
	DeferValidation *bool   `pulumi:"deferValidation,optional"`
	AdoptOnCreate   bool    `pulumi:"adoptOnCreate,optional"`
}

// State ...
type State struct {
	Inputs
}

var (
	_ infer.CustomDelete[State]         = (*Transform)(nil)
	_ infer.CustomRead[Inputs, State]   = (*Transform)(nil)
	_ infer.CustomUpdate[Inputs, State] = (*Transform)(nil)
)

// Annotate ...
func (r *Transform) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch transform.")
	a.SetToken("elasticsearch", "Transform")
}

// Annotate ...
func (i *Inputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The transform ID.")
	a.Describe(&i.Source, "The source configuration as JSON.")
	a.Describe(&i.Destination, "The destination configuration as JSON.")
	a.Describe(&i.Pivot, "The pivot configuration as JSON.")
	a.Describe(&i.Latest, "The latest configuration as JSON.")
	a.Describe(&i.Frequency, "The interval between checks for changes in the source indices.")
	a.Describe(&i.Sync, "The sync configuration as JSON.")
	a.Describe(&i.RetentionPolicy, "The retention policy as JSON.")
	a.Describe(&i.Description, "A description of the transform.")
	a.Describe(&i.Metadata, "Transform metadata as JSON.")
	a.Describe(&i.Enabled, "Whether the transform should be started after creation.")
	a.SetDefault(&i.Enabled, false)
	a.Describe(&i.DeferValidation, "When true, deferring validation is not performed on create.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing transform into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Transform) Create(
	ctx context.Context,
	req infer.CreateRequest[Inputs],
) (infer.CreateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_transform/"+name)
		if err != nil {
			return infer.CreateResponse[State]{}, err
		}
		if exists {
			if err := updateTransform(ctx, esClient, name, req.Inputs); err != nil {
				return infer.CreateResponse[State]{}, fmt.Errorf(
					"failed to update adopted transform %s: %w",
					name,
					err,
				)
			}
			return infer.CreateResponse[State]{
				ID:     name,
				Output: State{Inputs: req.Inputs},
			}, nil
		}
	}

	path := "/_transform/" + name
	if req.Inputs.DeferValidation != nil && *req.Inputs.DeferValidation {
		path += "?defer_validation=true"
	}

	body, err := buildTransformBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}
	if err := esClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.CreateResponse[State]{}, fmt.Errorf("failed to create transform %s: %w", name, err)
	}

	if req.Inputs.Enabled != nil && *req.Inputs.Enabled {
		if err := esClient.PostJSON(ctx, "/_transform/"+name+"/_start", nil, nil); err != nil {
			return infer.CreateResponse[State]{}, fmt.Errorf("failed to start transform %s: %w", name, err)
		}
	}

	return infer.CreateResponse[State]{
		ID:     name,
		Output: State{Inputs: req.Inputs},
	}, nil
}

// Read ...
func (r *Transform) Read(
	ctx context.Context,
	req infer.ReadRequest[Inputs, State],
) (infer.ReadResponse[Inputs, State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_transform/"+req.ID)
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}
	if !exists {
		return infer.ReadResponse[Inputs, State]{ID: ""}, nil
	}

	return infer.ReadResponse[Inputs, State](req), nil
}

// Update ...
func (r *Transform) Update(
	ctx context.Context,
	req infer.UpdateRequest[Inputs, State],
) (infer.UpdateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	name := req.Inputs.Name
	if err := updateTransform(ctx, esClient, name, req.Inputs); err != nil {
		return infer.UpdateResponse[State]{}, fmt.Errorf("failed to update transform %s: %w", name, err)
	}

	return infer.UpdateResponse[State]{
		Output: State{Inputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *Transform) Delete(ctx context.Context, req infer.DeleteRequest[State]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_transform/"+req.State.Name+"?force=true"); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func updateTransform(
	ctx context.Context,
	esClient *clients.ElasticsearchClient,
	name string,
	inputs Inputs,
) error {
	body, err := buildTransformUpdateBody(inputs)
	if err != nil {
		return err
	}
	if err := esClient.PostJSON(ctx, "/_transform/"+name+"/_update", body, nil); err != nil {
		return err
	}

	if inputs.Enabled != nil {
		if *inputs.Enabled {
			if err := esClient.PostJSON(ctx, "/_transform/"+name+"/_start", nil, nil); err != nil {
				return fmt.Errorf("failed to start transform %s: %w", name, err)
			}
		} else {
			if err := esClient.PostJSON(ctx, "/_transform/"+name+"/_stop?force=true", nil, nil); err != nil {
				return fmt.Errorf("failed to stop transform %s: %w", name, err)
			}
		}
	}

	return nil
}

func buildTransformBody(inputs Inputs) (map[string]any, error) {
	body := map[string]any{}

	if inputs.Source != nil {
		var source any
		if err := json.Unmarshal([]byte(*inputs.Source), &source); err != nil {
			return nil, fmt.Errorf("invalid source JSON: %w", err)
		}
		body["source"] = source
	}
	if inputs.Destination != nil {
		var dest any
		if err := json.Unmarshal([]byte(*inputs.Destination), &dest); err != nil {
			return nil, fmt.Errorf("invalid dest JSON: %w", err)
		}
		body["dest"] = dest
	}
	if inputs.Pivot != nil {
		var pivot any
		if err := json.Unmarshal([]byte(*inputs.Pivot), &pivot); err != nil {
			return nil, fmt.Errorf("invalid pivot JSON: %w", err)
		}
		body["pivot"] = pivot
	}
	if inputs.Latest != nil {
		var latest any
		if err := json.Unmarshal([]byte(*inputs.Latest), &latest); err != nil {
			return nil, fmt.Errorf("invalid latest JSON: %w", err)
		}
		body["latest"] = latest
	}
	if inputs.Frequency != nil {
		body["frequency"] = *inputs.Frequency
	}
	if inputs.Sync != nil {
		var sync any
		if err := json.Unmarshal([]byte(*inputs.Sync), &sync); err != nil {
			return nil, fmt.Errorf("invalid sync JSON: %w", err)
		}
		body["sync"] = sync
	}
	if inputs.RetentionPolicy != nil {
		var rp any
		if err := json.Unmarshal([]byte(*inputs.RetentionPolicy), &rp); err != nil {
			return nil, fmt.Errorf("invalid rp JSON: %w", err)
		}
		body["retention_policy"] = rp
	}
	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if inputs.Metadata != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.Metadata), &meta); err != nil {
			return nil, fmt.Errorf("invalid meta JSON: %w", err)
		}
		body["_meta"] = meta
	}

	return body, nil
}

func buildTransformUpdateBody(inputs Inputs) (map[string]any, error) {
	body, err := buildTransformBody(inputs)
	if err != nil {
		return map[string]any{}, err
	}
	return body, nil
}
