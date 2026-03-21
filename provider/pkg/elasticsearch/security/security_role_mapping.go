package security

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// RoleMapping manages a role mapping via PUT /_security/role_mapping/<name>.
type RoleMapping struct{}

// RoleMappingInputs ...
type RoleMappingInputs struct {
	Name          string   `pulumi:"name"`
	Enabled       *bool    `pulumi:"enabled,optional"`
	Roles         []string `pulumi:"roles,optional"`
	RoleTemplates *string  `pulumi:"roleTemplates,optional"`
	Rules         string   `pulumi:"rules"`
	Metadata      *string  `pulumi:"metadata,optional"`
	AdoptOnCreate bool     `pulumi:"adoptOnCreate,optional"`
}

// RoleMappingState ...
type RoleMappingState struct {
	RoleMappingInputs
}

var (
	_ infer.CustomDelete[RoleMappingState]                    = (*RoleMapping)(nil)
	_ infer.CustomRead[RoleMappingInputs, RoleMappingState]   = (*RoleMapping)(nil)
	_ infer.CustomUpdate[RoleMappingInputs, RoleMappingState] = (*RoleMapping)(nil)
)

// Annotate ...
func (r *RoleMapping) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch security role mapping.")
	a.SetToken("elasticsearch", "RoleMapping")
}

// Annotate ...
func (i *RoleMappingInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The role mapping name.")
	a.Describe(&i.Enabled, "Whether the role mapping is enabled.")
	a.SetDefault(&i.Enabled, true)
	a.Describe(&i.Roles, "Roles to assign when mapping matches.")
	a.Describe(&i.RoleTemplates, "Role templates as JSON.")
	a.Describe(&i.Rules, "Mapping rules as JSON.")
	a.Describe(&i.Metadata, "Role mapping metadata as JSON.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing role mapping into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *RoleMapping) Create(
	ctx context.Context,
	req infer.CreateRequest[RoleMappingInputs],
) (infer.CreateResponse[RoleMappingState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[RoleMappingState]{}, err
	}

	name := req.Inputs.Name
	body, err := buildRoleMappingBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[RoleMappingState]{}, err
	}

	if err := esClient.PutJSON(ctx, "/_security/role_mapping/"+name, body, nil); err != nil {
		return infer.CreateResponse[RoleMappingState]{}, fmt.Errorf(
			"failed to create role mapping %s: %w",
			name,
			err,
		)
	}

	return infer.CreateResponse[RoleMappingState]{
		ID:     name,
		Output: RoleMappingState{RoleMappingInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *RoleMapping) Read(
	ctx context.Context,
	req infer.ReadRequest[RoleMappingInputs, RoleMappingState],
) (infer.ReadResponse[RoleMappingInputs, RoleMappingState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[RoleMappingInputs, RoleMappingState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_security/role_mapping/"+req.ID)
	if err != nil {
		return infer.ReadResponse[RoleMappingInputs, RoleMappingState]{}, err
	}
	if !exists {
		return infer.ReadResponse[RoleMappingInputs, RoleMappingState]{ID: ""}, nil
	}

	return infer.ReadResponse[RoleMappingInputs, RoleMappingState](req), nil
}

// Update ...
func (r *RoleMapping) Update(
	ctx context.Context,
	req infer.UpdateRequest[RoleMappingInputs, RoleMappingState],
) (infer.UpdateResponse[RoleMappingState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[RoleMappingState]{}, err
	}

	body, err := buildRoleMappingBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[RoleMappingState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_security/role_mapping/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[RoleMappingState]{}, fmt.Errorf(
			"failed to update role mapping %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	return infer.UpdateResponse[RoleMappingState]{
		Output: RoleMappingState{RoleMappingInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *RoleMapping) Delete(
	ctx context.Context,
	req infer.DeleteRequest[RoleMappingState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_security/role_mapping/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildRoleMappingBody(inputs RoleMappingInputs) (map[string]any, error) {
	body := map[string]any{}

	if inputs.Enabled != nil {
		body["enabled"] = *inputs.Enabled
	}
	if len(inputs.Roles) > 0 {
		body["roles"] = inputs.Roles
	}
	if inputs.RoleTemplates != nil {
		var tmpl any
		if err := json.Unmarshal([]byte(*inputs.RoleTemplates), &tmpl); err != nil {
			return nil, fmt.Errorf("invalid tmpl JSON: %w", err)
		}
		body["role_templates"] = tmpl
	}

	var rules any
	if err := json.Unmarshal([]byte(inputs.Rules), &rules); err != nil {
		return nil, fmt.Errorf("invalid rules JSON: %w", err)
	}
	body["rules"] = rules

	if inputs.Metadata != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.Metadata), &meta); err != nil {
			return nil, fmt.Errorf("invalid meta JSON: %w", err)
		}
		body["metadata"] = meta
	}

	return body, nil
}
