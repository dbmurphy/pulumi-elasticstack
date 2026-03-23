package snapshot

import (
	"context"
	"encoding/json"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// Repository manages a snapshot repository via PUT /_snapshot/<name>.
type Repository struct{}

// RepositoryInputs defines the input properties for a snapshot repository.
type RepositoryInputs struct {
	Name               string `pulumi:"name"`
	Type               string `pulumi:"type"`
	Settings           string `pulumi:"settings"`
	Verify             *bool  `pulumi:"verify,optional"`
	AdoptOnCreate      bool   `pulumi:"adoptOnCreate,optional"`
	DeletionProtection *bool  `pulumi:"deletionProtection,optional"`
}

// RepositoryState defines the output state for a snapshot repository.
type RepositoryState struct {
	RepositoryInputs
}

var (
	_ infer.CustomDelete[RepositoryState]                   = (*Repository)(nil)
	_ infer.CustomRead[RepositoryInputs, RepositoryState]   = (*Repository)(nil)
	_ infer.CustomUpdate[RepositoryInputs, RepositoryState] = (*Repository)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *Repository) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch snapshot repository.")
	a.SetToken("elasticsearch", "SnapshotRepository")
}

// Annotate sets input property descriptions and defaults.
func (i *RepositoryInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.Name, "Repository name.")
	a.Describe(&i.Type, "Repository type (fs, s3, gcs, azure, url, source).")
	a.Describe(&i.Settings, "Type-specific repository settings as JSON.")
	a.Describe(&i.Verify, "Verify the repository after creation.")
	a.Describe(&i.AdoptOnCreate, "Adopt existing snapshot repository into state.")
	a.SetDefault(&i.AdoptOnCreate, false)
	a.Describe(&i.DeletionProtection, "Prevent deletion on destroy. Defaults to true.")
	a.SetDefault(&i.DeletionProtection, true)
}

// Create provisions a new snapshot repository.
func (r *Repository) Create(
	ctx context.Context,
	req infer.CreateRequest[RepositoryInputs],
) (infer.CreateResponse[RepositoryState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[RepositoryState]{}, err
	}

	name := req.Inputs.Name

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_snapshot/"+name)
		if err != nil {
			return infer.CreateResponse[RepositoryState]{}, err
		}
		if exists {
			return infer.CreateResponse[RepositoryState]{
				ID:     name,
				Output: RepositoryState{RepositoryInputs: req.Inputs},
			}, nil
		}
	}

	body := buildRepoBody(req.Inputs)
	path := "/_snapshot/" + name
	if req.Inputs.Verify != nil && !*req.Inputs.Verify {
		path += "?verify=false"
	}

	if err := esClient.PutJSON(ctx, path, body, nil); err != nil {
		return infer.CreateResponse[RepositoryState]{}, fmt.Errorf(
			"failed to create snapshot repository %s: %w",
			name,
			err,
		)
	}

	return infer.CreateResponse[RepositoryState]{
		ID:     name,
		Output: RepositoryState{RepositoryInputs: req.Inputs},
	}, nil
}

// Read fetches the current state of the snapshot repository.
func (r *Repository) Read(
	ctx context.Context,
	req infer.ReadRequest[RepositoryInputs, RepositoryState],
) (infer.ReadResponse[RepositoryInputs, RepositoryState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[RepositoryInputs, RepositoryState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_snapshot/"+req.ID)
	if err != nil {
		return infer.ReadResponse[RepositoryInputs, RepositoryState]{}, err
	}
	if !exists {
		return infer.ReadResponse[RepositoryInputs, RepositoryState]{ID: ""}, nil
	}

	return infer.ReadResponse[RepositoryInputs, RepositoryState](req), nil
}

// Update modifies an existing snapshot repository.
func (r *Repository) Update(
	ctx context.Context,
	req infer.UpdateRequest[RepositoryInputs, RepositoryState],
) (infer.UpdateResponse[RepositoryState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[RepositoryState]{}, err
	}

	body := buildRepoBody(req.Inputs)
	if err := esClient.PutJSON(ctx, "/_snapshot/"+req.Inputs.Name, body, nil); err != nil {
		return infer.UpdateResponse[RepositoryState]{}, fmt.Errorf(
			"failed to update snapshot repository %s: %w",
			req.Inputs.Name,
			err,
		)
	}

	return infer.UpdateResponse[RepositoryState]{
		Output: RepositoryState{RepositoryInputs: req.Inputs},
	}, nil
}

// Delete removes the snapshot repository.
func (r *Repository) Delete(
	ctx context.Context,
	req infer.DeleteRequest[RepositoryState],
) (infer.DeleteResponse, error) {
	if req.State.DeletionProtection != nil && *req.State.DeletionProtection {
		p.GetLogger(ctx).Warning("Snapshot repository has deletionProtection enabled; skipping deletion.")
		return infer.DeleteResponse{}, nil
	}

	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		p.GetLogger(ctx).Warning("Provider-level destroyProtection is enabled; skipping snapshot repository deletion.")
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_snapshot/"+req.State.Name); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildRepoBody(inputs RepositoryInputs) map[string]any {
	body := map[string]any{
		"type": inputs.Type,
	}

	var settings any
	if err := json.Unmarshal([]byte(inputs.Settings), &settings); err == nil {
		body["settings"] = settings
	}

	return body
}

func boolPtr(b bool) *bool {
	return &b
}
