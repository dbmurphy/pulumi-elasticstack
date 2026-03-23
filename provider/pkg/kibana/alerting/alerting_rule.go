package alerting

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Rule manages a Kibana alerting rule via the Alerting API.
type Rule struct{}

// RuleInputs ...
type RuleInputs struct {
	Name          string   `pulumi:"name"`
	Consumer      string   `pulumi:"consumer"`
	RuleTypeID    string   `pulumi:"ruleTypeId"`
	Schedule      string   `pulumi:"schedule"`
	Params        string   `pulumi:"params"`
	Actions       *string  `pulumi:"actions,optional"`
	Enabled       *bool    `pulumi:"enabled,optional"`
	Tags          []string `pulumi:"tags,optional"`
	Throttle      *string  `pulumi:"throttle,optional"`
	NotifyWhen    *string  `pulumi:"notifyWhen,optional"`
	SpaceID       *string  `pulumi:"spaceId,optional"`
	AdoptOnCreate bool     `pulumi:"adoptOnCreate,optional"`
}

// RuleState ...
type RuleState struct {
	RuleInputs

	// Outputs
	RuleID          string `pulumi:"ruleId"`
	ScheduledTaskID string `pulumi:"scheduledTaskId"`
}

var (
	_ infer.CustomDelete[RuleState]             = (*Rule)(nil)
	_ infer.CustomRead[RuleInputs, RuleState]   = (*Rule)(nil)
	_ infer.CustomUpdate[RuleInputs, RuleState] = (*Rule)(nil)
)

// Annotate ...
func (r *Rule) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages a Kibana alerting rule.")
	a.SetToken("kibana", "Rule")
}

