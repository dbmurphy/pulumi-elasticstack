package ingest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Pipeline manages an ingest pipeline via PUT /_ingest/pipeline/<id>.
type Pipeline struct{}

// PipelineInputs ...
type PipelineInputs struct {
	Name          string  `pulumi:"name"`
	Description   *string `pulumi:"description,optional"`
	Processors    *string `pulumi:"processors,optional"`
	OnFailure     *string `pulumi:"onFailure,optional"`
	Metadata      *string `pulumi:"metadata,optional"`
	Version       *int    `pulumi:"version,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// PipelineState ...
type PipelineState struct {
	PipelineInputs
}

var (
	_ infer.CustomDelete[PipelineState]                 = (*Pipeline)(nil)
	_ infer.CustomRead[PipelineInputs, PipelineState]   = (*Pipeline)(nil)
	_ infer.CustomUpdate[PipelineInputs, PipelineState] = (*Pipeline)(nil)
)

// Annotate ...
func (r *Pipeline) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch ingest pipeline.")
	a.SetToken("elasticsearch", "Pipeline")
}

// Annotate ...
func (i *PipelineInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The pipeline ID.")
	a.Describe(&i.Description, "Pipeline description.")
	a.Describe(&i.Processors, "JSON array of processor definitions.")
	a.Describe(&i.OnFailure, "JSON array of on-failure processor definitions.")
	a.Describe(&i.Metadata, "Pipeline metadata as JSON.")
	a.Describe(&i.Version, "Pipeline version.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing pipeline into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Pipeline) Create(
	ctx context.Context,
	req infer.CreateRequest[PipelineInputs],
) (infer.CreateResponse[PipelineState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[PipelineState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_ingest/pipeline/"+name)
		if err != nil {
			return infer.CreateResponse[PipelineState]{}, err
		}
		if exists {
			body, err := buildPipelineBody(req.Inputs)
			if err != nil {
				return infer.CreateResponse[PipelineState]{}, err
			}
			if err := esClient.PutJSON(ctx, "/_ingest/pipeline/"+name, body, nil); err != nil {
				return infer.CreateResponse[PipelineState]{}, fmt.Errorf(
					"failed to update adopted pipeline %s: %w",
					name,
					err,
				)
			}
			return infer.CreateResponse[PipelineState]{
				ID:     name,
				Output: PipelineState{PipelineInputs: req.Inputs},
			}, nil
		}
	}

	body, err := buildPipelineBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[PipelineState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_ingest/pipeline/"+name, body, nil); err != nil {
		return infer.CreateResponse[PipelineState]{}, fmt.Errorf(
			"failed to create ingest pipeline %s: %w",
			name,
			err,
		)
	}

	return infer.CreateResponse[PipelineState]{
		ID:     name,
		Output: PipelineState{PipelineInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *Pipeline) Read(
	ctx context.Context,
	req infer.ReadRequest[PipelineInputs, PipelineState],
) (infer.ReadResponse[PipelineInputs, PipelineState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[PipelineInputs, PipelineState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_ingest/pipeline/"+req.ID)
	if err != nil {
		return infer.ReadResponse[PipelineInputs, PipelineState]{}, err
	}
	if !exists {
		return infer.ReadResponse[PipelineInputs, PipelineState]{ID: ""}, nil
	}

	return infer.ReadResponse[PipelineInputs, PipelineState](req), nil
}

// Update ...
func (r *Pipeline) Update(
	ctx context.Context,
	req infer.UpdateRequest[PipelineInputs, PipelineState],
) (infer.UpdateResponse[PipelineState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[PipelineState]{}, err
	}

	body, err := buildPipelineBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[PipelineState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_ingest/pipeline/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[PipelineState]{}, fmt.Errorf(
			"failed to update ingest pipeline %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	return infer.UpdateResponse[PipelineState]{
		Output: PipelineState{PipelineInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *Pipeline) Delete(
	ctx context.Context,
	req infer.DeleteRequest[PipelineState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_ingest/pipeline/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildPipelineBody(inputs PipelineInputs) (map[string]any, error) {
	body := map[string]any{}

	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if inputs.Processors != nil {
		var processors any
		if err := json.Unmarshal([]byte(*inputs.Processors), &processors); err != nil {
			return nil, fmt.Errorf("invalid processors JSON: %w", err)
		}
		body["processors"] = processors
	}
	if inputs.OnFailure != nil {
		var onFailure any
		if err := json.Unmarshal([]byte(*inputs.OnFailure), &onFailure); err != nil {
			return nil, fmt.Errorf("invalid onFailure JSON: %w", err)
		}
		body["on_failure"] = onFailure
	}
	if inputs.Metadata != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.Metadata), &meta); err != nil {
			return nil, fmt.Errorf("invalid meta JSON: %w", err)
		}
		body["_meta"] = meta
	}
	if inputs.Version != nil {
		body["version"] = *inputs.Version
	}

	return body, nil
}
