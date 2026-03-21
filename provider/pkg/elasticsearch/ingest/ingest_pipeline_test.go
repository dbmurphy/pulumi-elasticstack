package ingest

import (
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestBuildPipelineBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs PipelineInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name:   "empty inputs produces empty body",
			inputs: PipelineInputs{Name: "my-pipeline"},
			check: func(t *testing.T, body map[string]any) {
				if len(body) != 0 {
					t.Errorf("expected empty body, got %v", body)
				}
			},
		},
		{
			name: "with description",
			inputs: PipelineInputs{
				Name:        "my-pipeline",
				Description: strPtr("A test pipeline"),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["description"] != "A test pipeline" {
					t.Errorf("description = %v, want 'A test pipeline'", body["description"])
				}
			},
		},
		{
			name: "with processors JSON",
			inputs: PipelineInputs{
				Name:       "my-pipeline",
				Processors: strPtr(`[{"set":{"field":"foo","value":"bar"}}]`),
			},
			check: func(t *testing.T, body map[string]any) {
				processors, ok := body["processors"]
				if !ok {
					t.Fatal("expected processors key")
				}
				arr, ok := processors.([]any)
				if !ok {
					t.Fatal("expected processors to be an array")
				}
				if len(arr) != 1 {
					t.Errorf("expected 1 processor, got %d", len(arr))
				}
			},
		},
		{
			name: "with on_failure JSON",
			inputs: PipelineInputs{
				Name:      "my-pipeline",
				OnFailure: strPtr(`[{"set":{"field":"error","value":"failed"}}]`),
			},
			check: func(t *testing.T, body map[string]any) {
				onFailure, ok := body["on_failure"]
				if !ok {
					t.Fatal("expected on_failure key")
				}
				arr, ok := onFailure.([]any)
				if !ok {
					t.Fatal("expected on_failure to be an array")
				}
				if len(arr) != 1 {
					t.Errorf("expected 1 on_failure processor, got %d", len(arr))
				}
			},
		},
		{
			name: "with version",
			inputs: PipelineInputs{
				Name:    "my-pipeline",
				Version: intPtr(3),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["version"] != 3 {
					t.Errorf("version = %v, want 3", body["version"])
				}
			},
		},
		{
			name: "with metadata",
			inputs: PipelineInputs{
				Name:     "my-pipeline",
				Metadata: strPtr(`{"managed_by":"pulumi"}`),
			},
			check: func(t *testing.T, body map[string]any) {
				meta, ok := body["_meta"]
				if !ok {
					t.Fatal("expected _meta key")
				}
				metaMap, ok := meta.(map[string]any)
				if !ok {
					t.Fatal("expected _meta to be a map")
				}
				if metaMap["managed_by"] != "pulumi" {
					t.Errorf("_meta.managed_by = %v, want pulumi", metaMap["managed_by"])
				}
			},
		},
		{
			name: "with all fields",
			inputs: PipelineInputs{
				Name:        "my-pipeline",
				Description: strPtr("Full pipeline"),
				Processors:  strPtr(`[{"uppercase":{"field":"name"}}]`),
				OnFailure:   strPtr(`[{"set":{"field":"_index","value":"failed"}}]`),
				Metadata:    strPtr(`{"version":"1.0"}`),
				Version:     intPtr(1),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["description"] != "Full pipeline" {
					t.Errorf("description = %v, want 'Full pipeline'", body["description"])
				}
				if body["processors"] == nil {
					t.Error("expected processors")
				}
				if body["on_failure"] == nil {
					t.Error("expected on_failure")
				}
				if body["_meta"] == nil {
					t.Error("expected _meta")
				}
				if body["version"] != 1 {
					t.Errorf("version = %v, want 1", body["version"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildPipelineBody(tt.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, body)
		})
	}
}
