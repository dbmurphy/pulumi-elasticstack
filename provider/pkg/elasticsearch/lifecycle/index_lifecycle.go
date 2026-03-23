package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// IndexLifecycle manages an ILM policy via PUT /_ilm/policy/<name>.
type IndexLifecycle struct{}

// IndexLifecycleInputs ...
type IndexLifecycleInputs struct {
	Name          string  `pulumi:"name"`
	Metadata      *string `pulumi:"metadata,optional"`
	Hot           *string `pulumi:"hot,optional"`
	Warm          *string `pulumi:"warm,optional"`
	Cold          *string `pulumi:"cold,optional"`
	Frozen        *string `pulumi:"frozen,optional"`
	Delete        *string `pulumi:"delete,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// IndexLifecycleState ...
type IndexLifecycleState struct {
	IndexLifecycleInputs
}

var (
	_ infer.CustomDelete[IndexLifecycleState]                       = (*IndexLifecycle)(nil)
	_ infer.CustomRead[IndexLifecycleInputs, IndexLifecycleState]   = (*IndexLifecycle)(nil)
	_ infer.CustomUpdate[IndexLifecycleInputs, IndexLifecycleState] = (*IndexLifecycle)(nil)
)

// Annotate ...
func (r *IndexLifecycle) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch Index Lifecycle Management (ILM) policy.")
	a.SetToken("elasticsearch", "IndexLifecycle")
}

// Annotate ...
func (i *IndexLifecycleInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the ILM policy.")
	a.Describe(&i.Metadata, "Policy metadata as JSON.")
	a.Describe(&i.Hot, "Hot phase configuration as JSON.")
	a.Describe(&i.Warm, "Warm phase configuration as JSON.")
	a.Describe(&i.Cold, "Cold phase configuration as JSON.")
	a.Describe(&i.Frozen, "Frozen phase configuration as JSON.")
	a.Describe(&i.Delete, "Delete phase configuration as JSON.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing ILM policy into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *IndexLifecycle) Create(
	ctx context.Context,
	req infer.CreateRequest[IndexLifecycleInputs],
) (infer.CreateResponse[IndexLifecycleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[IndexLifecycleState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_ilm/policy/"+name)
		if err != nil {
			return infer.CreateResponse[IndexLifecycleState]{}, err
		}
		if exists {
			body, err := buildILMBody(req.Inputs)
			if err != nil {
				return infer.CreateResponse[IndexLifecycleState]{}, err
			}
			if err := esClient.PutJSON(ctx, "/_ilm/policy/"+name, body, nil); err != nil {
				return infer.CreateResponse[IndexLifecycleState]{},
					fmt.Errorf("failed to update adopted ILM policy %s: %w", name, err)
			}
			return infer.CreateResponse[IndexLifecycleState]{
				ID:     name,
				Output: IndexLifecycleState{IndexLifecycleInputs: req.Inputs},
			}, nil
		}
	}

	body, err := buildILMBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[IndexLifecycleState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_ilm/policy/"+name, body, nil); err != nil {
		return infer.CreateResponse[IndexLifecycleState]{}, fmt.Errorf("failed to create ILM policy %s: %w", name, err)
	}

	return infer.CreateResponse[IndexLifecycleState]{
		ID:     name,
		Output: IndexLifecycleState{IndexLifecycleInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *IndexLifecycle) Read(
	ctx context.Context, req infer.ReadRequest[IndexLifecycleInputs, IndexLifecycleState],
) (infer.ReadResponse[IndexLifecycleInputs, IndexLifecycleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[IndexLifecycleInputs, IndexLifecycleState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_ilm/policy/"+req.ID)
	if err != nil {
		return infer.ReadResponse[IndexLifecycleInputs, IndexLifecycleState]{}, err
	}
	if !exists {
		return infer.ReadResponse[IndexLifecycleInputs, IndexLifecycleState]{ID: ""}, nil
	}

	return infer.ReadResponse[IndexLifecycleInputs, IndexLifecycleState](req), nil
}

// Update ...
func (r *IndexLifecycle) Update(
	ctx context.Context,
	req infer.UpdateRequest[IndexLifecycleInputs, IndexLifecycleState],
) (infer.UpdateResponse[IndexLifecycleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[IndexLifecycleState]{}, err
	}

	body, err := buildILMBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[IndexLifecycleState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_ilm/policy/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[IndexLifecycleState]{},
			fmt.Errorf("failed to update ILM policy %s: %w", req.Inputs.Name, err)
	}

	return infer.UpdateResponse[IndexLifecycleState]{
		Output: IndexLifecycleState{IndexLifecycleInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *IndexLifecycle) Delete(
	ctx context.Context,
	req infer.DeleteRequest[IndexLifecycleState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_ilm/policy/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildILMBody(inputs IndexLifecycleInputs) (map[string]any, error) {
	policy := map[string]any{}
	phases := map[string]any{}

	if inputs.Hot != nil {
		var hot any
		if err := json.Unmarshal([]byte(*inputs.Hot), &hot); err != nil {
			return nil, fmt.Errorf("invalid hot phase JSON: %w", err)
		}
		phases["hot"] = hot
	}
	if inputs.Warm != nil {
		var warm any
		if err := json.Unmarshal([]byte(*inputs.Warm), &warm); err != nil {
			return nil, fmt.Errorf("invalid warm phase JSON: %w", err)
		}
		phases["warm"] = warm
	}
	if inputs.Cold != nil {
		var cold any
		if err := json.Unmarshal([]byte(*inputs.Cold), &cold); err != nil {
			return nil, fmt.Errorf("invalid cold phase JSON: %w", err)
		}
		phases["cold"] = cold
	}
	if inputs.Frozen != nil {
		var frozen any
		if err := json.Unmarshal([]byte(*inputs.Frozen), &frozen); err != nil {
			return nil, fmt.Errorf("invalid frozen phase JSON: %w", err)
		}
		phases["frozen"] = frozen
	}
	if inputs.Delete != nil {
		var del any
		if err := json.Unmarshal([]byte(*inputs.Delete), &del); err != nil {
			return nil, fmt.Errorf("invalid delete phase JSON: %w", err)
		}
		phases["delete"] = del
	}

	policy["phases"] = phases

	if inputs.Metadata != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.Metadata), &meta); err != nil {
			return nil, fmt.Errorf("invalid metadata JSON: %w", err)
		}
		policy["_meta"] = meta
	}

	return map[string]any{"policy": policy}, nil
}
