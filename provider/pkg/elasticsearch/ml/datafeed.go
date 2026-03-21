package ml

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Datafeed manages an ML datafeed via PUT /_ml/datafeeds/<datafeed_id>.
type Datafeed struct{}

// DatafeedInputs ...
type DatafeedInputs struct {
	DatafeedId             string   `pulumi:"datafeedId"`
	JobId                  string   `pulumi:"jobId"`
	Indices                []string `pulumi:"indices"`
	Query                  *string  `pulumi:"query,optional"`
	Frequency              *string  `pulumi:"frequency,optional"`
	QueryDelay             *string  `pulumi:"queryDelay,optional"`
	MaxEmptySearches       *int     `pulumi:"maxEmptySearches,optional"`
	ScrollSize             *int     `pulumi:"scrollSize,optional"`
	ChunkingConfig         *string  `pulumi:"chunkingConfig,optional"`
	DelayedDataCheckConfig *string  `pulumi:"delayedDataCheckConfig,optional"`
	IndicesOptions         *string  `pulumi:"indicesOptions,optional"`
	RuntimeMappings        *string  `pulumi:"runtimeMappings,optional"`
	ScriptFields           *string  `pulumi:"scriptFields,optional"`
	AdoptOnCreate          bool     `pulumi:"adoptOnCreate,optional"`
}

// DatafeedState ...
type DatafeedState struct {
	DatafeedInputs
}

var (
	_ infer.CustomDelete[DatafeedState]                 = (*Datafeed)(nil)
	_ infer.CustomRead[DatafeedInputs, DatafeedState]   = (*Datafeed)(nil)
	_ infer.CustomUpdate[DatafeedInputs, DatafeedState] = (*Datafeed)(nil)
)

// Annotate ...
func (r *Datafeed) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch ML datafeed.")
	a.SetToken("elasticsearch", "Datafeed")
}

