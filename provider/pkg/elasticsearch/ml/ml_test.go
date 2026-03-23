package ml

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func boolPtr(b bool) *bool    { return &b }

func TestBuildAnomalyDetectionJobBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     AnomalyDetectionJobInputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name: "minimal required fields",
			inputs: AnomalyDetectionJobInputs{
				JobId:           "job-1",
				AnalysisConfig:  `{"detectors":[{"function":"mean","field_name":"response_time"}]}`,
				DataDescription: `{"time_field":"timestamp"}`,
			},
			wantKeys: map[string]any{
				"analysis_config": map[string]any{
					"detectors": []any{map[string]any{"function": "mean", "field_name": "response_time"}},
				},
				"data_description": map[string]any{"time_field": "timestamp"},
			},
			absentKeys: []string{
				"analysis_limits",
				"model_snapshot_retention_days",
				"daily_model_snapshot_retention_after_days",
				"results_index_name",
				"allow_lazy_open",
				"description",
				"groups",
				"custom_settings",
			},
		},
		{
			name: "with analysis_limits JSON",
			inputs: AnomalyDetectionJobInputs{
				JobId:           "job-limits",
				AnalysisConfig:  `{"detectors":[]}`,
				DataDescription: `{"time_field":"ts"}`,
				AnalysisLimits:  strPtr(`{"model_memory_limit":"256mb"}`),
			},
			wantKeys: map[string]any{
				"analysis_limits": map[string]any{"model_memory_limit": "256mb"},
			},
		},
		{
			name: "with snapshot retention days",
			inputs: AnomalyDetectionJobInputs{
				JobId:                                "job-retention",
				AnalysisConfig:                       `{"detectors":[]}`,
				DataDescription:                      `{"time_field":"ts"}`,
				ModelSnapshotRetentionDays:           intPtr(10),
				DailyModelSnapshotRetentionAfterDays: intPtr(5),
			},
			wantKeys: map[string]any{
				"model_snapshot_retention_days":             float64(10),
				"daily_model_snapshot_retention_after_days": float64(5),
			},
		},
		{
			name: "with results_index_name",
			inputs: AnomalyDetectionJobInputs{
				JobId:            "job-idx",
				AnalysisConfig:   `{"detectors":[]}`,
				DataDescription:  `{"time_field":"ts"}`,
				ResultsIndexName: strPtr("custom_results"),
			},
			wantKeys: map[string]any{
				"results_index_name": "custom_results",
			},
		},
		{
			name: "with allow_lazy_open",
			inputs: AnomalyDetectionJobInputs{
				JobId:           "job-lazy",
				AnalysisConfig:  `{"detectors":[]}`,
				DataDescription: `{"time_field":"ts"}`,
				AllowLazyOpen:   boolPtr(true),
			},
			wantKeys: map[string]any{
				"allow_lazy_open": true,
			},
		},
		{
			name: "with description",
			inputs: AnomalyDetectionJobInputs{
				JobId:           "job-desc",
				AnalysisConfig:  `{"detectors":[]}`,
				DataDescription: `{"time_field":"ts"}`,
				Description:     strPtr("A test job"),
			},
			wantKeys: map[string]any{
				"description": "A test job",
			},
		},
		{
			name: "with groups",
			inputs: AnomalyDetectionJobInputs{
				JobId:           "job-groups",
				AnalysisConfig:  `{"detectors":[]}`,
				DataDescription: `{"time_field":"ts"}`,
				Groups:          []string{"group-a", "group-b"},
			},
			wantKeys: map[string]any{
				"groups": []any{"group-a", "group-b"},
			},
		},
		{
			name: "with custom_settings JSON",
			inputs: AnomalyDetectionJobInputs{
				JobId:           "job-custom",
				AnalysisConfig:  `{"detectors":[]}`,
				DataDescription: `{"time_field":"ts"}`,
				CustomSettings:  strPtr(`{"custom_key":"custom_value"}`),
			},
			wantKeys: map[string]any{
				"custom_settings": map[string]any{"custom_key": "custom_value"},
			},
		},
		{
			name: "empty groups not included",
			inputs: AnomalyDetectionJobInputs{
				JobId:           "job-no-groups",
				AnalysisConfig:  `{"detectors":[]}`,
				DataDescription: `{"time_field":"ts"}`,
				Groups:          []string{},
			},
			absentKeys: []string{"groups"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildAnomalyDetectionJobBody(tc.inputs)
			if err != nil {
				t.Fatal(err)
			}
			assertBodyKeys(t, body, tc.wantKeys, tc.absentKeys)
		})
	}
}

func TestBuildDatafeedBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     DatafeedInputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name: "minimal required fields",
			inputs: DatafeedInputs{
				DatafeedId: "df-1",
				JobId:      "job-1",
				Indices:    []string{"logs-*"},
			},
			wantKeys: map[string]any{
				"job_id":  "job-1",
				"indices": []any{"logs-*"},
			},
			absentKeys: []string{
				"query",
				"frequency",
				"query_delay",
				"max_empty_searches",
				"scroll_size",
				"chunking_config",
				"delayed_data_check_config",
				"indices_options",
				"runtime_mappings",
				"script_fields",
			},
		},
		{
			name: "with query JSON",
			inputs: DatafeedInputs{
				DatafeedId: "df-q",
				JobId:      "job-q",
				Indices:    []string{"idx"},
				Query:      strPtr(`{"match_all":{}}`),
			},
			wantKeys: map[string]any{
				"query": map[string]any{"match_all": map[string]any{}},
			},
		},
		{
			name: "with frequency and query_delay",
			inputs: DatafeedInputs{
				DatafeedId: "df-freq",
				JobId:      "job-freq",
				Indices:    []string{"idx"},
				Frequency:  strPtr("60s"),
				QueryDelay: strPtr("30s"),
			},
			wantKeys: map[string]any{
				"frequency":   "60s",
				"query_delay": "30s",
			},
		},
		{
			name: "with max_empty_searches and scroll_size",
			inputs: DatafeedInputs{
				DatafeedId:       "df-scroll",
				JobId:            "job-scroll",
				Indices:          []string{"idx"},
				MaxEmptySearches: intPtr(10),
				ScrollSize:       intPtr(1000),
			},
			wantKeys: map[string]any{
				"max_empty_searches": float64(10),
				"scroll_size":        float64(1000),
			},
		},
		{
			name: "with chunking_config JSON",
			inputs: DatafeedInputs{
				DatafeedId:     "df-chunk",
				JobId:          "job-chunk",
				Indices:        []string{"idx"},
				ChunkingConfig: strPtr(`{"mode":"auto"}`),
			},
			wantKeys: map[string]any{
				"chunking_config": map[string]any{"mode": "auto"},
			},
		},
		{
			name: "with delayed_data_check_config JSON",
			inputs: DatafeedInputs{
				DatafeedId:             "df-delay",
				JobId:                  "job-delay",
				Indices:                []string{"idx"},
				DelayedDataCheckConfig: strPtr(`{"enabled":true}`),
			},
			wantKeys: map[string]any{
				"delayed_data_check_config": map[string]any{"enabled": true},
			},
		},
		{
			name: "with indices_options JSON",
			inputs: DatafeedInputs{
				DatafeedId:     "df-iopt",
				JobId:          "job-iopt",
				Indices:        []string{"idx"},
				IndicesOptions: strPtr(`{"allow_no_indices":true}`),
			},
			wantKeys: map[string]any{
				"indices_options": map[string]any{"allow_no_indices": true},
			},
		},
		{
			name: "with runtime_mappings JSON",
			inputs: DatafeedInputs{
				DatafeedId:      "df-rt",
				JobId:           "job-rt",
				Indices:         []string{"idx"},
				RuntimeMappings: strPtr(`{"day_of_week":{"type":"keyword"}}`),
			},
			wantKeys: map[string]any{
				"runtime_mappings": map[string]any{"day_of_week": map[string]any{"type": "keyword"}},
			},
		},
		{
			name: "with script_fields JSON",
			inputs: DatafeedInputs{
				DatafeedId:   "df-sf",
				JobId:        "job-sf",
				Indices:      []string{"idx"},
				ScriptFields: strPtr(`{"hour":{"script":{"source":"doc['timestamp'].value.getHour()"}}}`),
			},
			wantKeys: map[string]any{
				"script_fields": map[string]any{
					"hour": map[string]any{"script": map[string]any{"source": "doc['timestamp'].value.getHour()"}},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildDatafeedBody(tc.inputs)
			if err != nil {
				t.Fatal(err)
			}
			assertBodyKeys(t, body, tc.wantKeys, tc.absentKeys)
		})
	}
}

func assertBodyKeys(t *testing.T, body map[string]any, wantKeys map[string]any, absentKeys []string) {
	t.Helper()
	bodyJSON, _ := json.Marshal(body)
	wantJSON, _ := json.Marshal(wantKeys)
	var bodyMap, wantMap map[string]any
	if err := json.Unmarshal(bodyJSON, &bodyMap); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wantJSON, &wantMap); err != nil {
		t.Fatal(err)
	}

	for key, wantVal := range wantMap {
		gotVal, ok := bodyMap[key]
		if !ok {
			t.Errorf("missing key %q", key)
			continue
		}
		gotBytes, _ := json.Marshal(gotVal)
		wantBytes, _ := json.Marshal(wantVal)
		if string(gotBytes) != string(wantBytes) {
			t.Errorf("key %q: got %s, want %s", key, gotBytes, wantBytes)
		}
	}

	for _, key := range absentKeys {
		if _, ok := body[key]; ok {
			t.Errorf("unexpected key %q in body", key)
		}
	}
}
