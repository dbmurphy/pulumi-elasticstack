// Package logstash implements Elasticsearch Logstash pipeline management.
package logstash

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Pipeline manages a Logstash pipeline stored in Elasticsearch via PUT /_logstash/pipeline/<id>.
type Pipeline struct{}

// PipelineInputs defines the input properties for a Logstash pipeline.
type PipelineInputs struct {
	PipelineID                 string  `pulumi:"pipelineID"`
	Pipeline                   string  `pulumi:"pipeline"`
	Description                *string `pulumi:"description,optional"`
	PipelineBatchDelay         *int    `pulumi:"pipelineBatchDelay,optional"`
	PipelineBatchSize          *int    `pulumi:"pipelineBatchSize,optional"`
	PipelineEcsCompatibility   *string `pulumi:"pipelineEcsCompatibility,optional"`
	PipelineMetadata           *string `pulumi:"pipelineMetadata,optional"`
	PipelinePluginClassloaders *bool   `pulumi:"pipelinePluginClassloaders,optional"`
	PipelineWorkers            *int    `pulumi:"pipelineWorkers,optional"`
	QueueCheckpointWrites      *int    `pulumi:"queueCheckpointWrites,optional"`
	QueueDrain                 *bool   `pulumi:"queueDrain,optional"`
	QueueMaxBytes              *string `pulumi:"queueMaxBytes,optional"`
	QueueMaxEvents             *int    `pulumi:"queueMaxEvents,optional"`
	QueueType                  *string `pulumi:"queueType,optional"`
	Username                   *string `pulumi:"username,optional"`
	AdoptOnCreate              bool    `pulumi:"adoptOnCreate,optional"`
}

// PipelineState defines the output state for a Logstash pipeline.
type PipelineState struct {
	PipelineInputs
}

var (
	_ infer.CustomDelete[PipelineState]                 = (*Pipeline)(nil)
	_ infer.CustomRead[PipelineInputs, PipelineState]   = (*Pipeline)(nil)
	_ infer.CustomUpdate[PipelineInputs, PipelineState] = (*Pipeline)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *Pipeline) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Logstash pipeline stored in Elasticsearch.")
	a.SetToken("elasticsearch", "LogstashPipeline")
}

// Annotate sets input property descriptions and defaults.
func (i *PipelineInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.PipelineID, "The pipeline ID.")
	a.Describe(&i.Pipeline, "The Logstash pipeline configuration string.")
	a.Describe(&i.Description, "A description of the pipeline.")
	a.Describe(&i.PipelineBatchDelay, "Batch delay in milliseconds.")
	a.Describe(&i.PipelineBatchSize, "Maximum number of events per batch.")
	a.Describe(&i.PipelineEcsCompatibility, "ECS compatibility mode (e.g. 'disabled', 'v1', 'v8').")
	a.Describe(&i.PipelineMetadata, "Pipeline metadata as JSON.")
	a.Describe(&i.PipelinePluginClassloaders, "Whether to use per-plugin classloaders.")
	a.Describe(&i.PipelineWorkers, "Number of pipeline workers.")
	a.Describe(&i.QueueCheckpointWrites, "Number of events written before forcing a checkpoint.")
	a.Describe(&i.QueueDrain, "Whether to drain the queue before shutdown.")
	a.Describe(&i.QueueMaxBytes, "Maximum queue size in bytes (e.g. '1gb').")
	a.Describe(&i.QueueMaxEvents, "Maximum number of events in the queue.")
	a.Describe(&i.QueueType, "Queue type ('memory' or 'persisted').")
	a.Describe(&i.Username, "The username who last updated the pipeline.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing Logstash pipeline into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create provisions a new Logstash pipeline.
func (r *Pipeline) Create(
	ctx context.Context, req infer.CreateRequest[PipelineInputs],
) (infer.CreateResponse[PipelineState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[PipelineState]{}, err
	}

	pipelineID := req.Inputs.PipelineID

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_logstash/pipeline/"+pipelineID)
		if err != nil {
			return infer.CreateResponse[PipelineState]{}, err
		}
		if exists {
			body := buildLogstashPipelineBody(req.Inputs)
			if err := esClient.PutJSON(ctx, "/_logstash/pipeline/"+pipelineID, body, nil); err != nil {
				return infer.CreateResponse[PipelineState]{},
					fmt.Errorf("failed to update adopted Logstash pipeline %s: %w", pipelineID, err)
			}
			return infer.CreateResponse[PipelineState]{
				ID:     pipelineID,
				Output: PipelineState{PipelineInputs: req.Inputs},
			}, nil
		}
	}

	body := buildLogstashPipelineBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_logstash/pipeline/"+pipelineID, body, nil); err != nil {
		return infer.CreateResponse[PipelineState]{},
			fmt.Errorf("failed to create Logstash pipeline %s: %w", pipelineID, err)
	}

	return infer.CreateResponse[PipelineState]{
		ID:     pipelineID,
		Output: PipelineState{PipelineInputs: req.Inputs},
	}, nil
}

