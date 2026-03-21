package dataview

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

const jsonNull = "null"

// DataView manages a Kibana data view (index pattern) via the Data Views API.
type DataView struct{}

// Inputs ...
type Inputs struct {
	Title           string   `pulumi:"title"`
	Name            *string  `pulumi:"name,optional"`
	TimeFieldName   *string  `pulumi:"timeFieldName,optional"`
	SourceFilters   *string  `pulumi:"sourceFilters,optional"`
	FieldFormats    *string  `pulumi:"fieldFormats,optional"`
	FieldAttrs      *string  `pulumi:"fieldAttrs,optional"`
	RuntimeFieldMap *string  `pulumi:"runtimeFieldMap,optional"`
	AllowNoIndex    *bool    `pulumi:"allowNoIndex,optional"`
	Namespaces      []string `pulumi:"namespaces,optional"`
	SpaceID         *string  `pulumi:"spaceID,optional"`
	Override        *bool    `pulumi:"override,optional"`
	AdoptOnCreate   bool     `pulumi:"adoptOnCreate,optional"`
}

// State ...
type State struct {
	Inputs

	// Computed outputs
	DataViewID string `pulumi:"dataViewId"`
}

var (
	_ infer.CustomDelete[State]         = (*DataView)(nil)
	_ infer.CustomRead[Inputs, State]   = (*DataView)(nil)
	_ infer.CustomUpdate[Inputs, State] = (*DataView)(nil)
)

// Annotate ...
func (r *DataView) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana data view (index pattern).")
	a.SetToken("kibana", "DataView")
}

// Annotate ...
func (i *Inputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Title, "The index pattern (e.g. 'logs-*').")
	a.Describe(&i.Name, "Human-readable name for the data view.")
	a.Describe(&i.TimeFieldName, "The timestamp field name.")
	a.Describe(&i.SourceFilters, "Source filters as a JSON array.")
	a.Describe(&i.FieldFormats, "Field format configuration as JSON.")
	a.Describe(&i.FieldAttrs, "Field attributes as JSON.")
	a.Describe(&i.RuntimeFieldMap, "Runtime field definitions as JSON.")
	a.Describe(&i.AllowNoIndex, "Allow the data view to exist without matching indices.")
	a.Describe(&i.Namespaces, "Namespaces for the data view.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.Override, "Override an existing data view if a conflict occurs on create.")
	a.Describe(&i.AdoptOnCreate, "If true and the data view already exists, adopt it into state instead of failing.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *DataView) Create(
	ctx context.Context,
	req infer.CreateRequest[Inputs],
) (infer.CreateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	spaceID := derefString(req.Inputs.SpaceID)
	path := clients.SpacePath(spaceID, "/api/data_views/data_view")

	body, err := buildDataViewBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	// AdoptOnCreate not supported for data views (server-generated IDs).

	var result struct {
		DataView struct {
			ID string `json:"id"`
		} `json:"data_view"`
	}

	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[State]{}, fmt.Errorf("failed to create data view: %w", err)
	}

	return infer.CreateResponse[State]{
		ID: result.DataView.ID,
		Output: State{
			Inputs:     req.Inputs,
			DataViewID: result.DataView.ID,
		},
	}, nil
}

