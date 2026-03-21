package detection

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	clients "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	provider "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// SecurityListDataStreams manages the data stream associations for a security list.
type SecurityListDataStreams struct{}

// SecurityListDataStreamsInputs ...
type SecurityListDataStreamsInputs struct {
	ListID      string  `pulumi:"listId"`
	DataStreams string  `pulumi:"dataStreams"`
	SpaceID     *string `pulumi:"spaceId,optional"`
}

// SecurityListDataStreamsState ...
type SecurityListDataStreamsState struct {
	SecurityListDataStreamsInputs
}

var (
	_ infer.CustomDelete[SecurityListDataStreamsState]                              = (*SecurityListDataStreams)(nil)
	_ infer.CustomRead[SecurityListDataStreamsInputs, SecurityListDataStreamsState] = (*SecurityListDataStreams)(nil)
)

// Annotate ...
func (r *SecurityListDataStreams) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages data stream associations for a Kibana security list.")
	a.SetToken("kibana", "SecurityListDataStreams")
}

// Annotate ...
func (i *SecurityListDataStreamsInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.ListID, "The parent list ID.")
	a.Describe(&i.DataStreams, "Data stream objects as a JSON array string.")
	a.Describe(&i.SpaceID, "The Kibana space ID. Defaults to 'default'.")
}

// Create ...
func (r *SecurityListDataStreams) Create(
	ctx context.Context,
	req infer.CreateRequest[SecurityListDataStreamsInputs],
) (infer.CreateResponse[SecurityListDataStreamsState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	kbClient, err := cfg.KibanaClient()
	if err != nil {
		return infer.CreateResponse[SecurityListDataStreamsState]{}, err
	}

	spaceID := resolveSpaceID(req.Inputs.SpaceID)

	var dataStreams any
	if err := json.Unmarshal([]byte(req.Inputs.DataStreams), &dataStreams); err != nil {
		return infer.CreateResponse[SecurityListDataStreamsState]{}, fmt.Errorf(
			"failed to parse dataStreams JSON: %w",
			err,
		)
	}

	body := map[string]any{
		"list_id":      req.Inputs.ListID,
		"data_streams": dataStreams,
	}

	path := clients.SpacePath(spaceID, "/api/lists/data_streams")
	if err := kbClient.PostJSON(ctx, path, body, nil); err != nil {
		return infer.CreateResponse[SecurityListDataStreamsState]{}, fmt.Errorf(
			"failed to create list data streams for %s: %w",
			req.Inputs.ListID,
			err,
		)
	}

	return infer.CreateResponse[SecurityListDataStreamsState]{
		ID:     req.Inputs.ListID,
		Output: SecurityListDataStreamsState{SecurityListDataStreamsInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *SecurityListDataStreams) Read(
	_ context.Context,
	req infer.ReadRequest[SecurityListDataStreamsInputs, SecurityListDataStreamsState],
) (infer.ReadResponse[SecurityListDataStreamsInputs, SecurityListDataStreamsState], error) {
	// Minimal resource — return current state as-is.
	return infer.ReadResponse[SecurityListDataStreamsInputs, SecurityListDataStreamsState](req), nil
}

// Delete ...
func (r *SecurityListDataStreams) Delete(
	_ context.Context,
	_ infer.DeleteRequest[SecurityListDataStreamsState],
) (infer.DeleteResponse, error) {
	// No-op: data stream associations are removed when the list is deleted.
	return infer.DeleteResponse{}, nil
}
