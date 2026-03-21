package ml

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// AnomalyDetectionJob manages an ML anomaly detection job via PUT /_ml/anomaly_detectors/<job_id>.
type AnomalyDetectionJob struct{}

// AnomalyDetectionJobInputs ...
type AnomalyDetectionJobInputs struct {
	JobId                                string   `pulumi:"jobId"`
	AnalysisConfig                       string   `pulumi:"analysisConfig"`
	DataDescription                      string   `pulumi:"dataDescription"`
	AnalysisLimits                       *string  `pulumi:"analysisLimits,optional"`
	ModelSnapshotRetentionDays           *int     `pulumi:"modelSnapshotRetentionDays,optional"`
	DailyModelSnapshotRetentionAfterDays *int     `pulumi:"dailyModelSnapshotRetentionAfterDays,optional"`
	ResultsIndexName                     *string  `pulumi:"resultsIndexName,optional"`
	AllowLazyOpen                        *bool    `pulumi:"allowLazyOpen,optional"`
	Description                          *string  `pulumi:"description,optional"`
	Groups                               []string `pulumi:"groups,optional"`
	CustomSettings                       *string  `pulumi:"customSettings,optional"`
	AdoptOnCreate                        bool     `pulumi:"adoptOnCreate,optional"`
}

// AnomalyDetectionJobState ...
type AnomalyDetectionJobState struct {
	AnomalyDetectionJobInputs
}

var (
	_ infer.CustomDelete[AnomalyDetectionJobState]                            = (*AnomalyDetectionJob)(nil)
	_ infer.CustomRead[AnomalyDetectionJobInputs, AnomalyDetectionJobState]   = (*AnomalyDetectionJob)(nil)
	_ infer.CustomUpdate[AnomalyDetectionJobInputs, AnomalyDetectionJobState] = (*AnomalyDetectionJob)(nil)
)

// Annotate ...
func (r *AnomalyDetectionJob) Annotate(a infer.Annotator) {
	a.Describe(r, "Manages an Elasticsearch ML anomaly detection job.")
	a.SetToken("elasticsearch", "AnomalyDetectionJob")
}

// Annotate ...
func (i *AnomalyDetectionJobInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.JobId, "The unique identifier for the anomaly detection job.")
	a.Describe(&i.AnalysisConfig, "Analysis configuration as JSON. Specifies how to analyze the data.")
	a.Describe(&i.DataDescription, "Data description as JSON. Describes the format of the input data.")
	a.Describe(&i.AnalysisLimits, "Analysis limits as JSON. Limits can be applied to the resources used by the job.")
	a.Describe(
		&i.ModelSnapshotRetentionDays,
		"The number of days to retain model snapshots. Older snapshots are deleted.",
	)
	a.Describe(
		&i.DailyModelSnapshotRetentionAfterDays,
		"The number of days after which daily model snapshots are retained. After this period, only the first snapshot per day is kept.",
	)
	a.Describe(&i.ResultsIndexName, "A text string that affects the name of the machine learning results index.")
	a.Describe(
		&i.AllowLazyOpen,
		"Whether to allow the job to be opened when there is not a machine learning node with sufficient capacity.",
	)
	a.Describe(&i.Description, "A description of the job.")
	a.Describe(&i.Groups, "A list of job groups.")
	a.Describe(&i.CustomSettings, "Custom settings as JSON applied to the job.")
	a.Describe(&i.AdoptOnCreate, "Adopt an existing anomaly detection job into Pulumi state on create.")
	a.SetDefault(&i.AdoptOnCreate, false)
}

