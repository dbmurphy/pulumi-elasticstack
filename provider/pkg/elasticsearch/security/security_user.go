package security

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// User manages an Elasticsearch user via PUT /_security/user/<username>.
type User struct{}

// UserInputs ...
type UserInputs struct {
	Username             string   `pulumi:"username"`
	Password             *string  `pulumi:"password,optional"             provider:"secret"`
	PasswordHash         *string  `pulumi:"passwordHash,optional"         provider:"secret"`
	Roles                []string `pulumi:"roles"`
	FullName             *string  `pulumi:"fullName,optional"`
	Email                *string  `pulumi:"email,optional"`
	Metadata             *string  `pulumi:"metadata,optional"`
	Enabled              *bool    `pulumi:"enabled,optional"`
	AdoptOnCreate        bool     `pulumi:"adoptOnCreate,optional"`
	IgnorePasswordOnRead bool     `pulumi:"ignorePasswordOnRead,optional"`
}

// UserState ...
type UserState struct {
	UserInputs
}

var (
	_ infer.CustomDelete[UserState]             = (*User)(nil)
	_ infer.CustomRead[UserInputs, UserState]   = (*User)(nil)
	_ infer.CustomUpdate[UserInputs, UserState] = (*User)(nil)
)

// Annotate ...
func (r *User) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch security user.")
	a.SetToken("elasticsearch", "User")
}

// Annotate ...
func (i *UserInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Username, "The username.")
	a.Describe(&i.Password, "Plaintext password.")
	a.Describe(&i.PasswordHash, "Bcrypt password hash.")
	a.Describe(&i.Roles, "Roles assigned to the user.")
	a.Describe(&i.FullName, "Full name of the user.")
	a.Describe(&i.Email, "Email address.")
	a.Describe(&i.Metadata, "User metadata as JSON.")
	a.Describe(&i.Enabled, "Whether the user is enabled.")
	a.SetDefault(&i.Enabled, true)
	a.Describe(&i.AdoptOnCreate, "Adopt existing user into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
	a.Describe(&i.IgnorePasswordOnRead, "Never diff on password (ES doesn't return it).")
	a.SetDefault(&i.IgnorePasswordOnRead, true)
}

// Create ...
func (r *User) Create(
	ctx context.Context,
	req infer.CreateRequest[UserInputs],
) (infer.CreateResponse[UserState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[UserState]{}, err
	}

	username := req.Inputs.Username

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_security/user/"+username)
		if err != nil {
			return infer.CreateResponse[UserState]{}, err
		}
		if exists {
			return infer.CreateResponse[UserState]{
				ID:     username,
				Output: UserState{UserInputs: req.Inputs},
			}, nil
		}
	}

	body, err := buildUserBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[UserState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_security/user/"+username, body, nil); err != nil {
		return infer.CreateResponse[UserState]{}, fmt.Errorf("failed to create user %s: %w", username, err)
	}

	return infer.CreateResponse[UserState]{
		ID:     username,
		Output: UserState{UserInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *User) Read(
	ctx context.Context,
	req infer.ReadRequest[UserInputs, UserState],
) (infer.ReadResponse[UserInputs, UserState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[UserInputs, UserState]{}, err
	}

	var result map[string]json.RawMessage
	if err := esClient.GetJSON(ctx, "/_security/user/"+req.ID, &result); err != nil {
		if clients.IsNotFound(err) {
			return infer.ReadResponse[UserInputs, UserState]{ID: ""}, nil
		}
		return infer.ReadResponse[UserInputs, UserState]{}, err
	}

	return infer.ReadResponse[UserInputs, UserState](req), nil
}

// Update ...
func (r *User) Update(
	ctx context.Context,
	req infer.UpdateRequest[UserInputs, UserState],
) (infer.UpdateResponse[UserState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[UserState]{}, err
	}

	body, err := buildUserBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[UserState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_security/user/"+req.Inputs.Username, body, nil); err != nil {
		return infer.UpdateResponse[UserState]{}, fmt.Errorf(
			"failed to update user %s: %w",
			req.Inputs.Username,
			err,
		)
	}

	return infer.UpdateResponse[UserState]{
		Output: UserState{UserInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *User) Delete(
	ctx context.Context,
	req infer.DeleteRequest[UserState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_security/user/"+req.State.Username); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildUserBody(inputs UserInputs) (map[string]any, error) {
	body := map[string]any{
		"roles": inputs.Roles,
	}

	if inputs.Password != nil {
		body["password"] = *inputs.Password
	}
	if inputs.PasswordHash != nil {
		body["password_hash"] = *inputs.PasswordHash
	}
	if inputs.FullName != nil {
		body["full_name"] = *inputs.FullName
	}
	if inputs.Email != nil {
		body["email"] = *inputs.Email
	}
	if inputs.Metadata != nil {
		var meta any
		if err := json.Unmarshal([]byte(*inputs.Metadata), &meta); err != nil {
			return nil, fmt.Errorf("invalid meta JSON: %w", err)
		}
		body["metadata"] = meta
	}
	if inputs.Enabled != nil {
		body["enabled"] = *inputs.Enabled
	}

	return body, nil
}

func boolPtr(b bool) *bool {
	return &b
}
