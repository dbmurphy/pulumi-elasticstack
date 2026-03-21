package index

import (
	"context"
	"encoding/json"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Index manages an Elasticsearch index via PUT/GET/DELETE /<index>.
type Index struct{}

// Alias defines an alias configuration for an Elasticsearch index.
type Alias struct {
	Name          string  `pulumi:"name"`
	Filter        *string `pulumi:"filter,optional"`
	Routing       *string `pulumi:"routing,optional"`
	IndexRouting  *string `pulumi:"indexRouting,optional"`
	SearchRouting *string `pulumi:"searchRouting,optional"`
	IsWriteIndex  *bool   `pulumi:"isWriteIndex,optional"`
	IsHidden      *bool   `pulumi:"isHidden,optional"`
}

// Inputs defines the input properties for an Elasticsearch index.
type Inputs struct {
	Name                          string   `pulumi:"name"`
	Aliases                       []Alias  `pulumi:"aliases,optional"`
	Mappings                      *string  `pulumi:"mappings,optional"`
	Settings                      *string  `pulumi:"settings,optional"`
	SettingsRaw                   *string  `pulumi:"settingsRaw,optional"`
	NumberOfShards                *int     `pulumi:"numberOfShards,optional"`
	NumberOfReplicas              *int     `pulumi:"numberOfReplicas,optional"`
	NumberOfRoutingShards         *int     `pulumi:"numberOfRoutingShards,optional"`
	Codec                         *string  `pulumi:"codec,optional"`
	RoutingPartitionSize          *int     `pulumi:"routingPartitionSize,optional"`
	SortField                     []string `pulumi:"sortField,optional"`
	SortOrder                     []string `pulumi:"sortOrder,optional"`
	LoadFixedBitsetFiltersEagerly *bool    `pulumi:"loadFixedBitsetFiltersEagerly,optional"`
	Hidden                        *bool    `pulumi:"hidden,optional"`
	DeletionProtection            *bool    `pulumi:"deletionProtection,optional"`
	IncludeTypeName               *bool    `pulumi:"includeTypeName,optional"`
	WaitForActiveShards           *string  `pulumi:"waitForActiveShards,optional"`
	MasterTimeout                 *string  `pulumi:"masterTimeout,optional"`
	Timeout                       *string  `pulumi:"timeout,optional"`
	AdoptOnCreate                 bool     `pulumi:"adoptOnCreate,optional"`
	IgnoreShardCountChanges       bool     `pulumi:"ignoreShardCountChanges,optional"`
	IgnoreSettingsOnRead          []string `pulumi:"ignoreSettingsOnRead,optional"`
}

// State defines the output state for an Elasticsearch index.
type State struct {
	Inputs

	// Computed outputs
	UUID string `pulumi:"uuid"`
}

var (
	_ infer.CustomDelete[State]         = (*Index)(nil)
	_ infer.CustomRead[Inputs, State]   = (*Index)(nil)
	_ infer.CustomUpdate[Inputs, State] = (*Index)(nil)
	_ infer.CustomDiff[Inputs, State]   = (*Index)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *Index) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch index.")
	a.SetToken("elasticsearch", "Index")
}

// Annotate sets input property descriptions and defaults.
func (i *Inputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the index.")
	a.Describe(&i.Aliases, "Index aliases.")
	a.Describe(&i.Mappings, "JSON mapping definition.")
	a.Describe(&i.Settings, "JSON index settings.")
	a.Describe(&i.SettingsRaw, "Raw settings JSON.")
	a.Describe(&i.NumberOfShards, "Number of primary shards.")
	a.Describe(&i.NumberOfReplicas, "Number of replicas.")
	a.Describe(&i.NumberOfRoutingShards, "Number of routing shards.")
	a.Describe(&i.Codec, "Compression codec.")
	a.Describe(&i.RoutingPartitionSize, "Routing partition size.")
	a.Describe(&i.SortField, "Sort field(s).")
	a.Describe(&i.SortOrder, "Sort order(s).")
	a.Describe(&i.LoadFixedBitsetFiltersEagerly, "Load fixed bitset filters eagerly.")
	a.Describe(&i.Hidden, "Whether the index is hidden.")
	a.Describe(&i.DeletionProtection, "Prevent deletion on destroy. Defaults to true.")
	a.SetDefault(&i.DeletionProtection, true)
	a.Describe(&i.AdoptOnCreate, "Import existing index into state instead of erroring.")
	a.SetDefault(&i.AdoptOnCreate, false)
	a.Describe(&i.IgnoreShardCountChanges, "Suppress diff on numberOfShards since it's immutable after creation.")
	a.SetDefault(&i.IgnoreShardCountChanges, false)
	a.Describe(&i.IgnoreSettingsOnRead, "Setting keys to exclude from diff.")
	a.Describe(&i.WaitForActiveShards, "Wait for active shards.")
	a.Describe(&i.MasterTimeout, "Master node timeout.")
	a.Describe(&i.Timeout, "Request timeout.")
}

