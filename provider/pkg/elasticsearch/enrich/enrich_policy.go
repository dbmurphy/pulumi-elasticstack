// Package enrich implements Elasticsearch enrich policy management.
package enrich

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Policy manages an Elasticsearch enrich policy via PUT /_enrich/policy/<name>.
type Policy struct{}

// PolicyInputs defines the input properties for an enrich policy.
type PolicyInputs struct {
	Name             string   `pulumi:"name"`
	PolicyType       string   `pulumi:"policyType"`
	Indices          []string `pulumi:"indices"`
	MatchField       string   `pulumi:"matchField"`
	EnrichFields     []string `pulumi:"enrichFields"`
	Query            *string  `pulumi:"query,optional"`
	Execute          *bool    `pulumi:"execute,optional"`
	ExecuteOnCreate  *bool    `pulumi:"executeOnCreate,optional"`
	ExecuteOnUpdate  *bool    `pulumi:"executeOnUpdate,optional"`
	ExecutionTimeout *string  `pulumi:"executionTimeout,optional"`
	AdoptOnCreate    bool     `pulumi:"adoptOnCreate,optional"`
}

// PolicyState defines the output state for an enrich policy.
type PolicyState struct {
	PolicyInputs
}

var (
	_ infer.CustomDelete[PolicyState]               = (*Policy)(nil)
	_ infer.CustomRead[PolicyInputs, PolicyState]   = (*Policy)(nil)
	_ infer.CustomUpdate[PolicyInputs, PolicyState] = (*Policy)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *Policy) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch enrich policy.")
	a.SetToken("elasticsearch", "EnrichPolicy")
}

// Annotate sets input property descriptions and defaults.
func (i *PolicyInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The enrich policy name.")
	a.Describe(&i.PolicyType, "The policy type: 'match', 'range', or 'geo_match'.")
	a.Describe(&i.Indices, "Source indices for the enrich policy.")
	a.Describe(&i.MatchField, "The field to match in the source indices.")
	a.Describe(&i.EnrichFields, "Fields to add from matching source documents.")
	a.Describe(&i.Query, "A query to filter source documents as JSON.")
	a.Describe(&i.Execute, "Whether to execute the policy. Defaults to false.")
	a.SetDefault(&i.Execute, false)
	a.Describe(&i.ExecuteOnCreate, "Whether to execute the policy on creation. Defaults to false.")
	a.SetDefault(&i.ExecuteOnCreate, false)
	a.Describe(&i.ExecuteOnUpdate, "Whether to execute the policy on update. Defaults to false.")
	a.SetDefault(&i.ExecuteOnUpdate, false)
	a.Describe(&i.ExecutionTimeout, "Timeout for policy execution (e.g. '5m').")
	a.Describe(&i.AdoptOnCreate, "Adopt existing enrich policy into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create provisions a new enrich policy.
func (r *Policy) Create(
	ctx context.Context, req infer.CreateRequest[PolicyInputs],
) (infer.CreateResponse[PolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[PolicyState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_enrich/policy/"+name)
		if err != nil {
			return infer.CreateResponse[PolicyState]{}, err
		}
		if exists {
			if shouldExecute(req.Inputs.ExecuteOnCreate, req.Inputs.Execute) {
				if err := executePolicy(ctx, esClient, name, req.Inputs.ExecutionTimeout); err != nil {
					return infer.CreateResponse[PolicyState]{}, err
				}
			}
			return infer.CreateResponse[PolicyState]{
				ID:     name,
				Output: PolicyState{PolicyInputs: req.Inputs},
			}, nil
		}
	}

	body := buildEnrichPolicyBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_enrich/policy/"+name, body, nil); err != nil {
		return infer.CreateResponse[PolicyState]{}, fmt.Errorf("failed to create enrich policy %s: %w", name, err)
	}

	if shouldExecute(req.Inputs.ExecuteOnCreate, req.Inputs.Execute) {
		if err := executePolicy(ctx, esClient, name, req.Inputs.ExecutionTimeout); err != nil {
			return infer.CreateResponse[PolicyState]{}, err
		}
	}

	return infer.CreateResponse[PolicyState]{
		ID:     name,
		Output: PolicyState{PolicyInputs: req.Inputs},
	}, nil
}

// Read fetches the current state of the enrich policy.
func (r *Policy) Read(
	ctx context.Context, req infer.ReadRequest[PolicyInputs, PolicyState],
) (infer.ReadResponse[PolicyInputs, PolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[PolicyInputs, PolicyState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_enrich/policy/"+req.ID)
	if err != nil {
		return infer.ReadResponse[PolicyInputs, PolicyState]{}, err
	}
	if !exists {
		return infer.ReadResponse[PolicyInputs, PolicyState]{ID: ""}, nil
	}

	return infer.ReadResponse[PolicyInputs, PolicyState](req), nil
}

// Update modifies an existing enrich policy.
func (r *Policy) Update(
	ctx context.Context, req infer.UpdateRequest[PolicyInputs, PolicyState],
) (infer.UpdateResponse[PolicyState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[PolicyState]{}, err
	}

	name := req.Inputs.Name

	// Enrich policies are immutable in ES — delete and recreate on update.
	if err := esClient.Delete(ctx, "/_enrich/policy/"+name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.UpdateResponse[PolicyState]{},
				fmt.Errorf("failed to delete enrich policy %s for update: %w", name, err)
		}
	}

	body := buildEnrichPolicyBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_enrich/policy/"+name, body, nil); err != nil {
		return infer.UpdateResponse[PolicyState]{}, fmt.Errorf("failed to recreate enrich policy %s: %w", name, err)
	}

	if shouldExecute(req.Inputs.ExecuteOnUpdate, req.Inputs.Execute) {
		if err := executePolicy(ctx, esClient, name, req.Inputs.ExecutionTimeout); err != nil {
			return infer.UpdateResponse[PolicyState]{}, err
		}
	}

	return infer.UpdateResponse[PolicyState]{
		Output: PolicyState{PolicyInputs: req.Inputs},
	}, nil
}

// Delete removes the enrich policy.
func (r *Policy) Delete(
	ctx context.Context, req infer.DeleteRequest[PolicyState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_enrich/policy/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func shouldExecute(specific *bool, general *bool) bool {
	if specific != nil && *specific {
		return true
	}
	if general != nil && *general {
		return true
	}
	return false
}

func executePolicy(ctx context.Context, esClient *clients.ElasticsearchClient, name string, timeout *string) error {
	path := "/_enrich/policy/" + name + "/_execute"
	if timeout != nil {
		path += "?wait_for_completion=true&timeout=" + *timeout
	}
	if err := esClient.PostJSON(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("failed to execute enrich policy %s: %w", name, err)
	}
	return nil
}

func buildEnrichPolicyBody(inputs PolicyInputs) map[string]any {
	policy := map[string]any{
		"indices":       inputs.Indices,
		"match_field":   inputs.MatchField,
		"enrich_fields": inputs.EnrichFields,
	}

	if inputs.Query != nil {
		var query any
		if err := json.Unmarshal([]byte(*inputs.Query), &query); err == nil {
			policy["query"] = query
		}
	}

	body := map[string]any{
		inputs.PolicyType: policy,
	}

	return body
}
