// Package index implements Elasticsearch index and data stream management.
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

// DataStream manages an Elasticsearch data stream via PUT /_data_stream/<name>.
type DataStream struct{}

// DataStreamInputs defines the input properties for a data stream.
type DataStreamInputs struct {
	Name               string `pulumi:"name"`
	AdoptOnCreate      bool   `pulumi:"adoptOnCreate,optional"`
	DeletionProtection *bool  `pulumi:"deletionProtection,optional"`
}

// DataStreamState defines the output state for a data stream.
type DataStreamState struct {
	DataStreamInputs

	TimestampField string `pulumi:"timestampField"`
	Generation     int    `pulumi:"generation"`
	Status         string `pulumi:"status"`
}

var (
	_ infer.CustomDelete[DataStreamState]                 = (*DataStream)(nil)
	_ infer.CustomRead[DataStreamInputs, DataStreamState] = (*DataStream)(nil)
	_ infer.CustomDiff[DataStreamInputs, DataStreamState] = (*DataStream)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *DataStream) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch data stream.")
	a.SetToken("elasticsearch", "DataStream")
}

// Annotate sets input property descriptions and defaults.
func (i *DataStreamInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the data stream.")
	a.Describe(&i.AdoptOnCreate, "Import existing data stream into state instead of erroring.")
	a.SetDefault(&i.AdoptOnCreate, false)
	a.Describe(&i.DeletionProtection, "Prevent deletion on destroy. Defaults to true.")
	a.SetDefault(&i.DeletionProtection, true)
}

// Create provisions a new data stream.
func (r *DataStream) Create(
	ctx context.Context, req infer.CreateRequest[DataStreamInputs],
) (infer.CreateResponse[DataStreamState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[DataStreamState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_data_stream/"+name)
		if err != nil {
			return infer.CreateResponse[DataStreamState]{}, err
		}
		if exists {
			state, err := readDataStream(ctx, esClient, name, req.Inputs)
			if err != nil {
				return infer.CreateResponse[DataStreamState]{}, err
			}
			return infer.CreateResponse[DataStreamState]{ID: name, Output: state}, nil
		}
	}

	if err := esClient.PutJSON(ctx, "/_data_stream/"+name, nil, nil); err != nil {
		return infer.CreateResponse[DataStreamState]{}, fmt.Errorf("failed to create data stream %s: %w", name, err)
	}

	state, err := readDataStream(ctx, esClient, name, req.Inputs)
	if err != nil {
		return infer.CreateResponse[DataStreamState]{}, err
	}

	return infer.CreateResponse[DataStreamState]{ID: name, Output: state}, nil
}

// Read fetches the current state of the data stream.
func (r *DataStream) Read(
	ctx context.Context, req infer.ReadRequest[DataStreamInputs, DataStreamState],
) (infer.ReadResponse[DataStreamInputs, DataStreamState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[DataStreamInputs, DataStreamState]{}, err
	}

	state, err := readDataStream(ctx, esClient, req.ID, req.Inputs)
	if err != nil {
		if clients.IsNotFound(err) {
			return infer.ReadResponse[DataStreamInputs, DataStreamState]{ID: ""}, nil
		}
		return infer.ReadResponse[DataStreamInputs, DataStreamState]{}, err
	}

	return infer.ReadResponse[DataStreamInputs, DataStreamState]{
		ID: req.ID, Inputs: req.Inputs, State: state,
	}, nil
}

// Delete removes the data stream.
func (r *DataStream) Delete(
	ctx context.Context, req infer.DeleteRequest[DataStreamState],
) (infer.DeleteResponse, error) {
	if req.State.DeletionProtection != nil && *req.State.DeletionProtection {
		p.GetLogger(ctx).Warning("Data stream has deletionProtection enabled; skipping deletion.")
		return infer.DeleteResponse{}, nil
	}

	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		p.GetLogger(ctx).Warning("Provider-level destroyProtection is enabled; skipping data stream deletion.")
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_data_stream/"+req.State.Name); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete data stream %s: %w", req.State.Name, err)
	}

	return infer.DeleteResponse{}, nil
}

