package security

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Role manages an Elasticsearch role via PUT /_security/role/<name>.
type Role struct{}

// RoleInputs defines the input properties for a security role.
type RoleInputs struct {
	Name          string   `pulumi:"name"`
	Cluster       []string `pulumi:"cluster,optional"`
	Indices       *string  `pulumi:"indices,optional"`
	Applications  *string  `pulumi:"applications,optional"`
	RunAs         []string `pulumi:"runAs,optional"`
	Metadata      *string  `pulumi:"metadata,optional"`
	Global        *string  `pulumi:"global,optional"`
	AdoptOnCreate bool     `pulumi:"adoptOnCreate,optional"`
}

// RoleState defines the output state for a security role.
type RoleState struct {
	RoleInputs
}

var (
	_ infer.CustomDelete[RoleState]             = (*Role)(nil)
	_ infer.CustomRead[RoleInputs, RoleState]   = (*Role)(nil)
	_ infer.CustomUpdate[RoleInputs, RoleState] = (*Role)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *Role) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch security role.")
	a.SetToken("elasticsearch", "Role")
}

// Annotate sets input property descriptions and defaults.
func (i *RoleInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The role name.")
	a.Describe(&i.Cluster, "Cluster privileges.")
	a.Describe(&i.Indices, "Index privileges as JSON array.")
	a.Describe(&i.Applications, "Application privileges as JSON array.")
	a.Describe(&i.RunAs, "Users this role can impersonate.")
	a.Describe(&i.Metadata, "Role metadata as JSON.")
	a.Describe(&i.Global, "Global privileges as JSON.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing role into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create provisions a new security role.
func (r *Role) Create(
	ctx context.Context, req infer.CreateRequest[RoleInputs],
) (infer.CreateResponse[RoleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[RoleState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_security/role/"+name)
		if err != nil {
			return infer.CreateResponse[RoleState]{}, err
		}
		if exists {
			body := buildRoleBody(req.Inputs)
			if err := esClient.PutJSON(ctx, "/_security/role/"+name, body, nil); err != nil {
				return infer.CreateResponse[RoleState]{},
					fmt.Errorf("failed to update adopted role %s: %w", name, err)
			}
			return infer.CreateResponse[RoleState]{
				ID:     name,
				Output: RoleState{RoleInputs: req.Inputs},
			}, nil
		}
	}

	body := buildRoleBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_security/role/"+name, body, nil); err != nil {
		return infer.CreateResponse[RoleState]{},
			fmt.Errorf("failed to create role %s: %w", name, err)
	}

	return infer.CreateResponse[RoleState]{
		ID:     name,
		Output: RoleState{RoleInputs: req.Inputs},
	}, nil
}

// Read fetches the current state of the security role.
func (r *Role) Read(
	ctx context.Context,
	req infer.ReadRequest[RoleInputs, RoleState],
) (infer.ReadResponse[RoleInputs, RoleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[RoleInputs, RoleState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_security/role/"+req.ID)
	if err != nil {
		return infer.ReadResponse[RoleInputs, RoleState]{}, err
	}
	if !exists {
		return infer.ReadResponse[RoleInputs, RoleState]{ID: ""}, nil
	}

	return infer.ReadResponse[RoleInputs, RoleState](req), nil
}

// Update modifies an existing security role.
func (r *Role) Update(
	ctx context.Context,
	req infer.UpdateRequest[RoleInputs, RoleState],
) (infer.UpdateResponse[RoleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[RoleState]{}, err
	}

	body := buildRoleBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_security/role/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[RoleState]{},
			fmt.Errorf("failed to update role %s: %w", req.Inputs.Name, err)
	}

	return infer.UpdateResponse[RoleState]{
		Output: RoleState{RoleInputs: req.Inputs},
	}, nil
}

// Delete removes the security role.
func (r *Role) Delete(
	ctx context.Context, req infer.DeleteRequest[RoleState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_security/role/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildRoleBody(inputs RoleInputs) map[string]any {
	body := map[string]any{}

	if len(inputs.Cluster) > 0 {
		body["cluster"] = inputs.Cluster
	}
	if inputs.Indices != nil {
		var indices any
		_ = json.Unmarshal([]byte(*inputs.Indices), &indices)
		body["indices"] = indices
	}
	if inputs.Applications != nil {
		var apps any
		_ = json.Unmarshal([]byte(*inputs.Applications), &apps)
		body["applications"] = apps
	}
	if len(inputs.RunAs) > 0 {
		body["run_as"] = inputs.RunAs
	}
	if inputs.Metadata != nil {
		var meta any
		_ = json.Unmarshal([]byte(*inputs.Metadata), &meta)
		body["metadata"] = meta
	}
	if inputs.Global != nil {
		var global any
		_ = json.Unmarshal([]byte(*inputs.Global), &global)
		body["global"] = global
	}

	return body
}
