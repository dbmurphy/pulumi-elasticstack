package detection

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// SecurityDetectionRule manages a Kibana security detection rule.
type SecurityDetectionRule struct{}

// SecurityDetectionRuleInputs ...
type SecurityDetectionRuleInputs struct {
	Name           string   `pulumi:"name"`
	Description    string   `pulumi:"description"`
	RiskScore      int      `pulumi:"riskScore"`
	Severity       string   `pulumi:"severity"`
	RuleType       string   `pulumi:"ruleType"`
	Query          *string  `pulumi:"query,optional"`
	Language       *string  `pulumi:"language,optional"`
	IndexPatterns  []string `pulumi:"indexPatterns,optional"`
	Filters        *string  `pulumi:"filters,optional"`
	Enabled        *bool    `pulumi:"enabled,optional"`
	Interval       *string  `pulumi:"interval,optional"`
	FromTime       *string  `pulumi:"fromTime,optional"`
	ToTime         *string  `pulumi:"toTime,optional"`
	Tags           []string `pulumi:"tags,optional"`
	Actions        *string  `pulumi:"actions,optional"`
	ExceptionsList *string  `pulumi:"exceptionsList,optional"`
	SpaceID        *string  `pulumi:"spaceId,optional"`
	AdoptOnCreate  bool     `pulumi:"adoptOnCreate,optional"`
}

// SecurityDetectionRuleState ...
type SecurityDetectionRuleState struct {
	SecurityDetectionRuleInputs

	// Outputs
	RuleID string `pulumi:"ruleId"`
}

var (
	_ infer.CustomDelete[SecurityDetectionRuleState]                              = (*SecurityDetectionRule)(nil)
	_ infer.CustomRead[SecurityDetectionRuleInputs, SecurityDetectionRuleState]   = (*SecurityDetectionRule)(nil)
	_ infer.CustomUpdate[SecurityDetectionRuleInputs, SecurityDetectionRuleState] = (*SecurityDetectionRule)(nil)
)

// Annotate ...
func (r *SecurityDetectionRule) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana security detection rule.")
	a.SetToken("kibana", "SecurityDetectionRule")
}

