package ml

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// ---------------------------------------------------------------------------
// DatafeedStateControl — starts or stops a datafeed
// ---------------------------------------------------------------------------

// DatafeedStateControl manages the running state of an ML datafeed via
// POST /_ml/datafeeds/<datafeed_id>/_start or _stop.
type DatafeedStateControl struct{}

// DatafeedStateControlInputs ...
type DatafeedStateControlInputs struct {
	DatafeedId string `pulumi:"datafeedId"`
	Started    bool   `pulumi:"started"`
}

// DatafeedStateControlState ...
type DatafeedStateControlState struct {
	DatafeedStateControlInputs
}

var (
	_ infer.CustomDelete[DatafeedStateControlState]                             = (*DatafeedStateControl)(nil)
	_ infer.CustomUpdate[DatafeedStateControlInputs, DatafeedStateControlState] = (*DatafeedStateControl)(nil)
)

// Annotate ...
func (r *DatafeedStateControl) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages the running state (started/stopped) of an Elasticsearch ML datafeed.")
	a.SetToken("elasticsearch", "DatafeedState")
}

// Annotate ...
func (i *DatafeedStateControlInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.DatafeedId, "The unique identifier for the datafeed.")
	a.Describe(&i.Started, "Whether the datafeed should be started (true) or stopped (false).")
}

// Create ...
func (r *DatafeedStateControl) Create(
	ctx context.Context,
	req infer.CreateRequest[DatafeedStateControlInputs],
) (infer.CreateResponse[DatafeedStateControlState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[DatafeedStateControlState]{}, err
	}

	datafeedId := req.Inputs.DatafeedId

	if err := setDatafeedRunState(ctx, esClient, datafeedId, req.Inputs.Started); err != nil {
		return infer.CreateResponse[DatafeedStateControlState]{}, err
	}

	return infer.CreateResponse[DatafeedStateControlState]{
		ID:     datafeedId,
		Output: DatafeedStateControlState{DatafeedStateControlInputs: req.Inputs},
	}, nil
}

// Update ...
func (r *DatafeedStateControl) Update(
	ctx context.Context,
	req infer.UpdateRequest[DatafeedStateControlInputs, DatafeedStateControlState],
) (infer.UpdateResponse[DatafeedStateControlState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[DatafeedStateControlState]{}, err
	}

	if err := setDatafeedRunState(ctx, esClient, req.Inputs.DatafeedId, req.Inputs.Started); err != nil {
		return infer.UpdateResponse[DatafeedStateControlState]{}, err
	}

	return infer.UpdateResponse[DatafeedStateControlState]{
		Output: DatafeedStateControlState{DatafeedStateControlInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *DatafeedStateControl) Delete(
	ctx context.Context,
	req infer.DeleteRequest[DatafeedStateControlState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	// On delete, stop the datafeed to leave it in a clean state.
	if err := setDatafeedRunState(ctx, esClient, req.State.DatafeedId, false); err != nil {
		return infer.DeleteResponse{}, err
	}

	return infer.DeleteResponse{}, nil
}

func setDatafeedRunState(ctx context.Context, esClient interface {
	PostJSON(ctx context.Context, path string, body any, dest any) error
}, datafeedId string, started bool,
) error {
	action := "_stop"
	if started {
		action = "_start"
	}
	path := fmt.Sprintf("/_ml/datafeeds/%s/%s", datafeedId, action)
	if err := esClient.PostJSON(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("failed to %s datafeed %s: %w", action[1:], datafeedId, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// JobStateControl — opens or closes an anomaly detection job
// ---------------------------------------------------------------------------

// JobStateControl manages the running state of an ML anomaly detection job via
// POST /_ml/anomaly_detectors/<job_id>/_open or _close.
type JobStateControl struct{}

// JobStateControlInputs ...
type JobStateControlInputs struct {
	JobId  string `pulumi:"jobId"`
	Opened bool   `pulumi:"opened"`
}

// JobStateControlState ...
type JobStateControlState struct {
	JobStateControlInputs
}

var (
	_ infer.CustomDelete[JobStateControlState]                        = (*JobStateControl)(nil)
	_ infer.CustomUpdate[JobStateControlInputs, JobStateControlState] = (*JobStateControl)(nil)
)

// Annotate ...
func (r *JobStateControl) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages the running state (opened/closed) of an Elasticsearch ML anomaly detection job.")
	a.SetToken("elasticsearch", "MlJobState")
}

// Annotate ...
func (i *JobStateControlInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.JobId, "The unique identifier for the anomaly detection job.")
	a.Describe(&i.Opened, "Whether the job should be opened (true) or closed (false).")
}

// Create ...
func (r *JobStateControl) Create(
	ctx context.Context,
	req infer.CreateRequest[JobStateControlInputs],
) (infer.CreateResponse[JobStateControlState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[JobStateControlState]{}, err
	}

	jobId := req.Inputs.JobId

	if err := setJobRunState(ctx, esClient, jobId, req.Inputs.Opened); err != nil {
		return infer.CreateResponse[JobStateControlState]{}, err
	}

	return infer.CreateResponse[JobStateControlState]{
		ID:     jobId,
		Output: JobStateControlState{JobStateControlInputs: req.Inputs},
	}, nil
}

// Update ...
func (r *JobStateControl) Update(
	ctx context.Context,
	req infer.UpdateRequest[JobStateControlInputs, JobStateControlState],
) (infer.UpdateResponse[JobStateControlState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[JobStateControlState]{}, err
	}

	if err := setJobRunState(ctx, esClient, req.Inputs.JobId, req.Inputs.Opened); err != nil {
		return infer.UpdateResponse[JobStateControlState]{}, err
	}

	return infer.UpdateResponse[JobStateControlState]{
		Output: JobStateControlState{JobStateControlInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *JobStateControl) Delete(
	ctx context.Context,
	req infer.DeleteRequest[JobStateControlState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	// On delete, close the job to leave it in a clean state.
	if err := setJobRunState(ctx, esClient, req.State.JobId, false); err != nil {
		return infer.DeleteResponse{}, err
	}

	return infer.DeleteResponse{}, nil
}

func setJobRunState(ctx context.Context, esClient interface {
	PostJSON(ctx context.Context, path string, body any, dest any) error
}, jobId string, opened bool,
) error {
	action := "_close"
	if opened {
		action = "_open"
	}
	path := fmt.Sprintf("/_ml/anomaly_detectors/%s/%s", jobId, action)
	if err := esClient.PostJSON(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("failed to %s anomaly detection job %s: %w", action[1:], jobId, err)
	}
	return nil
}