// Annotate ...
func (i *RuleInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "The name of the alerting rule.")
	a.Describe(&i.Consumer, "The consumer application for the rule (e.g. 'alerts', 'siem').")
	a.Describe(&i.RuleTypeID, "The rule type ID (e.g. '.es-query', '.index-threshold').")
	a.Describe(&i.Schedule, "The rule schedule as a JSON object (e.g. '{\"interval\": \"1m\"}').")
	a.Describe(&i.Params, "The rule parameters as a JSON string.")
	a.Describe(&i.Actions, "The rule actions as a JSON array string.")
	a.Describe(&i.Enabled, "Whether the rule is enabled. Defaults to true.")
	a.SetDefault(&i.Enabled, true)
	a.Describe(&i.Tags, "Tags for the rule.")
	a.Describe(&i.Throttle, "The throttle interval (e.g. '1m', '1h').")
	a.Describe(&i.NotifyWhen, "When to notify (e.g. 'onActionGroupChange', 'onActiveAlert', 'onThrottleInterval').")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing rule into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *Rule) Create(
	ctx context.Context,
	req infer.CreateRequest[RuleInputs],
) (infer.CreateResponse[RuleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[RuleState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	body, err := buildRuleCreateBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[RuleState]{}, err
	}

	var result struct {
		ID              string `json:"id"`
		ScheduledTaskID string `json:"scheduled_task_id"`
	}

	path := clients.SpacePath(spaceID, "/api/alerting/rule")
	if err := kbClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[RuleState]{}, fmt.Errorf(
			"failed to create alerting rule %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	return infer.CreateResponse[RuleState]{
		ID: result.ID,
		Output: RuleState{
			RuleInputs:      req.Inputs,
			RuleID:          result.ID,
			ScheduledTaskID: result.ScheduledTaskID,
		},
	}, nil
}

// Read ...
func (r *Rule) Read(
	ctx context.Context,
	req infer.ReadRequest[RuleInputs, RuleState],
) (infer.ReadResponse[RuleInputs, RuleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[RuleInputs, RuleState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/alerting/rule/"+req.ID)

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[RuleInputs, RuleState]{}, err
	}
	if !exists {
		return infer.ReadResponse[RuleInputs, RuleState]{ID: ""}, nil
	}

	return infer.ReadResponse[RuleInputs, RuleState](req), nil
}

// Update ...
func (r *Rule) Update(
	ctx context.Context,
	req infer.UpdateRequest[RuleInputs, RuleState],
) (infer.UpdateResponse[RuleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.UpdateResponse[RuleState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)

	// PUT does not accept consumer or ruleTypeId — they are immutable.
	body, err := buildRuleUpdateBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[RuleState]{}, err
	}
	path := clients.SpacePath(spaceID, "/api/alerting/rule/"+req.ID)

	var result struct {
		ID              string `json:"id"`
		ScheduledTaskID string `json:"scheduled_task_id"`
	}

	if err := kbClient.PutJSON(ctx, path, body, &result); err != nil {
		return infer.UpdateResponse[RuleState]{}, fmt.Errorf(
			"failed to update alerting rule %s: %w",
			req.ID,
			err,
		)
	}

	return infer.UpdateResponse[RuleState]{
		Output: RuleState{
			RuleInputs:      req.Inputs,
			RuleID:          result.ID,
			ScheduledTaskID: result.ScheduledTaskID,
		},
	}, nil
}

// Delete ...
func (r *Rule) Delete(
	ctx context.Context,
	req infer.DeleteRequest[RuleState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, "/api/alerting/rule/"+req.State.RuleID)

	if err := kbClient.Delete(ctx, path); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

// buildRuleCreateBody builds the full request body for POST /api/alerting/rule.
func buildRuleCreateBody(inputs RuleInputs) (map[string]any, error) {
	body := map[string]any{
		"name":         inputs.Name,
		"consumer":     inputs.Consumer,
		"rule_type_id": inputs.RuleTypeID,
	}

	var schedule any
	if err := json.Unmarshal([]byte(inputs.Schedule), &schedule); err != nil {
		return nil, fmt.Errorf("invalid schedule JSON: %w", err)
	}
	body["schedule"] = schedule

	var params any
	if err := json.Unmarshal([]byte(inputs.Params), &params); err != nil {
		return nil, fmt.Errorf("invalid params JSON: %w", err)
	}
	body["params"] = params

	if inputs.Actions != nil {
		var actions any
		if err := json.Unmarshal([]byte(*inputs.Actions), &actions); err != nil {
			return nil, fmt.Errorf("invalid actions JSON: %w", err)
		}
		body["actions"] = actions
	}
	if inputs.Enabled != nil {
		body["enabled"] = *inputs.Enabled
	}
	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}
	if inputs.Throttle != nil {
		body["throttle"] = *inputs.Throttle
	}
	if inputs.NotifyWhen != nil {
		body["notify_when"] = *inputs.NotifyWhen
	}

	return body, nil
}

// buildRuleUpdateBody builds the request body for PUT /api/alerting/rule/{id}.
// The PUT endpoint does NOT accept consumer or ruleTypeId — they are immutable.
func buildRuleUpdateBody(inputs RuleInputs) (map[string]any, error) {
	body := map[string]any{
		"name": inputs.Name,
	}

	var schedule any
	if err := json.Unmarshal([]byte(inputs.Schedule), &schedule); err != nil {
		return nil, fmt.Errorf("invalid schedule JSON: %w", err)
	}
	body["schedule"] = schedule

	var params any
	if err := json.Unmarshal([]byte(inputs.Params), &params); err != nil {
		return nil, fmt.Errorf("invalid params JSON: %w", err)
	}
	body["params"] = params

	if inputs.Actions != nil {
		var actions any
		if err := json.Unmarshal([]byte(*inputs.Actions), &actions); err != nil {
			return nil, fmt.Errorf("invalid actions JSON: %w", err)
		}
		body["actions"] = actions
	}
	if len(inputs.Tags) > 0 {
		body["tags"] = inputs.Tags
	}
	if inputs.Throttle != nil {
		body["throttle"] = *inputs.Throttle
	}
	if inputs.NotifyWhen != nil {
		body["notify_when"] = *inputs.NotifyWhen
	}

	return body, nil
}
