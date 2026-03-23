// Package savedobj implements Kibana saved object import management.
package savedobj

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// ImportSavedObjects imports saved objects into Kibana from NDJSON content.
type ImportSavedObjects struct{}

// ImportSavedObjectsInputs defines the input properties for a saved objects import.
type ImportSavedObjectsInputs struct {
	FileContents string  `pulumi:"fileContents"`
	Overwrite    *bool   `pulumi:"overwrite,optional"`
	SpaceID      *string `pulumi:"spaceID,optional"`
}

// ImportSavedObjectsState defines the output state for a saved objects import.
type ImportSavedObjectsState struct {
	ImportSavedObjectsInputs

	// Computed outputs
	Success      bool `pulumi:"success"`
	SuccessCount int  `pulumi:"successCount"`
}

var _ infer.CustomDelete[ImportSavedObjectsState] = (*ImportSavedObjects)(nil)

// Annotate sets resource metadata and descriptions.
func (r *ImportSavedObjects) Annotate(a infer.Annotator) {
	a.Describe(r, "Imports saved objects into Kibana from NDJSON content.")
	a.SetToken("kibana", "ImportSavedObjects")
}

// Annotate sets input property descriptions and defaults.
func (i *ImportSavedObjectsInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.FileContents, "NDJSON content of saved objects to import.")
	a.Describe(&i.Overwrite, "Overwrite existing saved objects. Defaults to true.")
	a.SetDefault(&i.Overwrite, true)
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
}

// Create provisions a new saved objects import.
func (r *ImportSavedObjects) Create(
	ctx context.Context,
	req infer.CreateRequest[ImportSavedObjectsInputs],
) (infer.CreateResponse[ImportSavedObjectsState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[ImportSavedObjectsState]{}, err
	}

	spaceID := derefString(req.Inputs.SpaceID)
	path := clients.SpacePath(spaceID, "/api/saved_objects/_import")

	overwrite := true
	if req.Inputs.Overwrite != nil {
		overwrite = *req.Inputs.Overwrite
	}
	if overwrite {
		path += "?overwrite=true"
	}

	var result struct {
		Success      bool `json:"success"`
		SuccessCount int  `json:"successCount"`
	}

	// The import endpoint accepts NDJSON. The Do method sets Content-Type to
	// application/json, which Kibana accepts for this endpoint in practice.
	if err := kbClient.PostRaw(ctx, path, "application/ndjson", []byte(req.Inputs.FileContents), &result); err != nil {
		return infer.CreateResponse[ImportSavedObjectsState]{}, fmt.Errorf("failed to import saved objects: %w", err)
	}

	// Generate a stable ID from the content hash
	id := contentHash(req.Inputs.FileContents)

	return infer.CreateResponse[ImportSavedObjectsState]{
		ID: id,
		Output: ImportSavedObjectsState{
			ImportSavedObjectsInputs: req.Inputs,
			Success:                  result.Success,
			SuccessCount:             result.SuccessCount,
		},
	}, nil
}

// Delete removes the saved objects import resource.
func (r *ImportSavedObjects) Delete(
	_ context.Context,
	_ infer.DeleteRequest[ImportSavedObjectsState],
) (infer.DeleteResponse, error) {
	// Cannot un-import saved objects — this is a no-op.
	return infer.DeleteResponse{}, nil
}

func contentHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:16])
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