// Read ...
func (r *DataView) Read(
	ctx context.Context,
	req infer.ReadRequest[Inputs, State],
) (infer.ReadResponse[Inputs, State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}

	spaceID := derefString(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/data_views/data_view/"+req.ID)

	var result struct {
		DataView struct {
			ID              string          `json:"id"`
			Title           string          `json:"title"`
			Name            *string         `json:"name"`
			TimeFieldName   *string         `json:"timeFieldName"`
			SourceFilters   json.RawMessage `json:"sourceFilters"`
			FieldFormats    json.RawMessage `json:"fieldFormats"`
			FieldAttrs      json.RawMessage `json:"fieldAttrs"`
			RuntimeFieldMap json.RawMessage `json:"runtimeFieldMap"`
			AllowNoIndex    *bool           `json:"allowNoIndex"`
			Namespaces      []string        `json:"namespaces"`
		} `json:"data_view"`
	}

	if err := kbClient.GetJSON(ctx, path, &result); err != nil {
		if clients.IsNotFound(err) {
			return infer.ReadResponse[Inputs, State]{ID: ""}, nil
		}
		return infer.ReadResponse[Inputs, State]{}, err
	}

	inputs := req.Inputs
	inputs.Title = result.DataView.Title
	inputs.Name = result.DataView.Name
	inputs.TimeFieldName = result.DataView.TimeFieldName
	if len(result.DataView.SourceFilters) > 0 && string(result.DataView.SourceFilters) != jsonNull {
		s := string(result.DataView.SourceFilters)
		inputs.SourceFilters = &s
	}
	if len(result.DataView.FieldFormats) > 0 && string(result.DataView.FieldFormats) != jsonNull {
		s := string(result.DataView.FieldFormats)
		inputs.FieldFormats = &s
	}
	if len(result.DataView.FieldAttrs) > 0 && string(result.DataView.FieldAttrs) != jsonNull {
		s := string(result.DataView.FieldAttrs)
		inputs.FieldAttrs = &s
	}
	if len(result.DataView.RuntimeFieldMap) > 0 && string(result.DataView.RuntimeFieldMap) != jsonNull {
		s := string(result.DataView.RuntimeFieldMap)
		inputs.RuntimeFieldMap = &s
	}
	inputs.AllowNoIndex = result.DataView.AllowNoIndex
	if len(result.DataView.Namespaces) > 0 {
		inputs.Namespaces = result.DataView.Namespaces
	}

	state := State{
		Inputs:     inputs,
		DataViewID: result.DataView.ID,
	}

	return infer.ReadResponse[Inputs, State]{
		ID:     req.ID,
		Inputs: inputs,
		State:  state,
	}, nil
}

// Update ...
func (r *DataView) Update(
	ctx context.Context,
	req infer.UpdateRequest[Inputs, State],
) (infer.UpdateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	spaceID := derefString(req.Inputs.SpaceID)
	path := clients.SpacePath(spaceID, "/api/data_views/data_view/"+req.ID)

	body, err := buildDataViewBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	var result struct {
		DataView struct {
			ID string `json:"id"`
		} `json:"data_view"`
	}

	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.UpdateResponse[State]{}, fmt.Errorf("failed to update data view %s: %w", req.ID, err)
	}

	return infer.UpdateResponse[State]{
		Output: State{
			Inputs:     req.Inputs,
			DataViewID: result.DataView.ID,
		},
	}, nil
}

// Delete ...
func (r *DataView) Delete(ctx context.Context, req infer.DeleteRequest[State]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := derefString(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/data_views/data_view/"+req.State.DataViewID)

	if err := kbClient.Delete(ctx, path); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete data view %s: %w", req.State.DataViewID, err)
	}

	return infer.DeleteResponse{}, nil
}

func buildDataViewBody(inputs Inputs) (map[string]any, error) {
	dv := map[string]any{
		"title": inputs.Title,
	}

	if inputs.Name != nil {
		dv["name"] = *inputs.Name
	}
	if inputs.TimeFieldName != nil {
		dv["timeFieldName"] = *inputs.TimeFieldName
	}
	if inputs.SourceFilters != nil {
		var sf any
		if err := json.Unmarshal([]byte(*inputs.SourceFilters), &sf); err != nil {
			return nil, fmt.Errorf("invalid sf JSON: %w", err)
		}
		dv["sourceFilters"] = sf
	}
	if inputs.FieldFormats != nil {
		var ff any
		if err := json.Unmarshal([]byte(*inputs.FieldFormats), &ff); err != nil {
			return nil, fmt.Errorf("invalid ff JSON: %w", err)
		}
		dv["fieldFormats"] = ff
	}
	if inputs.FieldAttrs != nil {
		var fa any
		if err := json.Unmarshal([]byte(*inputs.FieldAttrs), &fa); err != nil {
			return nil, fmt.Errorf("invalid fa JSON: %w", err)
		}
		dv["fieldAttrs"] = fa
	}
	if inputs.RuntimeFieldMap != nil {
		var rfm any
		if err := json.Unmarshal([]byte(*inputs.RuntimeFieldMap), &rfm); err != nil {
			return nil, fmt.Errorf("invalid rfm JSON: %w", err)
		}
		dv["runtimeFieldMap"] = rfm
	}
	if inputs.AllowNoIndex != nil {
		dv["allowNoIndex"] = *inputs.AllowNoIndex
	}
	if len(inputs.Namespaces) > 0 {
		dv["namespaces"] = inputs.Namespaces
	}

	body := map[string]any{
		"data_view": dv,
	}

	if inputs.Override != nil && *inputs.Override {
		body["override"] = true
	}

	return body, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
