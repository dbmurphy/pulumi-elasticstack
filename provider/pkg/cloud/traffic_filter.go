package cloud

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// TrafficFilter manages an Elastic Cloud traffic filter ruleset (IP allowlist, Azure Private Link, etc.).
type TrafficFilter struct{}

// TrafficFilterRule represents an individual rule within a traffic filter ruleset.
type TrafficFilterRule struct {
	Source            *string            `pulumi:"source,optional"`
	Description       *string            `pulumi:"description,optional"`
	AzureEndpointName *string            `pulumi:"azureEndpointName,optional"`
	AzureEndpointGUID *string            `pulumi:"azureEndpointGuid,optional"`
	EgressRule        *TrafficEgressRule `pulumi:"egressRule,optional"`
}

// TrafficEgressRule defines an egress firewall rule.
type TrafficEgressRule struct {
	Target   string `pulumi:"target"`
	Protocol string `pulumi:"protocol"`
	Ports    []int  `pulumi:"ports,optional"`
}

// TrafficFilterInputs defines the input properties for a traffic filter ruleset.
type TrafficFilterInputs struct {
	Name             string              `pulumi:"name"`
	Type             string              `pulumi:"type"`
	Region           string              `pulumi:"region"`
	Description      *string             `pulumi:"description,optional"`
	IncludeByDefault bool                `pulumi:"includeByDefault,optional"`
	Rules            []TrafficFilterRule `pulumi:"rules"`
}

// TrafficFilterState defines the output state for a traffic filter ruleset.
type TrafficFilterState struct {
	TrafficFilterInputs

	// Outputs
	RulesetID string `pulumi:"rulesetId"`
}

var (
	_ infer.CustomDelete[TrafficFilterState]                      = (*TrafficFilter)(nil)
	_ infer.CustomRead[TrafficFilterInputs, TrafficFilterState]   = (*TrafficFilter)(nil)
	_ infer.CustomUpdate[TrafficFilterInputs, TrafficFilterState] = (*TrafficFilter)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *TrafficFilter) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elastic Cloud traffic filter ruleset. "+
		"Supports IP allowlists, Azure Private Link, AWS PrivateLink (VPC Endpoint), "+
		"GCP Private Service Connect, and egress firewall rules.")
	a.SetToken("cloud", "TrafficFilter")
}

// Annotate sets input property descriptions and defaults.
func (i *TrafficFilterInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "Name of the traffic filter ruleset.")
	a.Describe(&i.Type, "Ruleset type: 'ip', 'vpce', 'azure_private_endpoint', "+
		"'gcp_private_service_connect_endpoint', or 'egress_firewall'.")
	a.Describe(
		&i.Region,
		"Cloud region for the ruleset (e.g., 'us-east-1', 'azure-eastus2'). Immutable after creation.",
	)
	a.Describe(&i.Description, "Description of the traffic filter ruleset.")
	a.Describe(&i.IncludeByDefault, "Automatically attach this ruleset to new deployments in the region.")
	a.SetDefault(&i.IncludeByDefault, false)
	a.Describe(&i.Rules, "Traffic filter rules. Fields used depend on type: "+
		"'source' for ip/vpce/gcp, 'azureEndpointName'+'azureEndpointGuid' "+
		"for azure_private_endpoint, 'egressRule' for egress_firewall.")
}

// Create provisions a new traffic filter ruleset.
func (r *TrafficFilter) Create(
	ctx context.Context, req infer.CreateRequest[TrafficFilterInputs],
) (infer.CreateResponse[TrafficFilterState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.CreateResponse[TrafficFilterState]{}, err
	}

	body := buildRulesetBody(req.Inputs)

	var result struct {
		ID string `json:"id"`
	}

	if err := cloudClient.PostJSON(ctx, "/deployments/traffic-filter/rulesets", body, &result); err != nil {
		return infer.CreateResponse[TrafficFilterState]{}, fmt.Errorf("failed to create traffic filter: %w", err)
	}

	return infer.CreateResponse[TrafficFilterState]{
		ID: result.ID,
		Output: TrafficFilterState{
			TrafficFilterInputs: req.Inputs,
			RulesetID:           result.ID,
		},
	}, nil
}

