package template

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// IndexTemplate manages an Elasticsearch index template.
type IndexTemplate struct{}

// IndexTemplateInputs defines the input properties for an index template.
type IndexTemplateInputs struct {
	Name              string   `pulumi:"name"`
	IndexPatterns     []string `pulumi:"indexPatterns"`
	ComposedOf        []string `pulumi:"composedOf,optional"`
	DataStream        *string  `pulumi:"dataStream,optional"`
	Template          *string  `pulumi:"template,optional"`
	Priority          *int     `pulumi:"priority,optional"`
	Version           *int     `pulumi:"version,optional"`
	Meta              *string  `pulumi:"meta,optional"`
	AdoptOnCreate     bool     `pulumi:"adoptOnCreate,optional"`
	MergeWithExisting bool     `pulumi:"mergeWithExisting,optional"`
}

// IndexTemplateState defines the output state for an index template.
type IndexTemplateState struct {
	IndexTemplateInputs
}

var (
	_ infer.CustomDelete[IndexTemplateState]                      = (*IndexTemplate)(nil)
	_ infer.CustomRead[IndexTemplateInputs, IndexTemplateState]   = (*IndexTemplate)(nil)
	_ infer.CustomUpdate[IndexTemplateInputs, IndexTemplateState] = (*IndexTemplate)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *IndexTemplate) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch index template.")
	a.SetToken("elasticsearch", "IndexTemplate")
}

// Annotate sets input property descriptions and defaults.
func (i *IndexTemplateInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the index template.")
	a.Describe(&i.IndexPatterns, "Index patterns that this template applies to.")
	a.Describe(&i.ComposedOf, "Component template names to compose from.")
	a.Describe(&i.DataStream, "Data stream configuration as JSON.")
	a.Describe(&i.Template, "Template body as JSON (settings, mappings, aliases).")
	a.Describe(&i.Priority, "Template priority.")
	a.Describe(&i.Version, "Template version.")
	a.Describe(&i.Meta, "Template metadata as JSON.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing template into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
	a.Describe(&i.MergeWithExisting,
		"When adopting, merge settings rather than overwrite.")
	a.SetDefault(&i.MergeWithExisting, false)
}

// Create provisions a new index template.
func (r *IndexTemplate) Create(
	ctx context.Context, req infer.CreateRequest[IndexTemplateInputs],
) (infer.CreateResponse[IndexTemplateState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[IndexTemplateState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_index_template/"+name)
		if err != nil {
			return infer.CreateResponse[IndexTemplateState]{}, err
		}
		if exists {
			if !req.Inputs.MergeWithExisting {
				body := buildIndexTemplateBody(req.Inputs)
				if err := esClient.PutJSON(ctx, "/_index_template/"+name, body, nil); err != nil {
					return infer.CreateResponse[IndexTemplateState]{},
						fmt.Errorf("failed to update adopted template %s: %w", name, err)
				}
			}
			return infer.CreateResponse[IndexTemplateState]{
				ID:     name,
				Output: IndexTemplateState{IndexTemplateInputs: req.Inputs},
			}, nil
		}
	}

	body := buildIndexTemplateBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_index_template/"+name, body, nil); err != nil {
		return infer.CreateResponse[IndexTemplateState]{},
			fmt.Errorf("failed to create index template %s: %w", name, err)
	}

	return infer.CreateResponse[IndexTemplateState]{
		ID:     name,
		Output: IndexTemplateState{IndexTemplateInputs: req.Inputs},
	}, nil
}

// Read fetches the current state of the index template.
func (r *IndexTemplate) Read(
	ctx context.Context,
	req infer.ReadRequest[IndexTemplateInputs, IndexTemplateState],
) (infer.ReadResponse[IndexTemplateInputs, IndexTemplateState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[IndexTemplateInputs, IndexTemplateState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_index_template/"+req.ID)
	if err != nil {
		return infer.ReadResponse[IndexTemplateInputs, IndexTemplateState]{}, err
	}
	if !exists {
		return infer.ReadResponse[IndexTemplateInputs, IndexTemplateState]{ID: ""}, nil
	}

	return infer.ReadResponse[IndexTemplateInputs, IndexTemplateState](req), nil
}

// Update modifies an existing index template.
func (r *IndexTemplate) Update(
	ctx context.Context,
	req infer.UpdateRequest[IndexTemplateInputs, IndexTemplateState],
) (infer.UpdateResponse[IndexTemplateState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[IndexTemplateState]{}, err
	}

	body := buildIndexTemplateBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_index_template/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[IndexTemplateState]{},
			fmt.Errorf("failed to update index template %s: %w", req.Inputs.Name, err)
	}

	return infer.UpdateResponse[IndexTemplateState]{
		Output: IndexTemplateState{IndexTemplateInputs: req.Inputs},
	}, nil
}

// Delete removes the index template.
func (r *IndexTemplate) Delete(
	ctx context.Context, req infer.DeleteRequest[IndexTemplateState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_index_template/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildIndexTemplateBody(inputs IndexTemplateInputs) map[string]any {
	body := map[string]any{
		"index_patterns": inputs.IndexPatterns,
	}

	if len(inputs.ComposedOf) > 0 {
		body["composed_of"] = inputs.ComposedOf
	}
	if inputs.DataStream != nil {
		var ds any
		if err := json.Unmarshal([]byte(*inputs.DataStream), &ds); err == nil {
			body["data_stream"] = ds
		}
	}
	if inputs.Template != nil {
		var tmpl any
		if err := json.Unmarshal([]byte(*inputs.Template), &tmpl); err == nil {
			body["template"] = tmpl
		}
	}
	if inputs.Priority != nil {
		body["priority"] = *inputs.Priority
	}
	if inputs.Version != nil {
		body["version"] = *inputs.Version
	}
	if inputs.Meta != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.Meta), &meta); err == nil {
			body["_meta"] = meta
		}
	}

	return body
}