// Create provisions a new Elasticsearch index.
func (r *Index) Create(
	ctx context.Context, req infer.CreateRequest[Inputs],
) (infer.CreateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	indexName := req.Inputs.Name

	// adoptOnCreate: check if the index already exists
	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/"+indexName)
		if err != nil {
			return infer.CreateResponse[State]{}, fmt.Errorf("failed to check if index exists: %w", err)
		}
		if exists {
			state, err := readIndex(ctx, esClient, indexName, req.Inputs)
			if err != nil {
				return infer.CreateResponse[State]{}, err
			}
			return infer.CreateResponse[State]{
				ID:     indexName,
				Output: state,
			}, nil
		}
	}

	body := buildIndexBody(req.Inputs)

	if err := esClient.PutJSON(ctx, "/"+indexName, body, nil); err != nil {
		return infer.CreateResponse[State]{}, fmt.Errorf("failed to create index %s: %w", indexName, err)
	}

	state, err := readIndex(ctx, esClient, indexName, req.Inputs)
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	return infer.CreateResponse[State]{
		ID:     indexName,
		Output: state,
	}, nil
}

// Read fetches the current state of the Elasticsearch index.
func (r *Index) Read(
	ctx context.Context, req infer.ReadRequest[Inputs, State],
) (infer.ReadResponse[Inputs, State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}

	state, err := readIndex(ctx, esClient, req.ID, req.Inputs)
	if err != nil {
		if clients.IsNotFound(err) {
			// Index was deleted externally
			return infer.ReadResponse[Inputs, State]{
				ID: "",
			}, nil
		}
		return infer.ReadResponse[Inputs, State]{}, err
	}

	return infer.ReadResponse[Inputs, State]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

// Update modifies an existing Elasticsearch index.
func (r *Index) Update(
	ctx context.Context, req infer.UpdateRequest[Inputs, State],
) (infer.UpdateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	indexName := req.Inputs.Name

	// Update settings (only mutable settings)
	if req.Inputs.Settings != nil || req.Inputs.NumberOfReplicas != nil {
		settings := buildMutableSettings(req.Inputs)
		if len(settings) > 0 {
			settingsBody := map[string]any{"index": settings}
			if err := esClient.PutJSON(ctx, "/"+indexName+"/_settings", settingsBody, nil); err != nil {
				return infer.UpdateResponse[State]{}, fmt.Errorf("failed to update index settings: %w", err)
			}
		}
	}

	// Update mappings
	if req.Inputs.Mappings != nil {
		var mappings any
		if err := json.Unmarshal([]byte(*req.Inputs.Mappings), &mappings); err != nil {
			return infer.UpdateResponse[State]{}, fmt.Errorf("failed to parse mappings JSON: %w", err)
		}
		if err := esClient.PutJSON(ctx, "/"+indexName+"/_mapping", mappings, nil); err != nil {
			return infer.UpdateResponse[State]{}, fmt.Errorf("failed to update index mappings: %w", err)
		}
	}

	// Update aliases
	if len(req.Inputs.Aliases) > 0 {
		if err := updateAliases(ctx, esClient, indexName, req.Inputs.Aliases); err != nil {
			return infer.UpdateResponse[State]{}, err
		}
	}

	state, err := readIndex(ctx, esClient, indexName, req.Inputs)
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	return infer.UpdateResponse[State]{
		Output: state,
	}, nil
}

// Delete removes the Elasticsearch index.
func (r *Index) Delete(ctx context.Context, req infer.DeleteRequest[State]) (infer.DeleteResponse, error) {
	// Check deletion protection
	if req.State.DeletionProtection != nil && *req.State.DeletionProtection {
		p.GetLogger(ctx).Warning("Index has deletionProtection enabled; " +
			"skipping deletion. Set deletionProtection: false to allow deletion.")
		return infer.DeleteResponse{}, nil
	}

	// Check global destroy protection
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		p.GetLogger(ctx).Warning("Provider-level destroyProtection is enabled; skipping index deletion.")
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/"+req.State.Name); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete index %s: %w", req.State.Name, err)
	}

	return infer.DeleteResponse{}, nil
}

// Diff computes the difference between old and new state.
func (r *Index) Diff(_ context.Context, req infer.DiffRequest[Inputs, State]) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}

	if req.Inputs.Name != req.State.Name {
		diff["name"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}

	// numberOfShards is immutable — force replace if changed (unless ignored)
	if !req.Inputs.IgnoreShardCountChanges && req.Inputs.NumberOfShards != nil && req.State.NumberOfShards != nil {
		if *req.Inputs.NumberOfShards != *req.State.NumberOfShards {
			diff["numberOfShards"] = p.PropertyDiff{Kind: p.UpdateReplace}
		}
	}

	return p.DiffResponse{
		HasChanges:   len(diff) > 0,
		DetailedDiff: diff,
	}, nil
}