// Read fetches the current state of the Logstash pipeline.
func (r *Pipeline) Read(
	ctx context.Context, req infer.ReadRequest[PipelineInputs, PipelineState],
) (infer.ReadResponse[PipelineInputs, PipelineState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[PipelineInputs, PipelineState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_logstash/pipeline/"+req.ID)
	if err != nil {
		return infer.ReadResponse[PipelineInputs, PipelineState]{}, err
	}
	if !exists {
		return infer.ReadResponse[PipelineInputs, PipelineState]{ID: ""}, nil
	}

	return infer.ReadResponse[PipelineInputs, PipelineState](req), nil
}

// Update modifies an existing Logstash pipeline.
func (r *Pipeline) Update(
	ctx context.Context, req infer.UpdateRequest[PipelineInputs, PipelineState],
) (infer.UpdateResponse[PipelineState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[PipelineState]{}, err
	}

	body := buildLogstashPipelineBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_logstash/pipeline/"+req.Inputs.PipelineID, body, nil); err != nil {
		return infer.UpdateResponse[PipelineState]{},
			fmt.Errorf("failed to update Logstash pipeline %s: %w", req.Inputs.PipelineID, err)
	}

	return infer.UpdateResponse[PipelineState]{
		Output: PipelineState{PipelineInputs: req.Inputs},
	}, nil
}

// Delete removes the Logstash pipeline.
func (r *Pipeline) Delete(ctx context.Context, req infer.DeleteRequest[PipelineState]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_logstash/pipeline/"+req.State.PipelineID); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildLogstashPipelineBody(inputs PipelineInputs) map[string]any {
	body := map[string]any{
		"pipeline": inputs.Pipeline,
	}

	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if inputs.PipelineBatchDelay != nil {
		body["pipeline.batch.delay"] = *inputs.PipelineBatchDelay
	}
	if inputs.PipelineBatchSize != nil {
		body["pipeline.batch.size"] = *inputs.PipelineBatchSize
	}
	if inputs.PipelineEcsCompatibility != nil {
		body["pipeline.ecs_compatibility"] = *inputs.PipelineEcsCompatibility
	}
	if inputs.PipelineMetadata != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.PipelineMetadata), &meta); err == nil {
			body["pipeline_metadata"] = meta
		}
	}
	if inputs.PipelinePluginClassloaders != nil {
		body["pipeline.plugin_classloaders"] = *inputs.PipelinePluginClassloaders
	}
	if inputs.PipelineWorkers != nil {
		body["pipeline.workers"] = *inputs.PipelineWorkers
	}
	if inputs.QueueCheckpointWrites != nil {
		body["queue.checkpoint.writes"] = *inputs.QueueCheckpointWrites
	}
	if inputs.QueueDrain != nil {
		body["queue.drain"] = *inputs.QueueDrain
	}
	if inputs.QueueMaxBytes != nil {
		body["queue.max_bytes"] = *inputs.QueueMaxBytes
	}
	if inputs.QueueMaxEvents != nil {
		body["queue.max_events"] = *inputs.QueueMaxEvents
	}
	if inputs.QueueType != nil {
		body["queue.type"] = *inputs.QueueType
	}
	if inputs.Username != nil {
		body["username"] = *inputs.Username
	}

	return body
}