// Diff computes the difference between old and new state.
func (r *DataStream) Diff(
	_ context.Context, req infer.DiffRequest[DataStreamInputs, DataStreamState],
) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}
	if req.Inputs.Name != req.State.Name {
		diff["name"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	return p.DiffResponse{HasChanges: len(diff) > 0, DetailedDiff: diff}, nil
}

func readDataStream(
	ctx context.Context, esClient *clients.ElasticsearchClient,
	name string, inputs DataStreamInputs,
) (DataStreamState, error) {
	var result struct {
		DataStreams []struct {
			Name           string `json:"name"`
			TimestampField struct {
				Name string `json:"name"`
			} `json:"timestamp_field"`
			Generation int    `json:"generation"`
			Status     string `json:"status"`
		} `json:"data_streams"`
	}

	if err := esClient.GetJSON(ctx, "/_data_stream/"+name, &result); err != nil {
		return DataStreamState{}, err
	}

	if len(result.DataStreams) == 0 {
		return DataStreamState{}, &clients.NotFoundError{Path: "/_data_stream/" + name}
	}

	ds := result.DataStreams[0]
	return DataStreamState{
		DataStreamInputs: inputs,
		TimestampField:   ds.TimestampField.Name,
		Generation:       ds.Generation,
		Status:           ds.Status,
	}, nil
}

// DataStreamLifecycle manages data stream lifecycle settings.
type DataStreamLifecycle struct{}

// DataStreamLifecycleInputs defines the input properties for data stream lifecycle configuration.
type DataStreamLifecycleInputs struct {
	Name          string              `pulumi:"name"`
	DataRetention *string             `pulumi:"dataRetention,optional"`
	Downsampling  []DownsamplingRound `pulumi:"downsampling,optional"`
	Enabled       *bool               `pulumi:"enabled,optional"`
}

// DownsamplingRound defines a downsampling round configuration for data stream lifecycle.
type DownsamplingRound struct {
	After         string `pulumi:"after"`
	FixedInterval string `pulumi:"fixedInterval"`
}

// DataStreamLifecycleState defines the output state for data stream lifecycle configuration.
type DataStreamLifecycleState struct {
	DataStreamLifecycleInputs
}

var (
	_ infer.CustomDelete[DataStreamLifecycleState]                            = (*DataStreamLifecycle)(nil)
	_ infer.CustomUpdate[DataStreamLifecycleInputs, DataStreamLifecycleState] = (*DataStreamLifecycle)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *DataStreamLifecycle) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages the lifecycle configuration of an Elasticsearch data stream.")
	a.SetToken("elasticsearch", "DataStreamLifecycle")
}

// Annotate sets input property descriptions and defaults.
func (i *DataStreamLifecycleInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the data stream.")
	a.Describe(&i.DataRetention, "Data retention period (e.g. '30d').")
	a.Describe(&i.Downsampling, "Downsampling rounds configuration.")
	a.Describe(&i.Enabled, "Whether lifecycle management is enabled.")
	a.SetDefault(&i.Enabled, boolPtr(true))
}

// Create provisions a new data stream lifecycle configuration.
func (r *DataStreamLifecycle) Create(
	ctx context.Context, req infer.CreateRequest[DataStreamLifecycleInputs],
) (infer.CreateResponse[DataStreamLifecycleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[DataStreamLifecycleState]{}, err
	}

	body := buildLifecycleBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_data_stream/"+req.Inputs.Name+"/_lifecycle", body, nil); err != nil {
		return infer.CreateResponse[DataStreamLifecycleState]{}, fmt.Errorf(
			"failed to set data stream lifecycle: %w",
			err,
		)
	}

	return infer.CreateResponse[DataStreamLifecycleState]{
		ID:     req.Inputs.Name,
		Output: DataStreamLifecycleState{DataStreamLifecycleInputs: req.Inputs},
	}, nil
}

// Update modifies an existing data stream lifecycle configuration.
func (r *DataStreamLifecycle) Update(
	ctx context.Context,
	req infer.UpdateRequest[DataStreamLifecycleInputs, DataStreamLifecycleState],
) (infer.UpdateResponse[DataStreamLifecycleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[DataStreamLifecycleState]{}, err
	}

	body := buildLifecycleBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_data_stream/"+req.Inputs.Name+"/_lifecycle", body, nil); err != nil {
		return infer.UpdateResponse[DataStreamLifecycleState]{}, fmt.Errorf(
			"failed to update data stream lifecycle: %w",
			err,
		)
	}

	return infer.UpdateResponse[DataStreamLifecycleState]{
		Output: DataStreamLifecycleState{DataStreamLifecycleInputs: req.Inputs},
	}, nil
}

// Delete removes the data stream lifecycle configuration.
func (r *DataStreamLifecycle) Delete(
	ctx context.Context, req infer.DeleteRequest[DataStreamLifecycleState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_data_stream/"+req.State.Name+"/_lifecycle"); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete data stream lifecycle: %w", err)
	}

	return infer.DeleteResponse{}, nil
}

func buildLifecycleBody(inputs DataStreamLifecycleInputs) map[string]any {
	body := map[string]any{}
	lifecycle := map[string]any{}

	if inputs.DataRetention != nil {
		lifecycle["data_retention"] = *inputs.DataRetention
	}
	if inputs.Enabled != nil {
		lifecycle["enabled"] = *inputs.Enabled
	}
	if len(inputs.Downsampling) > 0 {
		rounds := []map[string]any{}
		for _, ds := range inputs.Downsampling {
			rounds = append(rounds, map[string]any{
				"after":          ds.After,
				"fixed_interval": ds.FixedInterval,
			})
		}
		lifecycle["downsampling"] = map[string]any{"rounds": rounds}
	}

	body["lifecycle"] = lifecycle
	return body
}

// Ensure json import is used
var _ = json.Marshal