func buildIndexBody(inputs Inputs) map[string]any {
	body := map[string]any{}

	// Settings
	settings := map[string]any{}
	if inputs.NumberOfShards != nil {
		settings["number_of_shards"] = *inputs.NumberOfShards
	}
	if inputs.NumberOfReplicas != nil {
		settings["number_of_replicas"] = *inputs.NumberOfReplicas
	}
	if inputs.NumberOfRoutingShards != nil {
		settings["number_of_routing_shards"] = *inputs.NumberOfRoutingShards
	}
	if inputs.Codec != nil {
		settings["codec"] = *inputs.Codec
	}
	if inputs.RoutingPartitionSize != nil {
		settings["routing_partition_size"] = *inputs.RoutingPartitionSize
	}
	if len(inputs.SortField) > 0 {
		settings["sort.field"] = inputs.SortField
		settings["sort.order"] = inputs.SortOrder
	}
	if inputs.LoadFixedBitsetFiltersEagerly != nil {
		settings["load_fixed_bitset_filters_eagerly"] = *inputs.LoadFixedBitsetFiltersEagerly
	}
	if inputs.Hidden != nil {
		settings["hidden"] = *inputs.Hidden
	}

	// Merge raw settings
	if inputs.Settings != nil {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(*inputs.Settings), &parsed); err == nil {
			for k, v := range parsed {
				settings[k] = v
			}
		}
	}
	if inputs.SettingsRaw != nil {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(*inputs.SettingsRaw), &parsed); err == nil {
			for k, v := range parsed {
				settings[k] = v
			}
		}
	}

	if len(settings) > 0 {
		body["settings"] = settings
	}

	// Mappings
	if inputs.Mappings != nil {
		var mappings any
		if err := json.Unmarshal([]byte(*inputs.Mappings), &mappings); err == nil {
			body["mappings"] = mappings
		}
	}

	// Aliases
	if len(inputs.Aliases) > 0 {
		aliases := map[string]any{}
		for _, a := range inputs.Aliases {
			aliasBody := map[string]any{}
			if a.Filter != nil {
				var filter any
				if err := json.Unmarshal([]byte(*a.Filter), &filter); err == nil {
					aliasBody["filter"] = filter
				}
			}
			if a.Routing != nil {
				aliasBody["routing"] = *a.Routing
			}
			if a.IndexRouting != nil {
				aliasBody["index_routing"] = *a.IndexRouting
			}
			if a.SearchRouting != nil {
				aliasBody["search_routing"] = *a.SearchRouting
			}
			if a.IsWriteIndex != nil {
				aliasBody["is_write_index"] = *a.IsWriteIndex
			}
			if a.IsHidden != nil {
				aliasBody["is_hidden"] = *a.IsHidden
			}
			aliases[a.Name] = aliasBody
		}
		body["aliases"] = aliases
	}

	return body
}

func buildMutableSettings(inputs Inputs) map[string]any {
	settings := map[string]any{}
	if inputs.NumberOfReplicas != nil {
		settings["number_of_replicas"] = *inputs.NumberOfReplicas
	}
	// Additional mutable settings can be merged from Settings JSON
	if inputs.Settings != nil {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(*inputs.Settings), &parsed); err == nil {
			for k, v := range parsed {
				settings[k] = v
			}
		}
	}
	return settings
}

func readIndex(
	ctx context.Context, esClient *clients.ElasticsearchClient,
	indexName string, inputs Inputs,
) (State, error) {
	var result map[string]json.RawMessage
	if err := esClient.GetJSON(ctx, "/"+indexName, &result); err != nil {
		return State{}, err
	}

	// Response is { "index-name": { "settings": {...}, "mappings": {...}, ... } }
	indexData, ok := result[indexName]
	if !ok {
		return State{}, fmt.Errorf("index %s not found in response", indexName)
	}

	var indexInfo struct {
		Settings struct {
			Index map[string]any `json:"index"`
		} `json:"settings"`
		Mappings json.RawMessage `json:"mappings"`
	}
	if err := json.Unmarshal(indexData, &indexInfo); err != nil {
		return State{}, fmt.Errorf("failed to parse index info: %w", err)
	}

	state := State{
		Inputs: inputs,
	}

	if uuid, ok := indexInfo.Settings.Index["uuid"].(string); ok {
		state.UUID = uuid
	}

	return state, nil
}

func updateAliases(
	ctx context.Context, esClient *clients.ElasticsearchClient,
	indexName string, aliases []Alias,
) error {
	actions := []any{}
	for _, a := range aliases {
		aliasBody := map[string]any{
			"index": indexName,
			"alias": a.Name,
		}
		if a.Filter != nil {
			var filter any
			if err := json.Unmarshal([]byte(*a.Filter), &filter); err == nil {
				aliasBody["filter"] = filter
			}
		}
		if a.Routing != nil {
			aliasBody["routing"] = *a.Routing
		}
		if a.IsWriteIndex != nil {
			aliasBody["is_write_index"] = *a.IsWriteIndex
		}
		actions = append(actions, map[string]any{"add": aliasBody})
	}

	body := map[string]any{"actions": actions}
	return esClient.PostJSON(ctx, "/_aliases", body, nil)
}

func boolPtr(b bool) *bool {
	return &b
}
