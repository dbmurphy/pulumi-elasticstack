// Package security implements Kibana security role management.
package security

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// KibanaSecurityRole manages a Kibana security role via PUT /api/security/role/{name}.
type KibanaSecurityRole struct{}

// KibanaSecurityRoleInputs defines the input properties for a Kibana security role.
type KibanaSecurityRoleInputs struct {
	Name          string  `pulumi:"name"`
	Elasticsearch *string `pulumi:"elasticsearch,optional"`
	Kibana        *string `pulumi:"kibana,optional"`
	Metadata      *string `pulumi:"metadata,optional"`
	AdoptOnCreate bool    `pulumi:"adoptOnCreate,optional"`
}

// KibanaSecurityRoleState defines the output state for a Kibana security role.
type KibanaSecurityRoleState struct {
	KibanaSecurityRoleInputs
}

var (
	_ infer.CustomDelete[KibanaSecurityRoleState]                           = (*KibanaSecurityRole)(nil)
	_ infer.CustomRead[KibanaSecurityRoleInputs, KibanaSecurityRoleState]   = (*KibanaSecurityRole)(nil)
	_ infer.CustomUpdate[KibanaSecurityRoleInputs, KibanaSecurityRoleState] = (*KibanaSecurityRole)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *KibanaSecurityRole) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana security role.")
	a.SetToken("kibana", "SecurityRole")
}

// Annotate sets input property descriptions and defaults.
func (i *KibanaSecurityRoleInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The role name.")
	a.Describe(&i.Elasticsearch, "Elasticsearch privileges as JSON object.")
	a.Describe(&i.Kibana, "Kibana feature privileges as JSON array of objects.")
	a.Describe(&i.Metadata, "Role metadata as JSON object.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing role into state instead of failing.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create provisions a new Kibana security role.
func (r *KibanaSecurityRole) Create(
	ctx context.Context,
	req infer.CreateRequest[KibanaSecurityRoleInputs],
) (infer.CreateResponse[KibanaSecurityRoleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[KibanaSecurityRoleState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := kbClient.Exists(ctx, "/api/security/role/"+name)
		if err != nil {
			return infer.CreateResponse[KibanaSecurityRoleState]{}, err
		}
		if exists {
			body := buildKibanaRoleBody(req.Inputs)
			if err := kbClient.PutJSON(ctx, "/api/security/role/"+name, body, nil); err != nil {
				return infer.CreateResponse[KibanaSecurityRoleState]{}, fmt.Errorf(
					"failed to update adopted role %s: %w",
					name,
					err,
				)
			}
			return infer.CreateResponse[KibanaSecurityRoleState]{
				ID:     name,
				Output: KibanaSecurityRoleState{KibanaSecurityRoleInputs: req.Inputs},
			}, nil
		}
	}

	body := buildKibanaRoleBody(req.Inputs)
	if err := kbClient.PutJSON(ctx, "/api/security/role/"+name, body, nil); err != nil {
		return infer.CreateResponse[KibanaSecurityRoleState]{}, fmt.Errorf("failed to create role %s: %w", name, err)
	}

	return infer.CreateResponse[KibanaSecurityRoleState]{
		ID:     name,
		Output: KibanaSecurityRoleState{KibanaSecurityRoleInputs: req.Inputs},
	}, nil
}

// Read fetches the current state of the Kibana security role.
func (r *KibanaSecurityRole) Read(
	ctx context.Context,
	req infer.ReadRequest[KibanaSecurityRoleInputs, KibanaSecurityRoleState],
) (infer.ReadResponse[KibanaSecurityRoleInputs, KibanaSecurityRoleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[KibanaSecurityRoleInputs, KibanaSecurityRoleState]{}, err
	}

	exists, err := kbClient.Exists(ctx, "/api/security/role/"+req.ID)
	if err != nil {
		return infer.ReadResponse[KibanaSecurityRoleInputs, KibanaSecurityRoleState]{}, err
	}
	if !exists {
		return infer.ReadResponse[KibanaSecurityRoleInputs, KibanaSecurityRoleState]{ID: ""}, nil
	}

	return infer.ReadResponse[KibanaSecurityRoleInputs, KibanaSecurityRoleState](req), nil
}

// Update modifies an existing Kibana security role.
func (r *KibanaSecurityRole) Update(
	ctx context.Context,
	req infer.UpdateRequest[KibanaSecurityRoleInputs, KibanaSecurityRoleState],
) (infer.UpdateResponse[KibanaSecurityRoleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[KibanaSecurityRoleState]{}, err
	}

	body := buildKibanaRoleBody(req.Inputs)
	if err := kbClient.PutJSON(ctx, "/api/security/role/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[KibanaSecurityRoleState]{}, fmt.Errorf(
			"failed to update role %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	return infer.UpdateResponse[KibanaSecurityRoleState]{
		Output: KibanaSecurityRoleState{KibanaSecurityRoleInputs: req.Inputs},
	}, nil
}

// Delete removes the Kibana security role.
func (r *KibanaSecurityRole) Delete(
	ctx context.Context,
	req infer.DeleteRequest[KibanaSecurityRoleState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := kbClient.Delete(ctx, "/api/security/role/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildKibanaRoleBody(inputs KibanaSecurityRoleInputs) map[string]any {
	body := map[string]any{}

	if inputs.Elasticsearch != nil {
		var es any
		if err := json.Unmarshal([]byte(*inputs.Elasticsearch), &es); err == nil {
			body["elasticsearch"] = es
		}
	}
	if inputs.Kibana != nil {
		var kb any
		if err := json.Unmarshal([]byte(*inputs.Kibana), &kb); err == nil {
			body["kibana"] = kb
		}
	}
	if inputs.Metadata != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.Metadata), &meta); err == nil {
			body["metadata"] = meta
		}
	}

	return body
}