// Annotate ...
func (i *SecurityDetectionRuleInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the detection rule.")
	a.Describe(&i.Description, "A description of the detection rule.")
	a.Describe(&i.RiskScore, "The risk score (0-100).")
	a.Describe(&i.Severity, "The severity level: low, medium, high, or critical.")
	a.Describe(&i.RuleType, "The rule type: query, eql, threshold, machine_learning, new_terms, esql, or threat_match.")
	a.Describe(&i.Query, "The KQL, EQL, or ESQL query string.")
	a.Describe(&i.Language, "The query language: kuery, lucene, eql, or esql.")
	a.Describe(&i.IndexPatterns, "Index patterns to search.")
	a.Describe(&i.Filters, "Additional filters as a JSON string.")
	a.Describe(&i.Enabled, "Whether the rule is enabled. Defaults to true.")
	a.Describe(&i.Interval, "How often the rule runs, e.g. '5m'.")
	a.Describe(&i.FromTime, "Relative start time for the rule's query, e.g. 'now-360s'.")
	a.Describe(&i.ToTime, "Relative end time for the rule's query, e.g. 'now'.")
	a.Describe(&i.Tags, "Tags for the rule.")
	a.Describe(&i.Actions, "Actions as a JSON array string.")
	a.Describe(&i.ExceptionsList, "Exception list references as a JSON array string.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing rule into Pulumi state on create.")
	a.SetDefault(&i.Enabled, true)
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *SecurityDetectionRule) Create(
	ctx context.Context,
	req infer.CreateRequest[SecurityDetectionRuleInputs],
) (infer.CreateResponse[SecurityDetectionRuleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[SecurityDetectionRuleState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildDetectionRuleBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[SecurityDetectionRuleState]{}, err
	}

	if req.Inputs.AdoptOnCreate {
		// Check if a rule with same name exists by searching
		var searchResult []map[string]any
		searchPath := clients.SpacePath(
			spaceID,
			"/api/detection_engine/rules/_find?per_page=1&filter=alert.attributes.name:\""+req.Inputs.Name+"\"",
		)
		var findResp struct {
			Data []map[string]any `json:"data"`
		}
		if err := kbClient.GetJSON(ctx, searchPath, &findResp); err == nil && len(findResp.Data) > 0 {
			searchResult = findResp.Data
		}
		if len(searchResult) > 0 {
			existing := searchResult[0]
			id, _ := existing["id"].(string)
			ruleID, _ := existing["rule_id"].(string)
			body["rule_id"] = ruleID
			path := clients.SpacePath(spaceID, "/api/detection_engine/rules")
			if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
				return infer.CreateResponse[SecurityDetectionRuleState]{}, fmt.Errorf(
					"failed to update adopted detection rule: %w",
					err,
				)
			}
			return infer.CreateResponse[SecurityDetectionRuleState]{
				ID: id,
				Output: SecurityDetectionRuleState{
					SecurityDetectionRuleInputs: req.Inputs,
					RuleID:                      ruleID,
				},
			}, nil
		}
	}

	var result map[string]any
	path := clients.SpacePath(spaceID, "/api/detection_engine/rules")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[SecurityDetectionRuleState]{}, fmt.Errorf(
			"failed to create detection rule %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	id, _ := result["id"].(string)
	ruleID, _ := result["rule_id"].(string)

	return infer.CreateResponse[SecurityDetectionRuleState]{
		ID: id,
		Output: SecurityDetectionRuleState{
			SecurityDetectionRuleInputs: req.Inputs,
			RuleID:                      ruleID,
		},
	}, nil
}

// Read ...
func (r *SecurityDetectionRule) Read(
	ctx context.Context,
	req infer.ReadRequest[SecurityDetectionRuleInputs, SecurityDetectionRuleState],
) (infer.ReadResponse[SecurityDetectionRuleInputs, SecurityDetectionRuleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[SecurityDetectionRuleInputs, SecurityDetectionRuleState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/detection_engine/rules?rule_id=%s", req.State.RuleID))

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[SecurityDetectionRuleInputs, SecurityDetectionRuleState]{}, err
	}
	if !exists {
		return infer.ReadResponse[SecurityDetectionRuleInputs, SecurityDetectionRuleState]{ID: ""}, nil
	}

	return infer.ReadResponse[SecurityDetectionRuleInputs, SecurityDetectionRuleState](req), nil
}

// Update ...
func (r *SecurityDetectionRule) Update(
	ctx context.Context,
	req infer.UpdateRequest[SecurityDetectionRuleInputs, SecurityDetectionRuleState],
) (infer.UpdateResponse[SecurityDetectionRuleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[SecurityDetectionRuleState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildDetectionRuleBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[SecurityDetectionRuleState]{}, err
	}
	body["rule_id"] = req.State.RuleID

	path := clients.SpacePath(spaceID, "/api/detection_engine/rules")
	if err := kbClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.UpdateResponse[SecurityDetectionRuleState]{}, fmt.Errorf(
			"failed to update detection rule %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[SecurityDetectionRuleState]{
		Output: SecurityDetectionRuleState{
			SecurityDetectionRuleInputs: req.Inputs,
			RuleID:                      req.State.RuleID,
		},
	}, nil
}

// Delete ...
func (r *SecurityDetectionRule) Delete(
	ctx context.Context,
	req infer.DeleteRequest[SecurityDetectionRuleState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/detection_engine/rules?rule_id=%s", req.State.RuleID))

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildDetectionRuleBody(inputs SecurityDetectionRuleInputs) (map[string]any, error) {
	body := map[string]any{
		"name":        inputs.Name,
		"description": inputs.Description,
		"risk_score":  inputs.RiskScore,
		"severity":    inputs.Severity,
		"type":        inputs.RuleType,
	}

	if inputs.Query != nil {
		body["query"] = *inputs.Query
	}
	if inputs.Language != nil {
		body["language"] = *inputs.Language
	}
	if len(inputs.IndexPatterns) > 0 {
		body["index"] = inputs.IndexPatterns
	}
	if inputs.Filters != nil {
		var filters any
		if err := json.Unmarshal([]byte(*inputs.Filters), &filters); err != nil {
			return nil, fmt.Errorf("invalid filters JSON: %w", err)
		}
		body["filters"] = filters
	}
	if inputs.Enabled != nil {
		body["enabled"] = *inputs.Enabled
	}
	if inputs.Interval != nil {
		body["interval"] = *inputs.Interval
	}
	if inputs.FromTime != nil {
		body["from"] = *inputs.FromTime
	}
	if inputs.ToTime != nil {
		body["to"] = *inputs.ToTime
	}
	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}
	if inputs.Actions != nil {
		var actions any
		if err := json.Unmarshal([]byte(*inputs.Actions), &actions); err != nil {
			return nil, fmt.Errorf("invalid actions JSON: %w", err)
		}
		body["actions"] = actions
	}
	if inputs.ExceptionsList != nil {
		var exceptions any
		if err := json.Unmarshal([]byte(*inputs.ExceptionsList), &exceptions); err != nil {
			return nil, fmt.Errorf("invalid exceptions JSON: %w", err)
		}
		body["exceptions_list"] = exceptions
	}

	return body, nil
}

func resolveSpaceID(spaceID *string) string {
	if spaceID == nil || *spaceID == "" {
		return "default"
	}
	return *spaceID
}
