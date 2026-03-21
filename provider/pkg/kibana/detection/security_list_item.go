package detection

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// SecurityListItem manages an item in a Kibana security value list.
type SecurityListItem struct{}

// SecurityListItemInputs ...
type SecurityListItemInputs struct {
	ListID  string  `pulumi:"listId"`
	Value   string  `pulumi:"value"`
	SpaceID *string `pulumi:"spaceId,optional"`
}

// SecurityListItemState ...
type SecurityListItemState struct {
	SecurityListItemInputs

	// Outputs
	ItemID string `pulumi:"itemId"`
}

var (
	_ infer.CustomDelete[SecurityListItemState]                         = (*SecurityListItem)(nil)
	_ infer.CustomRead[SecurityListItemInputs, SecurityListItemState]   = (*SecurityListItem)(nil)
	_ infer.CustomUpdate[SecurityListItemInputs, SecurityListItemState] = (*SecurityListItem)(nil)
)

// Annotate ...
func (r *SecurityListItem) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an item in a Kibana security value list.")
	a.SetToken("kibana", "SecurityListItem")
}

// Annotate ...
func (i *SecurityListItemInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.ListID, "The parent list ID.")
	a.Describe(&i.Value, "The value of the list item.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
}

// Create ...
func (r *SecurityListItem) Create(
	ctx context.Context,
	req infer.CreateRequest[SecurityListItemInputs],
) (infer.CreateResponse[SecurityListItemState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[SecurityListItemState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := map[string]any{
		"list_id": req.Inputs.ListID,
		"value":   req.Inputs.Value,
	}

	var result map[string]any
	path := clients.SpacePath(spaceID, "/api/lists/items")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[SecurityListItemState]{}, fmt.Errorf("failed to create list item: %w", err)
	}

	id, _ := result["id"].(string)

	return infer.CreateResponse[SecurityListItemState]{
		ID: id,
		Output: SecurityListItemState{
			SecurityListItemInputs: req.Inputs,
			ItemID:                 id,
		},
	}, nil
}

// Read ...
func (r *SecurityListItem) Read(
	ctx context.Context,
	req infer.ReadRequest[SecurityListItemInputs, SecurityListItemState],
) (infer.ReadResponse[SecurityListItemInputs, SecurityListItemState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[SecurityListItemInputs, SecurityListItemState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/lists/items?list_id=%s&value=%s",
		url.QueryEscape(req.State.ListID), url.QueryEscape(req.State.Value)))

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[SecurityListItemInputs, SecurityListItemState]{}, err
	}
	if !exists {
		return infer.ReadResponse[SecurityListItemInputs, SecurityListItemState]{ID: ""}, nil
	}

	return infer.ReadResponse[SecurityListItemInputs, SecurityListItemState](req), nil
}

// Update ...
func (r *SecurityListItem) Update(
	ctx context.Context,
	req infer.UpdateRequest[SecurityListItemInputs, SecurityListItemState],
) (infer.UpdateResponse[SecurityListItemState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[SecurityListItemState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := map[string]any{
		"id":    req.ID,
		"value": req.Inputs.Value,
	}

	path := clients.SpacePath(spaceID, "/api/lists/items")
	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[SecurityListItemState]{}, fmt.Errorf(
			"failed to update list item %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[SecurityListItemState]{
		Output: SecurityListItemState{
			SecurityListItemInputs: req.Inputs,
			ItemID:                 req.ID,
		},
	}, nil
}

// Delete ...
func (r *SecurityListItem) Delete(
	ctx context.Context,
	req infer.DeleteRequest[SecurityListItemState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/lists/items?id=%s", url.QueryEscape(req.ID)))

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}
