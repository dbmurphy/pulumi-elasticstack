// Package template implements Elasticsearch template management.
package template

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// ComponentTemplate manages an Elasticsearch component template.
type ComponentTemplate struct{}

// ComponentTemplateInputs defines the input properties for a component template.
type ComponentTemplateInputs struct {
	Name          string  `pulumi:"name"`
	Template      string  `pulumi:"template"`
	Version       *int    `pulumi:"version,optional"`
	Meta          *string `pulumi:"meta,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// ComponentTemplateState defines the output state for a component template.
type ComponentTemplateState struct {
	ComponentTemplateInputs
}

var (
	_ infer.CustomDelete[ComponentTemplateState]                          = (*ComponentTemplate)(nil)
	_ infer.CustomRead[ComponentTemplateInputs, ComponentTemplateState]   = (*ComponentTemplate)(nil)
	_ infer.CustomUpdate[ComponentTemplateInputs, ComponentTemplateState] = (*ComponentTemplate)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *ComponentTemplate) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch component template.")
	a.SetToken("elasticsearch", "ComponentTemplate")
}

// Annotate sets input property descriptions and defaults.
func (i *ComponentTemplateInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the component template.")
	a.Describe(&i.Template, "Template body as JSON (settings, mappings, aliases).")
	a.Describe(&i.Version, "Template version.")
	a.Describe(&i.Meta, "Template metadata as JSON.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing component template into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create provisions a new component template.
func (r *ComponentTemplate) Create(
	ctx context.Context, req infer.CreateRequest[ComponentTemplateInputs],
) (infer.CreateResponse[ComponentTemplateState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[ComponentTemplateState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_component_template/"+name)
		if err != nil {
			return infer.CreateResponse[ComponentTemplateState]{}, err
		}
		if exists {
			body := buildComponentTemplateBody(req.Inputs)
			if err := esClient.PutJSON(ctx, "/_component_template/"+name, body, nil); err != nil {
				return infer.CreateResponse[ComponentTemplateState]{},
					fmt.Errorf("failed to update adopted component template %s: %w", name, err)
			}
			return infer.CreateResponse[ComponentTemplateState]{
				ID:     name,
				Output: ComponentTemplateState{ComponentTemplateInputs: req.Inputs},
			}, nil
		}
	}

	body := buildComponentTemplateBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_component_template/"+name, body, nil); err != nil {
		return infer.CreateResponse[ComponentTemplateState]{},
			fmt.Errorf("failed to create component template %s: %w", name, err)
	}

	return infer.CreateResponse[ComponentTemplateState]{
		ID:     name,
		Output: ComponentTemplateState{ComponentTemplateInputs: req.Inputs},
	}, nil
}

// Read fetches the current state of the component template.
func (r *ComponentTemplate) Read(
	ctx context.Context,
	req infer.ReadRequest[ComponentTemplateInputs, ComponentTemplateState],
) (infer.ReadResponse[ComponentTemplateInputs, ComponentTemplateState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[ComponentTemplateInputs, ComponentTemplateState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_component_template/"+req.ID)
	if err != nil {
		return infer.ReadResponse[ComponentTemplateInputs, ComponentTemplateState]{}, err
	}
	if !exists {
		return infer.ReadResponse[ComponentTemplateInputs, ComponentTemplateState]{ID: ""}, nil
	}

	return infer.ReadResponse[ComponentTemplateInputs, ComponentTemplateState](req), nil
}

// Update modifies an existing component template.
func (r *ComponentTemplate) Update(
	ctx context.Context,
	req infer.UpdateRequest[ComponentTemplateInputs, ComponentTemplateState],
) (infer.UpdateResponse[ComponentTemplateState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[ComponentTemplateState]{}, err
	}

	body := buildComponentTemplateBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_component_template/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[ComponentTemplateState]{},
			fmt.Errorf("failed to update component template %s: %w", req.Inputs.Name, err)
	}

	return infer.UpdateResponse[ComponentTemplateState]{
		Output: ComponentTemplateState{ComponentTemplateInputs: req.Inputs},
	}, nil
}

// Delete removes the component template.
func (r *ComponentTemplate) Delete(
	ctx context.Context, req infer.DeleteRequest[ComponentTemplateState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_component_template/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildComponentTemplateBody(inputs ComponentTemplateInputs) map[string]any {
	body := map[string]any{}

	var tmpl any
	if err := json.Unmarshal([]byte(inputs.Template), &tmpl); err == nil {
		body["template"] = tmpl
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
