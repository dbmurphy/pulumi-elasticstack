package script

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Script manages an Elasticsearch stored script via PUT /_scripts/<id>.
type Script struct{}

// Inputs ...
type Inputs struct {
	ScriptId      string  `pulumi:"scriptId"`
	Lang          string  `pulumi:"lang"`
	Source        string  `pulumi:"source"`
	Context       *string `pulumi:"context,optional"`
	Params        *string `pulumi:"params,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// State ...
type State struct {
	Inputs
}

var (
	_ infer.CustomDelete[State]         = (*Script)(nil)
	_ infer.CustomRead[Inputs, State]   = (*Script)(nil)
	_ infer.CustomUpdate[Inputs, State] = (*Script)(nil)
)

// Annotate ...
func (r *Script) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch stored script.")
	a.SetToken("elasticsearch", "Script")
}

// Annotate ...
func (i *Inputs) Annotate(a infer.Annotator) {
	a.Describe(&i.ScriptId, "The stored script ID.")
	a.Describe(&i.Lang, "The script language (e.g. 'painless', 'mustache').")
	a.Describe(&i.Source, "The script source code.")
	a.Describe(&i.Context, "The context in which the script is used.")
	a.Describe(&i.Params, "Default parameters for the script as JSON.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing stored script into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Script) Create(
	ctx context.Context, req infer.CreateRequest[Inputs],
) (infer.CreateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	scriptId := req.Inputs.ScriptId

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_scripts/"+scriptId)
		if err != nil {
			return infer.CreateResponse[State]{}, err
		}
		if exists {
			body, err := buildScriptBody(req.Inputs)
			if err != nil {
				return infer.CreateResponse[State]{}, err
			}
			if err := esClient.PutJSON(ctx, "/_scripts/"+scriptId, body, nil); err != nil {
				return infer.CreateResponse[State]{}, fmt.Errorf(
					"failed to update adopted script %s: %w",
					scriptId,
					err,
				)
			}
			return infer.CreateResponse[State]{
				ID:     scriptId,
				Output: State{Inputs: req.Inputs},
			}, nil
		}
	}

	body, err := buildScriptBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_scripts/"+scriptId, body, nil); err != nil {
		return infer.CreateResponse[State]{}, fmt.Errorf("failed to create stored script %s: %w", scriptId, err)
	}

	return infer.CreateResponse[State]{
		ID:     scriptId,
		Output: State{Inputs: req.Inputs},
	}, nil
}

// Read ...
func (r *Script) Read(
	ctx context.Context, req infer.ReadRequest[Inputs, State],
) (infer.ReadResponse[Inputs, State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_scripts/"+req.ID)
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}
	if !exists {
		return infer.ReadResponse[Inputs, State]{ID: ""}, nil
	}

	return infer.ReadResponse[Inputs, State](req), nil
}

// Update ...
func (r *Script) Update(
	ctx context.Context, req infer.UpdateRequest[Inputs, State],
) (infer.UpdateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	body, err := buildScriptBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_scripts/"+req.Inputs.ScriptId, body, nil); err != nil {
		return infer.UpdateResponse[State]{},
			fmt.Errorf("failed to update stored script %s: %w", req.Inputs.ScriptId, err)
	}

	return infer.UpdateResponse[State]{
		Output: State{Inputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *Script) Delete(ctx context.Context, req infer.DeleteRequest[State]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_scripts/"+req.State.ScriptId); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildScriptBody(inputs Inputs) (map[string]any, error) {
	script := map[string]any{
		"lang":   inputs.Lang,
		"source": inputs.Source,
	}

	if inputs.Params != nil {
		var params any
		if err := json.Unmarshal([]byte(*inputs.Params), &params); err != nil {
			return nil, fmt.Errorf("invalid params JSON: %w", err)
		}
		script["params"] = params
	}

	body := map[string]any{
		"script": script,
	}

	if inputs.Context != nil {
		body["context"] = *inputs.Context
	}

	return body, nil
}
