package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// SecurityExceptionItem manages an item in a Kibana security exception list.
type SecurityExceptionItem struct{}

// SecurityExceptionItemInputs ...
type SecurityExceptionItemInputs struct {
	ListID        string   `pulumi:"listId"`
	Name          string   `pulumi:"name"`
	Description   string   `pulumi:"description"`
	ItemType      string   `pulumi:"itemType"`
	NamespaceType *string  `pulumi:"namespaceType,optional"`
	Entries       string   `pulumi:"entries"`
	ItemID        *string  `pulumi:"itemId,optional"`
	Tags          []string `pulumi:"tags,optional"`
	ExpireTime    *string  `pulumi:"expireTime,optional"`
	OsTypes       []string `pulumi:"osTypes,optional"`
	Comments      *string  `pulumi:"comments,optional"`
	SpaceID       *string  `pulumi:"spaceId,optional"`
}

// SecurityExceptionItemState ...
type SecurityExceptionItemState struct {
	SecurityExceptionItemInputs
}

var (
	_ infer.CustomDelete[SecurityExceptionItemState]                              = (*SecurityExceptionItem)(nil)
	_ infer.CustomRead[SecurityExceptionItemInputs, SecurityExceptionItemState]   = (*SecurityExceptionItem)(nil)
	_ infer.CustomUpdate[SecurityExceptionItemInputs, SecurityExceptionItemState] = (*SecurityExceptionItem)(nil)
)

// Annotate ...
func (r *SecurityExceptionItem) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an item in a Kibana security exception list.")
	a.SetToken("kibana", "SecurityExceptionItem")
}

// Annotate ...
func (i *SecurityExceptionItemInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.ListID, "The parent exception list ID.")
	a.Describe(&i.Name, "The name of the exception item.")
	a.Describe(&i.Description, "A description of the exception item.")
	a.Describe(&i.ItemType, "The exception item type, e.g. 'simple'.")
	a.Describe(&i.NamespaceType, "The namespace type: single or agnostic. Defaults to 'single'.")
	a.Describe(&i.Entries, "Match conditions as a JSON array string.")
	a.Describe(&i.ItemID, "The item ID. Auto-generated if not provided.")
	a.Describe(&i.Tags, "Tags for the exception item.")
	a.Describe(&i.ExpireTime, "Expiration time in ISO 8601 format.")
	a.Describe(&i.OsTypes, "OS types this exception applies to.")
	a.Describe(&i.Comments, "Comments as a JSON array string.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.SetDefault(&i.NamespaceType, "single")
}

// Create ...
func (r *SecurityExceptionItem) Create(
	ctx context.Context,
	req infer.CreateRequest[SecurityExceptionItemInputs],
) (infer.CreateResponse[SecurityExceptionItemState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[SecurityExceptionItemState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildExceptionItemBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[SecurityExceptionItemState]{}, err
	}

	var result map[string]any
	path := clients.SpacePath(spaceID, "/api/exception_lists/items")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[SecurityExceptionItemState]{}, fmt.Errorf(
			"failed to create exception item %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	itemID, _ := result["item_id"].(string)

	inputs := req.Inputs
	if inputs.ItemID == nil {
		inputs.ItemID = &itemID
	}

	return infer.CreateResponse[SecurityExceptionItemState]{
		ID:     itemID,
		Output: SecurityExceptionItemState{SecurityExceptionItemInputs: inputs},
	}, nil
}

// Read ...
func (r *SecurityExceptionItem) Read(
	ctx context.Context,
	req infer.ReadRequest[SecurityExceptionItemInputs, SecurityExceptionItemState],
) (infer.ReadResponse[SecurityExceptionItemInputs, SecurityExceptionItemState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[SecurityExceptionItemInputs, SecurityExceptionItemState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	ns := resolveNamespaceType(req.State.NamespaceType)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/exception_lists/items?item_id=%s&namespace_type=%s",
		url.QueryEscape(req.ID), url.QueryEscape(ns)))

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[SecurityExceptionItemInputs, SecurityExceptionItemState]{}, err
	}
	if !exists {
		return infer.ReadResponse[SecurityExceptionItemInputs, SecurityExceptionItemState]{ID: ""}, nil
	}

	return infer.ReadResponse[SecurityExceptionItemInputs, SecurityExceptionItemState](req), nil
}

// Update ...
func (r *SecurityExceptionItem) Update(
	ctx context.Context,
	req infer.UpdateRequest[SecurityExceptionItemInputs, SecurityExceptionItemState],
) (infer.UpdateResponse[SecurityExceptionItemState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[SecurityExceptionItemState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildExceptionItemBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[SecurityExceptionItemState]{}, err
	}
	body["item_id"] = req.ID

	path := clients.SpacePath(spaceID, "/api/exception_lists/items")
	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[SecurityExceptionItemState]{}, fmt.Errorf(
			"failed to update exception item %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[SecurityExceptionItemState]{
		Output: SecurityExceptionItemState{SecurityExceptionItemInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *SecurityExceptionItem) Delete(
	ctx context.Context,
	req infer.DeleteRequest[SecurityExceptionItemState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	ns := resolveNamespaceType(req.State.NamespaceType)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/exception_lists/items?item_id=%s&namespace_type=%s",
		url.QueryEscape(req.ID), url.QueryEscape(ns)))

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildExceptionItemBody(inputs SecurityExceptionItemInputs) (map[string]any, error) {
	body := map[string]any{
		"list_id":     inputs.ListID,
		"name":        inputs.Name,
		"description": inputs.Description,
		"type":        inputs.ItemType,
	}

	ns := resolveNamespaceType(inputs.NamespaceType)
	body["namespace_type"] = ns

	var entries any
	if err := json.Unmarshal([]byte(inputs.Entries), &entries); err != nil {
		return nil, fmt.Errorf("invalid entries JSON: %w", err)
	}
	body["entries"] = entries

	if inputs.ItemID != nil {
		body["item_id"] = *inputs.ItemID
	}
	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}
	if inputs.ExpireTime != nil {
		body["expire_time"] = *inputs.ExpireTime
	}
	if len(inputs.OsTypes) > 0 {
		body["os_types"] = inputs.OsTypes
	}
	if inputs.Comments != nil {
		var comments any
		if err := json.Unmarshal([]byte(*inputs.Comments), &comments); err != nil {
			return nil, fmt.Errorf("invalid comments JSON: %w", err)
		}
		body["comments"] = comments
	}

	return body, nil
}