// Create ...
func (r *AnomalyDetectionJob) Create(
	ctx context.Context,
	req infer.CreateRequest[AnomalyDetectionJobInputs],
) (infer.CreateResponse[AnomalyDetectionJobState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.CreateResponse[AnomalyDetectionJobState]{}, err
	}

	jobId := req.Inputs.JobId

	if req.Inputs.AdoptOnCreate {
		exists, err := esClient.Exists(ctx, "/_ml/anomaly_detectors/"+jobId)
		if err != nil {
			return infer.CreateResponse[AnomalyDetectionJobState]{}, err
		}
		if exists {
			body, err := buildAnomalyDetectionJobBody(req.Inputs)
			if err != nil {
				return infer.CreateResponse[AnomalyDetectionJobState]{}, err
			}
			if err := esClient.PutJSON(ctx, "/_ml/anomaly_detectors/"+jobId+"/_update", body, nil); err != nil {
				return infer.CreateResponse[AnomalyDetectionJobState]{}, fmt.Errorf(
					"failed to update adopted anomaly detection job %s: %w",
					jobId,
					err,
				)
			}
			return infer.CreateResponse[AnomalyDetectionJobState]{
				ID:     jobId,
				Output: AnomalyDetectionJobState{AnomalyDetectionJobInputs: req.Inputs},
			}, nil
		}
	}

	body, err := buildAnomalyDetectionJobBody(req.Inputs)
	if err != nil {
		return infer.CreateResponse[AnomalyDetectionJobState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_ml/anomaly_detectors/"+jobId, body, nil); err != nil {
		return infer.CreateResponse[AnomalyDetectionJobState]{}, fmt.Errorf(
			"failed to create anomaly detection job %s: %w",
			jobId,
			err,
		)
	}

	return infer.CreateResponse[AnomalyDetectionJobState]{
		ID:     jobId,
		Output: AnomalyDetectionJobState{AnomalyDetectionJobInputs: req.Inputs},
	}, nil
}

// Read ...
func (r *AnomalyDetectionJob) Read(
	ctx context.Context,
	req infer.ReadRequest[AnomalyDetectionJobInputs, AnomalyDetectionJobState],
) (infer.ReadResponse[AnomalyDetectionJobInputs, AnomalyDetectionJobState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.ReadResponse[AnomalyDetectionJobInputs, AnomalyDetectionJobState]{}, err
	}

	exists, err := esClient.Exists(ctx, "/_ml/anomaly_detectors/"+req.ID)
	if err != nil {
		return infer.ReadResponse[AnomalyDetectionJobInputs, AnomalyDetectionJobState]{}, err
	}
	if !exists {
		return infer.ReadResponse[AnomalyDetectionJobInputs, AnomalyDetectionJobState]{ID: ""}, nil
	}

	return infer.ReadResponse[AnomalyDetectionJobInputs, AnomalyDetectionJobState](req), nil
}

// Update ...
func (r *AnomalyDetectionJob) Update(
	ctx context.Context,
	req infer.UpdateRequest[AnomalyDetectionJobInputs, AnomalyDetectionJobState],
) (infer.UpdateResponse[AnomalyDetectionJobState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.UpdateResponse[AnomalyDetectionJobState]{}, err
	}

	body, err := buildAnomalyDetectionJobBody(req.Inputs)
	if err != nil {
		return infer.UpdateResponse[AnomalyDetectionJobState]{}, err
	}
	if err := esClient.PutJSON(ctx, "/_ml/anomaly_detectors/"+req.Inputs.JobId, body, nil); err != nil {
		return infer.UpdateResponse[AnomalyDetectionJobState]{}, fmt.Errorf(
			"failed to update anomaly detection job %s: %w",
			req.Inputs.JobId,
			err,
		)
	}

	return infer.UpdateResponse[AnomalyDetectionJobState]{
		Output: AnomalyDetectionJobState{AnomalyDetectionJobInputs: req.Inputs},
	}, nil
}

// Delete ...
func (r *AnomalyDetectionJob) Delete(
	ctx context.Context,
	req infer.DeleteRequest[AnomalyDetectionJobState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	if cfg.DestroyProtection {
		return infer.DeleteResponse{}, nil
	}

	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := esClient.Delete(ctx, "/_ml/anomaly_detectors/"+req.State.JobId+"?force=true"); err != nil {
		if !clients.IsNotFound(err) {
			return infer.DeleteResponse{}, err
		}
	}

	return infer.DeleteResponse{}, nil
}

func buildAnomalyDetectionJobBody(inputs AnomalyDetectionJobInputs) (map[string]any, error) {
	body := map[string]any{}

	var analysisConfig any
	if err := json.Unmarshal([]byte(inputs.AnalysisConfig), &analysisConfig); err != nil {
		return nil, fmt.Errorf("invalid analysisConfig JSON: %w", err)
	}
	body["analysis_config"] = analysisConfig

	var dataDescription any
	if err := json.Unmarshal([]byte(inputs.DataDescription), &dataDescription); err != nil {
		return nil, fmt.Errorf("invalid dataDescription JSON: %w", err)
	}
	body["data_description"] = dataDescription

	if inputs.AnalysisLimits != nil {
		var analysisLimits any
		if err := json.Unmarshal([]byte(*inputs.AnalysisLimits), &analysisLimits); err != nil {
			return nil, fmt.Errorf("invalid analysisLimits JSON: %w", err)
		}
		body["analysis_limits"] = analysisLimits
	}
	if inputs.ModelSnapshotRetentionDays != nil {
		body["model_snapshot_retention_days"] = *inputs.ModelSnapshotRetentionDays
	}
	if inputs.DailyModelSnapshotRetentionAfterDays != nil {
		body["daily_model_snapshot_retention_after_days"] = *inputs.DailyModelSnapshotRetentionAfterDays
	}
	if inputs.ResultsIndexName != nil {
		body["results_index_name"] = *inputs.ResultsIndexName
	}
	if inputs.AllowLazyOpen != nil {
		body["allow_lazy_open"] = *inputs.AllowLazyOpen
	}
	if inputs.Description != nil {
		body["description"] = *inputs.Description
	}
	if len(inputs.Groups) > 0 {
		body["groups"] = inputs.Groups
	}
	if inputs.CustomSettings != nil {
		var customSettings any
		if err := json.Unmarshal([]byte(*inputs.CustomSettings), &customSettings); err != nil {
			return nil, fmt.Errorf("invalid customSettings JSON: %w", err)
		}
		body["custom_settings"] = customSettings
	}

	return body, nil
}
