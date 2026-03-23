package dataview

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// DefaultDataView sets the default data view for a Kibana space.
type DefaultDataView struct{}

// DefaultInputs ...
type DefaultInputs struct {
	DataViewID string  `pulumi:"dataViewId"`
	Force      *bool   `pulumi:"force,optional"`
	SpaceID    *string `pulumi:"spaceID,optional"`
}

// DefaultState ...
type DefaultState struct {
	DefaultInputs
}

var _ infer.CustomDelete[DefaultState] = (*DefaultDataView)(nil)

// Annotate ...
func (r *DefaultDataView) Annotate(a infer.Annotator) {
	a.Describe(r, "Sets the default data view for a Kibana space.")
	a.SetToken("kibana", "DefaultDataView")
}

// Annotate ...
func (i *DefaultInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.DataViewID, "The ID of the data view to set as default.")
	a.Describe(&i.Force, "Force setting the default data view.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
}

// Create ...
func (r *DefaultDataView) Create(
	ctx context.Context,
	req infer.CreateRequest[DefaultInputs],
) (infer.CreateResponse[DefaultState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[DefaultState]{}, err
	}

	spaceID := derefString(req.Inputs.SpaceID)
	path := clients.SpacePath(spaceID, "/api/data_views/default")

	body := map[string]any{
		"data_view_id": req.Inputs.DataViewID,
	}
	if req.Inputs.Force != nil {
		body["force"] = *req.Inputs.Force
	}

	if err := kbClient.PostJSON(ctx, path, body, nil); err != nil {
		return infer.CreateResponse[DefaultState]{}, fmt.Errorf("failed to set default data view: %w", err)
	}

	return infer.CreateResponse[DefaultState]{
		ID: req.Inputs.DataViewID,
		Output: DefaultState{
			DefaultInputs: req.Inputs,
		},
	}, nil
}

// Delete ...
func (r *DefaultDataView) Delete(
	_ context.Context,
	_ infer.DeleteRequest[DefaultState],
) (infer.DeleteResponse, error) {
	// Cannot unset a default data view — this is a no-op.
	return infer.DeleteResponse{}, nil
}
