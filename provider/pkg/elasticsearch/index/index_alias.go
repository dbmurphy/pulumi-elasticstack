package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// AliasResource manages index aliases via POST /_aliases.
type AliasResource struct{}

// AliasInputs defines the input properties for an index alias resource.
type AliasInputs struct {
	Name          string   `pulumi:"name"`
	Indices       []string `pulumi:"indices"`
	Filter        *string  `pulumi:"filter,optional"`
	Routing       *string  `pulumi:"routing,optional"`
	IndexRouting  *string  `pulumi:"indexRouting,optional"`
	SearchRouting *string  `pulumi:"searchRouting,optional"`
	IsWriteIndex  *bool    `pulumi:"isWriteIndex,optional"`
	IsHidden      *bool    `pulumi:"isHidden,optional"`
	AdoptOnCreate bool     `pulumi:"adoptOnCreate,optional"`
}

// AliasState defines the output state for an index alias resource.
type AliasState struct {
	AliasInputs
}

var (
	_ infer.CustomDelete[AliasState]              = (*AliasResource)(nil)
	_ infer.CustomUpdate[AliasInputs, AliasState] = (*AliasResource)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *AliasResource) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch index alias.")
	a.SetToken("elasticsearch", "IndexAlias")
}

// Annotate sets input property descriptions and defaults.
func (i *AliasInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The alias name.")
	a.Describe(&i.Indices, "Target index names.")
	a.Describe(&i.Filter, "JSON filter query.")
	a.Describe(&i.Routing, "Routing value.")
	a.Describe(&i.IndexRouting, "Index routing value.")
	a.Describe(&i.SearchRouting, "Search routing value.")
	a.Describe(&i.IsWriteIndex, "Whether this is the write index.")
	a.Describe(&i.IsHidden, "Whether the alias is hidden.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing alias into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create provisions a new index alias.
func (r *AliasResource) Create(
	ctx context.Context, req infer.CreateRequest[AliasInputs],
) (infer.CreateResponse[AliasState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[AliasState]{}, err
	}

	actions := buildAliasActions("add", req.Inputs)
	body := map[string]any{"actions": actions}
	if err := esClient.PostJSON(ctx, "/_aliases", body, nil); err != nil {
		return infer.CreateResponse[AliasState]{}, fmt.Errorf("failed to create alias %s: %w", req.Inputs.Name, err)
	}

	return infer.CreateResponse[AliasState]{
		ID:     req.Inputs.Name,
		Output: AliasState{AliasInputs: req.Inputs},
	}, nil
}

// Update modifies an existing index alias.
func (r *AliasResource) Update(
	ctx context.Context, req infer.UpdateRequest[AliasInputs, AliasState],
) (infer.UpdateResponse[AliasState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[AliasState]{}, err
	}

	// Remove old aliases, then add new ones
	removeActions := buildAliasActions("remove", AliasInputs{
		Name:    req.State.Name,
		Indices: req.State.Indices,
	})
	addActions := buildAliasActions("add", req.Inputs)
	actions := append(removeActions, addActions...)

	body := map[string]any{"actions": actions}
	if err := esClient.PostJSON(ctx, "/_aliases", body, nil); err != nil {
		return infer.UpdateResponse[AliasState]{}, fmt.Errorf("failed to update alias %s: %w", req.Inputs.Name, err)
	}

	return infer.UpdateResponse[AliasState]{
		Output: AliasState{AliasInputs: req.Inputs},
	}, nil
}

// Delete removes the index alias.
func (r *AliasResource) Delete(ctx context.Context, req infer.DeleteRequest[AliasState]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	actions := buildAliasActions("remove", req.State.AliasInputs)
	body := map[string]any{"actions": actions}
	if err := esClient.PostJSON(ctx, "/_aliases", body, nil); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete alias %s: %w", req.State.Name, err)
	}

	return infer.DeleteResponse{}, nil
}

func buildAliasActions(action string, inputs AliasInputs) []any {
	actions := []any{}
	for _, idx := range inputs.Indices {
		aliasBody := map[string]any{
			"index": idx,
			"alias": inputs.Name,
		}
		if inputs.Filter != nil {
			var filter any
			if err := json.Unmarshal([]byte(*inputs.Filter), &filter); err == nil {
				aliasBody["filter"] = filter
			}
		}
		if inputs.Routing != nil {
			aliasBody["routing"] = *inputs.Routing
		}
		if inputs.IndexRouting != nil {
			aliasBody["index_routing"] = *inputs.IndexRouting
		}
		if inputs.SearchRouting != nil {
			aliasBody["search_routing"] = *inputs.SearchRouting
		}
		if inputs.IsWriteIndex != nil {
			aliasBody["is_write_index"] = *inputs.IsWriteIndex
		}
		if inputs.IsHidden != nil {
			aliasBody["is_hidden"] = *inputs.IsHidden
		}
		actions = append(actions, map[string]any{action: aliasBody})
	}
	return actions
}
