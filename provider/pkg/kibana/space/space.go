package space

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Space manages a Kibana space via the /api/spaces/space API.
type Space struct{}

// Inputs ...
type Inputs struct {
	SpaceID          string   `pulumi:"spaceId"`
	Name             string   `pulumi:"name"`
	Description      *string  `pulumi:"description,optional"`
	Color            *string  `pulumi:"color,optional"`
	Initials         *string  `pulumi:"initials,optional"`
	DisabledFeatures []string `pulumi:"disabledFeatures,optional"`
	ImageUrl         *string  `pulumi:"imageUrl,optional"`
	AdoptOnCreate    bool     `pulumi:"adoptOnCreate,optional"`
}

// State ...
type State struct {
	Inputs
}

var (
	_ infer.CustomDelete[State]         = (*Space)(nil)
	_ infer.CustomRead[Inputs, State]   = (*Space)(nil)
	_ infer.CustomUpdate[Inputs, State] = (*Space)(nil)
)

// Annotate ...
func (r *Space) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana space.")
	a.SetToken("kibana", "Space")
}

// Annotate ...
func (i *Inputs) Annotate(a infer.Annotator) {
	a.Describe(&i.SpaceID, "The space ID.")
	a.Describe(&i.Name, "The display name for the space.")
	a.Describe(&i.Description, "A description for the space.")
	a.Describe(&i.Color, "The hex color code for the space avatar.")
	a.Describe(&i.Initials, "One or two characters shown in the space avatar (defaults to first letters of name).")
	a.Describe(&i.DisabledFeatures, "List of feature IDs to disable in this space.")
	a.Describe(&i.ImageUrl, "Data URL (data:image/...) for the space avatar image.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing space into state instead of failing.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Space) Create(
	ctx context.Context,
	req infer.CreateRequest[Inputs],
) (infer.CreateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[State]{}, err
	}

	spaceID := req.Inputs.SpaceID

	if req.Inputs.AdoptOnCreate {
		exists, err := kbClient.Exists(ctx, "/api/spaces/space/"+spaceID)
		if err != nil {
			return infer.CreateResponse[State]{}, err
		}
		if exists {
			body := buildSpaceBody(req.Inputs)
			if err := kbClient.PutJSON(ctx, "/api/spaces/space/"+spaceID, body, nil); err != nil {
				return infer.CreateResponse[State]{}, fmt.Errorf(
					"failed to update adopted space %s: %w",
					spaceID,
					err,
				)
			}
			return infer.CreateResponse[State]{
				ID:     spaceID,
				Output: State{Inputs: req.Inputs},
			}, nil
		}
	}

	body := buildSpaceBody(req.Inputs)
	if err := kbClient.PostJSON(ctx, "/api/spaces/space", body, nil); err != nil {
		return infer.CreateResponse[State]{}, fmt.Errorf("failed to create space %s: %w", spaceID, err)
	}

	return infer.CreateResponse[State]{
		ID:     spaceID,
		Output: State{Inputs: req.Inputs},
	}, nil
}

// Read ...
func (r *Space) Read(
	ctx context.Context,
	req infer.ReadRequest[Inputs, State],
) (infer.ReadResponse[Inputs, State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}

	exists, err := kbClient.Exists(ctx, "/api/spaces/space/"+req.ID)
	if err != nil {
		return infer.ReadResponse[Inputs, State]{}, err
	}
	if !exists {
		return infer.ReadResponse[Inputs, State]{ID: ""}, nil
	}

	return infer.ReadResponse[Inputs, State](req), nil
}

// Update ...
func (r *Space) Update(
	ctx context.Context,
	req infer.UpdateRequest[Inputs, State],
) (infer.UpdateResponse[State], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[State]{}, err
	}

	body := buildSpaceBody(req.Inputs)
	if err := kbClient.PutJSON(ctx, "/api/spaces/space/"+req.Inputs.SpaceID, body, nil); err != nil {
		return infer.UpdateResponse[State]{}, fmt.Errorf("failed to update space %s: %w", req.Inputs.SpaceID, err)
	}

	return infer.UpdateResponse[State]{
		Output: State{Inputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *Space) Delete(ctx context.Context, req infer.DeleteRequest[State]) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := kbClient.Delete(ctx, "/api/spaces/space/"+req.State.SpaceID); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildSpaceBody(inputs Inputs) map[string]any {
	body := map[string]any{
		"id":   inputs.SpaceID,
		"name": inputs.Name,
	}

	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if inputs.Color != nil {
		body["color"] = *inputs.Color
	}
	if inputs.Initials != nil {
		body["initials"] = *inputs.Initials
	}
	if len(inputs.DisabledFeatures) > 0 {
		body["disabledFeatures"] = inputs.DisabledFeatures
	}
	if inputs.ImageUrl != nil {
		body["imageUrl"] = *inputs.ImageUrl
	}

	return body
}