// Annotate ...
func (i *DatafeedInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.DatafeedId, "The unique identifier for the datafeed.")
	a.Describe(&i.JobId, "The identifier for the anomaly detection job that the datafeed sends data to.")
	a.Describe(&i.Indices, "An array of index names to search for the data.")
	a.Describe(&i.Query, "The Elasticsearch query DSL as JSON used to search for data.")
	a.Describe(
		&i.Frequency,
		"The interval at which scheduled queries are made while the datafeed runs in real time (e.g. '60s').",
	)
	a.Describe(&i.QueryDelay, "The number of seconds behind real time that data is queried (e.g. '60s').")
	a.Describe(
		&i.MaxEmptySearches,
		"If a real-time datafeed has never seen any data, this controls how many empty searches it runs before it stops.",
	)
	a.Describe(
		&i.ScrollSize,
		"The size parameter used in Elasticsearch searches when the datafeed does not use aggregations.",
	)
	a.Describe(
		&i.ChunkingConfig,
		"Chunking configuration as JSON. Controls how data searches are split into time chunks.",
	)
	a.Describe(
		&i.DelayedDataCheckConfig,
		"Delayed data check configuration as JSON. Controls whether the datafeed looks back for missing data.",
	)
	a.Describe(&i.IndicesOptions, "Indices options as JSON. Specifies index expansion options used during search.")
	a.Describe(&i.RuntimeMappings, "Runtime mappings as JSON. Specifies runtime fields for the datafeed search.")
	a.Describe(
		&i.ScriptFields,
		"Script fields as JSON. Specifies scripts that evaluate custom expressions at query time.",
	)
	a.Describe(&i.AdoptOnCreate, "Adopt an existing datafeed into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Datafeed) Create(
	ctx context.Context,
	req infer.CreateRequest[DatafeedInputs],
) (infer.CreateResponse[DatafeedState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[DatafeedState]{}, err
	}

	datafeedId := req.Inputs.DatafeedId

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_ml/datafeeds/"+datafeedId)
		if err != nil {
			return infer.CreateResponse[DatafeedState]{}, err
		}
		if exists {
			body, err := buildDatafeedBody(req.Inputs)
			if err != nil {
				return infer.CreateResponse[DatafeedState]{}, err
			}
			if err := esClient.PostJSON(ctx, "/_ml/datafeeds/"+datafeedId+"/_update", body, nil); err != nil {
				return infer.CreateResponse[DatafeedState]{}, fmt.Errorf(
					"failed to update adopted datafeed %s: %w",
					datafeedId,
					err,
				)
			}
			return infer.CreateResponse[DatafeedState]{
				ID:     datafeedId,
				Output: DatafeedState{DatafeedInputs: req.Inputs},
			}, nil
		}
	}

	body, err := buildDatafeedBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[DatafeedState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_ml/datafeeds/"+datafeedId, body, nil); err != nil {
		return infer.CreateResponse[DatafeedState]{}, fmt.Errorf("failed to create datafeed %s: %w", datafeedId, err)
	}

	return infer.CreateResponse[DatafeedState]{
		ID:     datafeedId,
		Output: DatafeedState{DatafeedInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *Datafeed) Read(
	ctx context.Context,
	req infer.ReadRequest[DatafeedInputs, DatafeedState],
) (infer.ReadResponse[DatafeedInputs, DatafeedState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[DatafeedInputs, DatafeedState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_ml/datafeeds/"+req.ID)
	if err != nil {
		return infer.ReadResponse[DatafeedInputs, DatafeedState]{}, err
	}
	if !exists {
		return infer.ReadResponse[DatafeedInputs, DatafeedState]{ID: ""}, nil
	}

	return infer.ReadResponse[DatafeedInputs, DatafeedState](req), nil
}

// Update ...
func (r *Datafeed) Update(
	ctx context.Context,
	req infer.UpdateRequest[DatafeedInputs, DatafeedState],
) (infer.UpdateResponse[DatafeedState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[DatafeedState]{}, err
	}

	body, err := buildDatafeedBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[DatafeedState]{}, err
	}
	if err := esClient.PostJSON(ctx, "/_ml/datafeeds/"+req.Inputs.DatafeedId+"/_update", body, nil); err != nil {
		return infer.UpdateResponse[DatafeedState]{}, fmt.Errorf(
			"failed to update datafeed %s: %w",
			req.Inputs.DatafeedId,
			err,
		)
	}

	return infer.UpdateResponse[DatafeedState]{
		Output: DatafeedState{DatafeedInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *Datafeed) Delete(
	ctx context.Context,
	req infer.DeleteRequest[DatafeedState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_ml/datafeeds/"+req.State.DatafeedId+"?force=true"); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildDatafeedBody(inputs DatafeedInputs) (map[string]any, error) {
	body := map[string]any{
		"job_id":  inputs.JobId,
		"indices": inputs.Indices,
	}

	if inputs.Query != nil {
		var query any
		if err := json.Unmarshal([]byte(*inputs.Query), &query); err != nil {
			return nil, fmt.Errorf("invalid query JSON: %w", err)
		}
		body["query"] = query
	}
	if inputs.Frequency != nil {
		body["frequency"] = *inputs.Frequency
	}
	if inputs.QueryDelay != nil {
		body["query_delay"] = *inputs.QueryDelay
	}
	if inputs.MaxEmptySearches != nil {
		body["max_empty_searches"] = *inputs.MaxEmptySearches
	}
	if inputs.ScrollSize != nil {
		body["scroll_size"] = *inputs.ScrollSize
	}
	if inputs.ChunkingConfig != nil {
		var chunkingConfig any
		if err := json.Unmarshal([]byte(*inputs.ChunkingConfig), &chunkingConfig); err != nil {
			return nil, fmt.Errorf("invalid chunkingConfig JSON: %w", err)
		}
		body["chunking_config"] = chunkingConfig
	}
	if inputs.DelayedDataCheckConfig != nil {
		var delayedDataCheckConfig any
		if err := json.Unmarshal([]byte(*inputs.DelayedDataCheckConfig), &delayedDataCheckConfig); err != nil {
			return nil, fmt.Errorf("invalid delayedDataCheckConfig JSON: %w", err)
		}
		body["delayed_data_check_config"] = delayedDataCheckConfig
	}
	if inputs.IndicesOptions != nil {
		var indicesOptions any
		if err := json.Unmarshal([]byte(*inputs.IndicesOptions), &indicesOptions); err != nil {
			return nil, fmt.Errorf("invalid indicesOptions JSON: %w", err)
		}
		body["indices_options"] = indicesOptions
	}
	if inputs.RuntimeMappings != nil {
		var runtimeMappings any
		if err := json.Unmarshal([]byte(*inputs.RuntimeMappings), &runtimeMappings); err != nil {
			return nil, fmt.Errorf("invalid runtimeMappings JSON: %w", err)
		}
		body["runtime_mappings"] = runtimeMappings
	}
	if inputs.ScriptFields != nil {
		var scriptFields any
		if err := json.Unmarshal([]byte(*inputs.ScriptFields), &scriptFields); err != nil {
			return nil, fmt.Errorf("invalid scriptFields JSON: %w", err)
		}
		body["script_fields"] = scriptFields
	}

	return body, nil
}
