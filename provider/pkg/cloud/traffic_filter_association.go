package cloud

import (
	"context"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// TrafficFilterAssociation attaches a traffic filter ruleset to an Elastic Cloud deployment.
type TrafficFilterAssociation struct{}

// TrafficFilterAssociationInputs defines the input properties for a traffic filter association.
type TrafficFilterAssociationInputs struct {
	RulesetID    string `pulumi:"rulesetId"`
	DeploymentID string `pulumi:"deploymentId"`
}

// TrafficFilterAssociationState defines the output state for a traffic filter association.
type TrafficFilterAssociationState struct {
	TrafficFilterAssociationInputs
}

var (
	_ infer.CustomDiff[TrafficFilterAssociationInputs, TrafficFilterAssociationState] = (*TrafficFilterAssociation)(nil)
	_ infer.CustomDelete[TrafficFilterAssociationState]                               = (*TrafficFilterAssociation)(nil)
	_ infer.CustomRead[TrafficFilterAssociationInputs, TrafficFilterAssociationState] = (*TrafficFilterAssociation)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *TrafficFilterAssociation) Annotate(a infer.Annotator) {
	a.Describe(r, "Associates a traffic filter ruleset with an Elastic Cloud deployment. "+
		"This controls which traffic filter rules are enforced on the deployment.")
	a.SetToken("cloud", "TrafficFilterAssociation")
}

// Annotate sets input property descriptions and defaults.
func (i *TrafficFilterAssociationInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.RulesetID, "The traffic filter ruleset ID to associate.")
	a.Describe(&i.DeploymentID, "The deployment ID to attach the ruleset to.")
}

// Create provisions a new traffic filter association.
func (r *TrafficFilterAssociation) Create(
	ctx context.Context, req infer.CreateRequest[TrafficFilterAssociationInputs],
) (infer.CreateResponse[TrafficFilterAssociationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.CreateResponse[TrafficFilterAssociationState]{}, err
	}

	path := fmt.Sprintf("/deployments/traffic-filter/rulesets/%s/associations", req.Inputs.RulesetID)
	body := map[string]any{
		"entity_type": "deployment",
		"id":          req.Inputs.DeploymentID,
	}

	if err := cloudClient.PostJSON(ctx, path, body, nil); err != nil {
		return infer.CreateResponse[TrafficFilterAssociationState]{},
			fmt.Errorf("failed to associate traffic filter %s with deployment %s: %w",
				req.Inputs.RulesetID, req.Inputs.DeploymentID, err)
	}

	id := req.Inputs.RulesetID + "/" + req.Inputs.DeploymentID

	return infer.CreateResponse[TrafficFilterAssociationState]{
		ID: id,
		Output: TrafficFilterAssociationState{
			TrafficFilterAssociationInputs: req.Inputs,
		},
	}, nil
}

// Read fetches the current state of the traffic filter association.
func (r *TrafficFilterAssociation) Read(
	ctx context.Context,
	req infer.ReadRequest[TrafficFilterAssociationInputs, TrafficFilterAssociationState],
) (infer.ReadResponse[TrafficFilterAssociationInputs, TrafficFilterAssociationState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.ReadResponse[TrafficFilterAssociationInputs, TrafficFilterAssociationState]{}, err
	}

	// Check if the ruleset still exists and has this deployment association
	path := fmt.Sprintf("/deployments/traffic-filter/rulesets/%s?include_associations=true", req.State.RulesetID)
	var result struct {
		ID           string `json:"id"`
		Associations []struct {
			EntityType string `json:"entity_type"`
			ID         string `json:"id"`
		} `json:"associations"`
	}

	if err := cloudClient.GetJSON(ctx, path, &result); err != nil {
		if _, ok := err.(*clients.NotFoundError); ok {
			return infer.ReadResponse[TrafficFilterAssociationInputs, TrafficFilterAssociationState]{ID: ""}, nil
		}
		return infer.ReadResponse[TrafficFilterAssociationInputs, TrafficFilterAssociationState]{}, err
	}

	for _, assoc := range result.Associations {
		if assoc.EntityType == "deployment" && assoc.ID == req.State.DeploymentID {
			return infer.ReadResponse[TrafficFilterAssociationInputs, TrafficFilterAssociationState](req), nil
		}
	}

	// Association no longer exists
	return infer.ReadResponse[TrafficFilterAssociationInputs, TrafficFilterAssociationState]{ID: ""}, nil
}

// Diff computes the difference between old and new state.
func (r *TrafficFilterAssociation) Diff(
	_ context.Context,
	req infer.DiffRequest[TrafficFilterAssociationInputs, TrafficFilterAssociationState],
) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}

	if req.Inputs.RulesetID != req.State.RulesetID {
		diff["rulesetId"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if req.Inputs.DeploymentID != req.State.DeploymentID {
		diff["deploymentId"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}

	return p.DiffResponse{
		HasChanges:          len(diff) > 0,
		DetailedDiff:        diff,
		DeleteBeforeReplace: true,
	}, nil
}

// Delete removes the traffic filter association.
func (r *TrafficFilterAssociation) Delete(
	ctx context.Context, req infer.DeleteRequest[TrafficFilterAssociationState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	path := fmt.Sprintf("/deployments/traffic-filter/rulesets/%s/associations/deployment/%s",
		req.State.RulesetID, req.State.DeploymentID)

	if err := cloudClient.Delete(ctx, path); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to remove traffic filter association: %w", err)
	}

	return infer.DeleteResponse{}, nil
}
