package detection

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// SecurityList manages a Kibana security list (value list).
type SecurityList struct{}

// SecurityListInputs ...
type SecurityListInputs struct {
	Name          string  `pulumi:"name"`
	Description   string  `pulumi:"description"`
	ListType      string  `pulumi:"listType"`
	SpaceID       *string `pulumi:"spaceId,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// SecurityListState ...
type SecurityListState struct {
	SecurityListInputs

	// Outputs
	ListID string `pulumi:"listId"`
}

var (
	_ infer.CustomDelete[SecurityListState]                     = (*SecurityList)(nil)
	_ infer.CustomRead[SecurityListInputs, SecurityListState]   = (*SecurityList)(nil)
	_ infer.CustomUpdate[SecurityListInputs, SecurityListState] = (*SecurityList)(nil)
)

// Annotate ...
func (r *SecurityList) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana security value list.")
	a.SetToken("kibana", "SecurityList")
}

// Annotate ...
func (i *SecurityListInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the list.")
	a.Describe(&i.Description, "A description of the list.")
	a.Describe(&i.ListType, "The list type: keyword, ip, or ip_range.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing list into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *SecurityList) Create(
	ctx context.Context,
	req infer.CreateRequest[SecurityListInputs],
) (infer.CreateResponse[SecurityListState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[SecurityListState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := buildSecurityListBody(req.Inputs)

	var result map[string]any
	path := clients.SpacePath(spaceID, "/api/lists")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[SecurityListState]{}, fmt.Errorf(
			"failed to create security list %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	id, _ := result["id"].(string)

	return infer.CreateResponse[SecurityListState]{
		ID: id,
		Output: SecurityListState{
			SecurityListInputs: req.Inputs,
			ListID:             id,
		},
	}, nil
}

// Read ...
func (r *SecurityList) Read(
	ctx context.Context,
	req infer.ReadRequest[SecurityListInputs, SecurityListState],
) (infer.ReadResponse[SecurityListInputs, SecurityListState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[SecurityListInputs, SecurityListState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/lists?id=%s", url.QueryEscape(req.ID)))

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[SecurityListInputs, SecurityListState]{}, err
	}
	if !exists {
		return infer.ReadResponse[SecurityListInputs, SecurityListState]{ID: ""}, nil
	}

	return infer.ReadResponse[SecurityListInputs, SecurityListState](req), nil
}

// Update ...
func (r *SecurityList) Update(
	ctx context.Context,
	req infer.UpdateRequest[SecurityListInputs, SecurityListState],
) (infer.UpdateResponse[SecurityListState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[SecurityListState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := buildSecurityListBody(req.Inputs)
	body["id"] = req.ID

	path := clients.SpacePath(spaceID, "/api/lists")
	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[SecurityListState]{}, fmt.Errorf(
			"failed to update security list %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[SecurityListState]{
		Output: SecurityListState{
			SecurityListInputs: req.Inputs,
			ListID:             req.ID,
		},
	}, nil
}

// Delete ...
func (r *SecurityList) Delete(
	ctx context.Context,
	req infer.DeleteRequest[SecurityListState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/lists?id=%s", url.QueryEscape(req.ID)))

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildSecurityListBody(inputs SecurityListInputs) map[string]any {
	return map[string]any{
		"name":        inputs.Name,
		"description": inputs.Description,
		"type":        inputs.ListType,
	}
}
