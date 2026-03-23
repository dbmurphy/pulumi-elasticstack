package security

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// SystemUser manages passwords of built-in system users.
type SystemUser struct{}

// SystemUserInputs ...
type SystemUserInputs struct {
	Username     string  `pulumi:"username"`
	Password     *string `pulumi:"password,optional"     provider:"secret"`
	PasswordHash *string `pulumi:"passwordHash,optional" provider:"secret"`
}

// SystemUserState ...
type SystemUserState struct {
	SystemUserInputs
}

var _ infer.CustomUpdate[SystemUserInputs, SystemUserState] = (*SystemUser)(nil)

// Annotate ...
func (r *SystemUser) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages passwords of built-in Elasticsearch system users (elastic, kibana_system, etc.).")
	a.SetToken("elasticsearch", "SystemUser")
}

// Annotate ...
func (i *SystemUserInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Username, "The built-in username (e.g. elastic, kibana_system).")
	a.Describe(&i.Password, "New plaintext password.")
	a.Describe(&i.PasswordHash, "New bcrypt password hash.")
}

// Create ...
func (r *SystemUser) Create(
	ctx context.Context,
	req infer.CreateRequest[SystemUserInputs],
) (infer.CreateResponse[SystemUserState], error) {
	if err := changeSystemPassword(ctx, req.Inputs); err != nil {
		return infer.CreateResponse[SystemUserState]{}, err
	}

	return infer.CreateResponse[SystemUserState]{
		ID:     req.Inputs.Username,
		Output: SystemUserState{SystemUserInputs: req.Inputs},
	}, nil
}

// Update ...
func (r *SystemUser) Update(
	ctx context.Context,
	req infer.UpdateRequest[SystemUserInputs, SystemUserState],
) (infer.UpdateResponse[SystemUserState], error) {
	if err := changeSystemPassword(ctx, req.Inputs); err != nil {
		return infer.UpdateResponse[SystemUserState]{}, err
	}

	return infer.UpdateResponse[SystemUserState]{
		Output: SystemUserState{SystemUserInputs: req.Inputs},
	}, nil
}

func changeSystemPassword(ctx context.Context, inputs SystemUserInputs) error {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return err
	}

	body := map[string]any{}
	if inputs.Password != nil {
		body["password"] = *inputs.Password
	}
	if inputs.PasswordHash != nil {
		body["password_hash"] = *inputs.PasswordHash
	}

	if err := esClient.PostJSON(ctx, "/_security/user/"+inputs.Username+"/_password", body, nil); err != nil {
		return fmt.Errorf("failed to change password for system user %s: %w", inputs.Username, err)
	}

	return nil
}
