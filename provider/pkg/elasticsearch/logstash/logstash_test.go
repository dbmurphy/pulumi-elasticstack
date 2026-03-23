package logstash

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func boolPtr(b bool) *bool    { return &b }

func TestBuildLogstashPipelineBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     PipelineInputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name: "minimal required fields",
			inputs: PipelineInputs{
				PipelineID: "my-pipeline",
				Pipeline:   `input { stdin {} } output { stdout {} }`,
			},
			wantKeys: map[string]any{
				"pipeline": `input { stdin {} } output { stdout {} }`,
			},
			absentKeys: []string{
				"description", "pipeline.batch.delay", "pipeline.batch.size",
				"pipeline.ecs_compatibility", "pipeline_metadata",
				"pipeline.plugin_classloaders", "pipeline.workers",
				"queue.checkpoint.writes", "queue.drain", "queue.max_bytes",
				"queue.max_events", "queue.type", "username",
			},
		},
		{
			name: "with description",
			inputs: PipelineInputs{
				PipelineID:  "desc-pipe",
				Pipeline:    "input {}",
				Description: strPtr("A test pipeline"),
			},
			wantKeys: map[string]any{
				"pipeline":    "input {}",
				"description": "A test pipeline",
			},
		},
		{
			name: "with batch settings",
			inputs: PipelineInputs{
				PipelineID:         "batch-pipe",
				Pipeline:           "input {}",
				PipelineBatchDelay: intPtr(50),
				PipelineBatchSize:  intPtr(125),
			},
			wantKeys: map[string]any{
				"pipeline":             "input {}",
				"pipeline.batch.delay": float64(50),
				"pipeline.batch.size":  float64(125),
			},
		},
		{
			name: "with ecs_compatibility",
			inputs: PipelineInputs{
				PipelineID:               "ecs-pipe",
				Pipeline:                 "input {}",
				PipelineEcsCompatibility: strPtr("v8"),
			},
			wantKeys: map[string]any{
				"pipeline":                   "input {}",
				"pipeline.ecs_compatibility": "v8",
			},
		},
		{
			name: "with pipeline_metadata JSON",
			inputs: PipelineInputs{
				PipelineID:       "meta-pipe",
				Pipeline:         "input {}",
				PipelineMetadata: strPtr(`{"version":3}`),
			},
			wantKeys: map[string]any{
				"pipeline":          "input {}",
				"pipeline_metadata": map[string]any{"version": float64(3)},
			},
		},
		{
			name: "with plugin_classloaders",
			inputs: PipelineInputs{
				PipelineID:                 "classloader-pipe",
				Pipeline:                   "input {}",
				PipelinePluginClassloaders: boolPtr(true),
			},
			wantKeys: map[string]any{
				"pipeline":                     "input {}",
				"pipeline.plugin_classloaders": true,
			},
		},
		{
			name: "with workers",
			inputs: PipelineInputs{
				PipelineID:      "workers-pipe",
				Pipeline:        "input {}",
				PipelineWorkers: intPtr(4),
			},
			wantKeys: map[string]any{
				"pipeline":         "input {}",
				"pipeline.workers": float64(4),
			},
		},
		{
			name: "with queue settings",
			inputs: PipelineInputs{
				PipelineID:            "queue-pipe",
				Pipeline:              "input {}",
				QueueCheckpointWrites: intPtr(1024),
				QueueDrain:            boolPtr(true),
				QueueMaxBytes:         strPtr("1gb"),
				QueueMaxEvents:        intPtr(0),
				QueueType:             strPtr("persisted"),
			},
			wantKeys: map[string]any{
				"pipeline":                "input {}",
				"queue.checkpoint.writes": float64(1024),
				"queue.drain":             true,
				"queue.max_bytes":         "1gb",
				"queue.max_events":        float64(0),
				"queue.type":              "persisted",
			},
		},
		{
			name: "with username",
			inputs: PipelineInputs{
				PipelineID: "user-pipe",
				Pipeline:   "input {}",
				Username:   strPtr("logstash_admin"),
			},
			wantKeys: map[string]any{
				"pipeline": "input {}",
				"username": "logstash_admin",
			},
		},
		{
			name: "all fields set",
			inputs: PipelineInputs{
				PipelineID:                 "full-pipe",
				Pipeline:                   "input { beats { port => 5044 } }",
				Description:                strPtr("Full pipeline"),
				PipelineBatchDelay:         intPtr(100),
				PipelineBatchSize:          intPtr(250),
				PipelineEcsCompatibility:   strPtr("disabled"),
				PipelineMetadata:           strPtr(`{"env":"prod"}`),
				PipelinePluginClassloaders: boolPtr(false),
				PipelineWorkers:            intPtr(8),
				QueueCheckpointWrites:      intPtr(2048),
				QueueDrain:                 boolPtr(false),
				QueueMaxBytes:              strPtr("2gb"),
				QueueMaxEvents:             intPtr(5000),
				QueueType:                  strPtr("memory"),
				Username:                   strPtr("admin"),
			},
			wantKeys: map[string]any{
				"pipeline":                     "input { beats { port => 5044 } }",
				"description":                  "Full pipeline",
				"pipeline.batch.delay":         float64(100),
				"pipeline.batch.size":          float64(250),
				"pipeline.ecs_compatibility":   "disabled",
				"pipeline_metadata":            map[string]any{"env": "prod"},
				"pipeline.plugin_classloaders": false,
				"pipeline.workers":             float64(8),
				"queue.checkpoint.writes":      float64(2048),
				"queue.drain":                  false,
				"queue.max_bytes":              "2gb",
				"queue.max_events":             float64(5000),
				"queue.type":                   "memory",
				"username":                     "admin",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildLogstashPipelineBody(tc.inputs)
			bodyJSON, _ := json.Marshal(body)
			wantJSON, _ := json.Marshal(tc.wantKeys)
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

			for _, key := range tc.absentKeys {
				if _, ok := body[key]; ok {
					t.Errorf("unexpected key %q in body", key)
				}
			}
		})
	}
}
