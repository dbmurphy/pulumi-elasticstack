package detection

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// SecurityExceptionList manages a Kibana security exception list.
type SecurityExceptionList struct{}

// SecurityExceptionListInputs ...
type SecurityExceptionListInputs struct {
	Name          string   `pulumi:"name"`
	Description   string   `pulumi:"description"`
	ListID        *string  `pulumi:"listId,optional"`
	ListType      string   `pulumi:"listType"`
	NamespaceType *string  `pulumi:"namespaceType,optional"`
	Tags          []string `pulumi:"tags,optional"`
	SpaceID       *string  `pulumi:"spaceId,optional"`
	AdoptOnCreate bool     `pulumi:"adoptOnCreate,optional"`
}

// SecurityExceptionListState ...
type SecurityExceptionListState struct {
	SecurityExceptionListInputs
}

var (
	_ infer.CustomDelete[SecurityExceptionListState]                              = (*SecurityExceptionList)(nil)
	_ infer.CustomRead[SecurityExceptionListInputs, SecurityExceptionListState]   = (*SecurityExceptionList)(nil)
	_ infer.CustomUpdate[SecurityExceptionListInputs, SecurityExceptionListState] = (*SecurityExceptionList)(nil)
)

// Annotate ...
func (r *SecurityExceptionList) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana security exception list.")
	a.SetToken("kibana", "SecurityExceptionList")
}

// Annotate ...
func (i *SecurityExceptionListInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the exception list.")
	a.Describe(&i.Description, "A description of the exception list.")
	a.Describe(&i.ListID, "The list ID. Auto-generated if not provided.")
	a.Describe(&i.ListType, "The list type: detection, endpoint, or rule_default.")
	a.Describe(&i.NamespaceType, "The namespace type: single or agnostic. Defaults to 'single'.")
	a.Describe(&i.Tags, "Tags for the exception list.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing exception list into Pulumi state on create.")
	a.SetDefault(&i.NamespaceType, "single")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *SecurityExceptionList) Create(
	ctx context.Context,
	req infer.CreateRequest[SecurityExceptionListInputs],
) (infer.CreateResponse[SecurityExceptionListState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[SecurityExceptionListState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	ns := resolveNamespaceType(req.Inputs.NamespaceType)

	if req.Inputs.AdoptOnCreate && req.Inputs.ListID != nil {
		readPath := clients.SpacePath(spaceID, fmt.Sprintf("/api/exception_lists?list_id=%s&namespace_type=%s",
			url.QueryEscape(*req.Inputs.ListID), url.QueryEscape(ns)))
		exists, err := kbClient.Exists(ctx, readPath)
		if err != nil {
			return infer.CreateResponse[SecurityExceptionListState]{}, err
		}
		if exists {
			body := buildExceptionListBody(req.Inputs)
			body["list_id"] = *req.Inputs.ListID
			path := clients.SpacePath(spaceID, "/api/exception_lists")
			if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
				return infer.CreateResponse[SecurityExceptionListState]{}, fmt.Errorf(
					"failed to update adopted exception list: %w",
					err,
				)
			}
			return infer.CreateResponse[SecurityExceptionListState]{
				ID:     *req.Inputs.ListID,
				Output: SecurityExceptionListState{SecurityExceptionListInputs: req.Inputs},
			}, nil
		}
	}

	body := buildExceptionListBody(req.Inputs)
	if req.Inputs.ListID != nil {
		body["list_id"] = *req.Inputs.ListID
	}

	var result map[string]any
	path := clients.SpacePath(spaceID, "/api/exception_lists")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[SecurityExceptionListState]{}, fmt.Errorf(
			"failed to create exception list %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	listID, _ := result["list_id"].(string)

	// Backfill ListID into inputs if it was auto-generated.
	inputs := req.Inputs
	if inputs.ListID == nil {
		inputs.ListID = &listID
	}

	return infer.CreateResponse[SecurityExceptionListState]{
		ID:     listID,
		Output: SecurityExceptionListState{SecurityExceptionListInputs: inputs},
	}, nil
}

// Read ...
func (r *SecurityExceptionList) Read(
	ctx context.Context,
	req infer.ReadRequest[SecurityExceptionListInputs, SecurityExceptionListState],
) (infer.ReadResponse[SecurityExceptionListInputs, SecurityExceptionListState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[SecurityExceptionListInputs, SecurityExceptionListState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	ns := resolveNamespaceType(req.State.NamespaceType)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/exception_lists?list_id=%s&namespace_type=%s",
		url.QueryEscape(req.ID), url.QueryEscape(ns)))

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[SecurityExceptionListInputs, SecurityExceptionListState]{}, err
	}
	if !exists {
		return infer.ReadResponse[SecurityExceptionListInputs, SecurityExceptionListState]{ID: ""}, nil
	}

	return infer.ReadResponse[SecurityExceptionListInputs, SecurityExceptionListState](req), nil
}

// Update ...
func (r *SecurityExceptionList) Update(
	ctx context.Context,
	req infer.UpdateRequest[SecurityExceptionListInputs, SecurityExceptionListState],
) (infer.UpdateResponse[SecurityExceptionListState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[SecurityExceptionListState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body := buildExceptionListBody(req.Inputs)
	body["list_id"] = req.ID

	path := clients.SpacePath(spaceID, "/api/exception_lists")
	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[SecurityExceptionListState]{}, fmt.Errorf(
			"failed to update exception list %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[SecurityExceptionListState]{
		Output: SecurityExceptionListState{SecurityExceptionListInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *SecurityExceptionList) Delete(
	ctx context.Context,
	req infer.DeleteRequest[SecurityExceptionListState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	ns := resolveNamespaceType(req.State.NamespaceType)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/exception_lists?list_id=%s&namespace_type=%s",
		url.QueryEscape(req.ID), url.QueryEscape(ns)))

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildExceptionListBody(inputs SecurityExceptionListInputs) map[string]any {
	body := map[string]any{
		"name":        inputs.Name,
		"description": inputs.Description,
		"type":        inputs.ListType,
	}

	ns := resolveNamespaceType(inputs.NamespaceType)
	body["namespace_type"] = ns

	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}

	return body
}

func resolveNamespaceType(ns *string) string {
	if ns == nil || *ns == "" {
		return "single"
	}
	return *ns
}
