package detection

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// InstallPrebuiltRules installs or updates Elastic prebuilt detection rules.
type InstallPrebuiltRules struct{}

// InstallPrebuiltRulesInputs ...
type InstallPrebuiltRulesInputs struct {
	SpaceID *string `pulumi:"spaceId,optional"`
}

// InstallPrebuiltRulesState ...
type InstallPrebuiltRulesState struct {
	InstallPrebuiltRulesInputs

	// Outputs
	RulesInstalled int `pulumi:"rulesInstalled"`
	RulesUpdated   int `pulumi:"rulesUpdated"`
}

var (
	_ infer.CustomDelete[InstallPrebuiltRulesState]                           = (*InstallPrebuiltRules)(nil)
	_ infer.CustomRead[InstallPrebuiltRulesInputs, InstallPrebuiltRulesState] = (*InstallPrebuiltRules)(nil)
)

// Annotate ...
func (r *InstallPrebuiltRules) Annotate(a infer.Annotator) {
	a.Describe(r, "Installs or updates Elastic prebuilt detection rules in a Kibana space.")
	a.SetToken("kibana", "InstallPrebuiltRules")
}

// Annotate ...
func (i *InstallPrebuiltRulesInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
}

// Create ...
func (r *InstallPrebuiltRules) Create(
	ctx context.Context,
	req infer.CreateRequest[InstallPrebuiltRulesInputs],
) (infer.CreateResponse[InstallPrebuiltRulesState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[InstallPrebuiltRulesState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)
	path := clients.SpacePath(spaceID, "/api/detection_engine/rules/prepackaged")

	var result struct {
		RulesInstalled int `json:"rules_installed"`
		RulesUpdated   int `json:"rules_updated"`
	}

	if err := kbClient.PutJSON(ctx, path, nil, &result); err != nil {
		return infer.CreateResponse[InstallPrebuiltRulesState]{}, fmt.Errorf(
			"failed to install prebuilt rules: %w",
			err,
		)
	}

	return infer.CreateResponse[InstallPrebuiltRulesState]{
		ID: "prebuilt-rules",
		Output: InstallPrebuiltRulesState{
			InstallPrebuiltRulesInputs: req.Inputs,
			RulesInstalled:             result.RulesInstalled,
			RulesUpdated:               result.RulesUpdated,
		},
	}, nil
}

// Read ...
func (r *InstallPrebuiltRules) Read(
	_ context.Context,
	req infer.ReadRequest[InstallPrebuiltRulesInputs, InstallPrebuiltRulesState],
) (infer.ReadResponse[InstallPrebuiltRulesInputs, InstallPrebuiltRulesState], error) {
	// Singleton resource — always return current state.
	return infer.ReadResponse[InstallPrebuiltRulesInputs, InstallPrebuiltRulesState](req), nil
}

// Delete ...
func (r *InstallPrebuiltRules) Delete(
	_ context.Context,
	_ infer.DeleteRequest[InstallPrebuiltRulesState],
) (infer.DeleteResponse, error) {
	// No-op: prebuilt rules cannot be uninstalled via API.
	return infer.DeleteResponse{}, nil
}
