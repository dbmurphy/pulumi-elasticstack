package detection

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// SecurityEnableRule manages the enabled state of a Kibana security detection rule.
type SecurityEnableRule struct{}

// SecurityEnableRuleInputs ...
type SecurityEnableRuleInputs struct {
	RuleID  string  `pulumi:"ruleId"`
	Enabled bool    `pulumi:"enabled"`
	SpaceID *string `pulumi:"spaceId,optional"`
}

// SecurityEnableRuleState ...
type SecurityEnableRuleState struct {
	SecurityEnableRuleInputs
}

var (
	_ infer.CustomDelete[SecurityEnableRuleState]                           = (*SecurityEnableRule)(nil)
	_ infer.CustomRead[SecurityEnableRuleInputs, SecurityEnableRuleState]   = (*SecurityEnableRule)(nil)
	_ infer.CustomUpdate[SecurityEnableRuleInputs, SecurityEnableRuleState] = (*SecurityEnableRule)(nil)
)

// Annotate ...
func (r *SecurityEnableRule) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages the enabled/disabled state of a Kibana security detection rule.")
	a.SetToken("kibana", "SecurityEnableRule")
}

// Annotate ...
func (i *SecurityEnableRuleInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.RuleID, "The rule_id of the detection rule to enable or disable.")
	a.Describe(&i.Enabled, "Whether the rule should be enabled.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
}

// Create ...
func (r *SecurityEnableRule) Create(
	ctx context.Context,
	req infer.CreateRequest[SecurityEnableRuleInputs],
) (infer.CreateResponse[SecurityEnableRuleState], error) {
	if err := patchRuleEnabled(ctx, req.Inputs.SpaceID, req.Inputs.RuleID, req.Inputs.Enabled); err != nil {
		return infer.CreateResponse[SecurityEnableRuleState]{}, err
	}

	return infer.CreateResponse[SecurityEnableRuleState]{
		ID:     req.Inputs.RuleID,
		Output: SecurityEnableRuleState{SecurityEnableRuleInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *SecurityEnableRule) Read(
	ctx context.Context,
	req infer.ReadRequest[SecurityEnableRuleInputs, SecurityEnableRuleState],
) (infer.ReadResponse[SecurityEnableRuleInputs, SecurityEnableRuleState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.ReadResponse[SecurityEnableRuleInputs, SecurityEnableRuleState]{}, err
	}

	spaceID := resolveSpaceID(req.State.SpaceID)
	path := clients.SpacePath(spaceID, fmt.Sprintf("/api/detection_engine/rules?rule_id=%s", req.State.RuleID))

	exists, err := kbClient.Exists(ctx, path)
	if err != nil {
		return infer.ReadResponse[SecurityEnableRuleInputs, SecurityEnableRuleState]{}, err
	}
	if !exists {
		return infer.ReadResponse[SecurityEnableRuleInputs, SecurityEnableRuleState]{ID: ""}, nil
	}

	return infer.ReadResponse[SecurityEnableRuleInputs, SecurityEnableRuleState](req), nil
}

// Update ...
func (r *SecurityEnableRule) Update(
	ctx context.Context,
	req infer.UpdateRequest[SecurityEnableRuleInputs, SecurityEnableRuleState],
) (infer.UpdateResponse[SecurityEnableRuleState], error) {
	if err := patchRuleEnabled(ctx, req.Inputs.SpaceID, req.Inputs.RuleID, req.Inputs.Enabled); err != nil {
		return infer.UpdateResponse[SecurityEnableRuleState]{}, err
	}

	return infer.UpdateResponse[SecurityEnableRuleState]{
		Output: SecurityEnableRuleState{SecurityEnableRuleInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *SecurityEnableRule) Delete(
	_ context.Context,
	_ infer.DeleteRequest[SecurityEnableRuleState],
) (infer.DeleteResponse, error) {
	// No-op: removing from state does not disable or delete the rule.
	return infer.DeleteResponse{}, nil
}

func patchRuleEnabled(ctx context.Context, spaceIDPtr *string, ruleID string, enabled bool) error {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return err
	}

	spaceID := resolveSpaceID(spaceIDPtr)
	path := clients.SpacePath(spaceID, "/api/detection_engine/rules")

	body := map[string]any{
		"rule_id": ruleID,
		"enabled": enabled,
	}

	if err := kbClient.PatchJSON(ctx, path, body, nil); err != nil {
		return fmt.Errorf("failed to patch detection rule enabled state for %s: %w", ruleID, err)
	}

	return nil
}