// Read fetches the current state of the traffic filter ruleset.
func (r *TrafficFilter) Read(
	ctx context.Context, req infer.ReadRequest[TrafficFilterInputs, TrafficFilterState],
) (infer.ReadResponse[TrafficFilterInputs, TrafficFilterState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.ReadResponse[TrafficFilterInputs, TrafficFilterState]{}, err
	}

	path := fmt.Sprintf("/deployments/traffic-filter/rulesets/%s", req.ID)
	var result struct {
		ID               string `json:"id"`
		Name             string `json:"name"`
		Type             string `json:"type"`
		Region           string `json:"region"`
		Description      string `json:"description"`
		IncludeByDefault bool   `json:"include_by_default"`
		Rules            []struct {
			ID                string `json:"id"`
			Source            string `json:"source"`
			Description       string `json:"description"`
			AzureEndpointName string `json:"azure_endpoint_name"`
			AzureEndpointGUID string `json:"azure_endpoint_guid"`
			EgressRule        *struct {
				Target   string `json:"target"`
				Protocol string `json:"protocol"`
				Ports    []int  `json:"ports"`
			} `json:"egress_rule"`
		} `json:"rules"`
	}

	if err := cloudClient.GetJSON(ctx, path, &result); err != nil {
		if _, ok := err.(*clients.NotFoundError); ok {
			return infer.ReadResponse[TrafficFilterInputs, TrafficFilterState]{ID: ""}, nil
		}
		return infer.ReadResponse[TrafficFilterInputs, TrafficFilterState]{}, err
	}

	// Reconstruct inputs from API response
	rules := make([]TrafficFilterRule, 0, len(result.Rules))
	for _, r := range result.Rules {
		rule := TrafficFilterRule{}
		if r.Source != "" {
			rule.Source = &r.Source
		}
		if r.Description != "" {
			rule.Description = &r.Description
		}
		if r.AzureEndpointName != "" {
			rule.AzureEndpointName = &r.AzureEndpointName
		}
		if r.AzureEndpointGUID != "" {
			rule.AzureEndpointGUID = &r.AzureEndpointGUID
		}
		if r.EgressRule != nil {
			rule.EgressRule = &TrafficEgressRule{
				Target:   r.EgressRule.Target,
				Protocol: r.EgressRule.Protocol,
				Ports:    r.EgressRule.Ports,
			}
		}
		rules = append(rules, rule)
	}

	var desc *string
	if result.Description != "" {
		desc = &result.Description
	}

	inputs := TrafficFilterInputs{
		Name:             result.Name,
		Type:             result.Type,
		Region:           result.Region,
		Description:      desc,
		IncludeByDefault: result.IncludeByDefault,
		Rules:            rules,
	}

	return infer.ReadResponse[TrafficFilterInputs, TrafficFilterState]{
		ID:     req.ID,
		Inputs: inputs,
		State: TrafficFilterState{
			TrafficFilterInputs: inputs,
			RulesetID:           result.ID,
		},
	}, nil
}

// Update modifies an existing traffic filter ruleset.
func (r *TrafficFilter) Update(
	ctx context.Context, req infer.UpdateRequest[TrafficFilterInputs, TrafficFilterState],
) (infer.UpdateResponse[TrafficFilterState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.UpdateResponse[TrafficFilterState]{}, err
	}

	body := buildRulesetBody(req.Inputs)
	path := fmt.Sprintf("/deployments/traffic-filter/rulesets/%s", req.ID)

	var result struct {
		ID string `json:"id"`
	}

	if err := cloudClient.PutJSON(ctx, path, body, &result); err != nil {
		return infer.UpdateResponse[TrafficFilterState]{}, fmt.Errorf("failed to update traffic filter: %w", err)
	}

	return infer.UpdateResponse[TrafficFilterState]{
		Output: TrafficFilterState{
			TrafficFilterInputs: req.Inputs,
			RulesetID:           req.ID,
		},
	}, nil
}

// Delete removes the traffic filter ruleset.
func (r *TrafficFilter) Delete(
	ctx context.Context, req infer.DeleteRequest[TrafficFilterState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	path := fmt.Sprintf("/deployments/traffic-filter/rulesets/%s?ignore_associations=true", req.ID)
	if err := cloudClient.Delete(ctx, path); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to delete traffic filter: %w", err)
	}

	return infer.DeleteResponse{}, nil
}

func buildRulesetBody(inputs TrafficFilterInputs) map[string]any {
	body := map[string]any{
		"name":               inputs.Name,
		"type":               inputs.Type,
		"region":             inputs.Region,
		"include_by_default": inputs.IncludeByDefault,
	}

	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}

	rules := make([]map[string]any, 0, len(inputs.Rules))
	for _, r := range inputs.Rules {
		rule := map[string]any{}
		if r.Source != nil {
			rule["source"] = *r.Source
		}
		if r.Description != nil {
			rule["description"] = *r.Description
		}
		if r.AzureEndpointName != nil {
			rule["azure_endpoint_name"] = *r.AzureEndpointName
		}
		if r.AzureEndpointGUID != nil {
			rule["azure_endpoint_guid"] = *r.AzureEndpointGUID
		}
		if r.EgressRule != nil {
			egress := map[string]any{
				"target":   r.EgressRule.Target,
				"protocol": r.EgressRule.Protocol,
			}
			if len(r.EgressRule.Ports) > 0 {
				egress["ports"] = r.EgressRule.Ports
			}
			rule["egress_rule"] = egress
		}
		rules = append(rules, rule)
	}
	body["rules"] = rules

	return body
}
