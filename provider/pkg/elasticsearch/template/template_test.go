package template

import (
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestBuildIndexTemplateBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs IndexTemplateInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal with index patterns only",
			inputs: IndexTemplateInputs{
				Name:          "my-template",
				IndexPatterns: []string{"logs-*"},
			},
			check: func(t *testing.T, body map[string]any) {
				patterns, ok := body["index_patterns"].([]string)
				if !ok {
					t.Fatal("expected index_patterns to be []string")
				}
				if len(patterns) != 1 || patterns[0] != "logs-*" {
					t.Errorf("index_patterns = %v, want [logs-*]", patterns)
				}
				if _, ok := body["template"]; ok {
					t.Error("expected no template key")
				}
				if _, ok := body["composed_of"]; ok {
					t.Error("expected no composed_of key")
				}
				if _, ok := body["priority"]; ok {
					t.Error("expected no priority key")
				}
			},
		},
		{
			name: "with template JSON",
			inputs: IndexTemplateInputs{
				Name:          "my-template",
				IndexPatterns: []string{"logs-*"},
				Template:      strPtr(`{"settings":{"number_of_replicas":1}}`),
			},
			check: func(t *testing.T, body map[string]any) {
				tmpl, ok := body["template"]
				if !ok {
					t.Fatal("expected template key")
				}
				tmplMap, ok := tmpl.(map[string]any)
				if !ok {
					t.Fatal("expected template to be a map")
				}
				if _, ok := tmplMap["settings"]; !ok {
					t.Error("expected settings in template")
				}
			},
		},
		{
			name: "with priority",
			inputs: IndexTemplateInputs{
				Name:          "my-template",
				IndexPatterns: []string{"logs-*"},
				Priority:      intPtr(100),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["priority"] != 100 {
					t.Errorf("priority = %v, want 100", body["priority"])
				}
			},
		},
		{
			name: "with composed_of",
			inputs: IndexTemplateInputs{
				Name:          "my-template",
				IndexPatterns: []string{"logs-*"},
				ComposedOf:    []string{"component-a", "component-b"},
			},
			check: func(t *testing.T, body map[string]any) {
				composed, ok := body["composed_of"].([]string)
				if !ok {
					t.Fatal("expected composed_of to be []string")
				}
				if len(composed) != 2 || composed[0] != "component-a" || composed[1] != "component-b" {
					t.Errorf("composed_of = %v, want [component-a component-b]", composed)
				}
			},
		},
		{
			name: "with data stream flag",
			inputs: IndexTemplateInputs{
				Name:          "my-template",
				IndexPatterns: []string{"logs-*"},
				DataStream:    strPtr(`{}`),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["data_stream"] == nil {
					t.Error("expected data_stream to be present")
				}
			},
		},
		{
			name: "with version",
			inputs: IndexTemplateInputs{
				Name:          "my-template",
				IndexPatterns: []string{"logs-*"},
				Version:       intPtr(3),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["version"] != 3 {
					t.Errorf("version = %v, want 3", body["version"])
				}
			},
		},
		{
			name: "with meta",
			inputs: IndexTemplateInputs{
				Name:          "my-template",
				IndexPatterns: []string{"logs-*"},
				Meta:          strPtr(`{"managed_by":"pulumi"}`),
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
			inputs: IndexTemplateInputs{
				Name:          "my-template",
				IndexPatterns: []string{"logs-*", "metrics-*"},
				ComposedOf:    []string{"comp-1"},
				DataStream:    strPtr(`{}`),
				Template:      strPtr(`{"settings":{"number_of_shards":1}}`),
				Priority:      intPtr(200),
				Version:       intPtr(1),
				Meta:          strPtr(`{"owner":"team-a"}`),
			},
			check: func(t *testing.T, body map[string]any) {
				patterns := body["index_patterns"].([]string)
				if len(patterns) != 2 {
					t.Errorf("expected 2 index patterns, got %d", len(patterns))
				}
				if body["priority"] != 200 {
					t.Errorf("priority = %v, want 200", body["priority"])
				}
				if body["version"] != 1 {
					t.Errorf("version = %v, want 1", body["version"])
				}
				if body["data_stream"] == nil {
					t.Error("expected data_stream")
				}
				if body["template"] == nil {
					t.Error("expected template")
				}
				if body["_meta"] == nil {
					t.Error("expected _meta")
				}
				if body["composed_of"] == nil {
					t.Error("expected composed_of")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildIndexTemplateBody(tt.inputs)
			tt.check(t, body)
		})
	}
}

func TestBuildComponentTemplateBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs ComponentTemplateInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "with template only",
			inputs: ComponentTemplateInputs{
				Name:     "my-component",
				Template: `{"settings":{"number_of_replicas":1}}`,
			},
			check: func(t *testing.T, body map[string]any) {
				tmpl, ok := body["template"]
				if !ok {
					t.Fatal("expected template key")
				}
				tmplMap, ok := tmpl.(map[string]any)
				if !ok {
					t.Fatal("expected template to be a map")
				}
				if _, ok := tmplMap["settings"]; !ok {
					t.Error("expected settings in template")
				}
				if _, ok := body["version"]; ok {
					t.Error("expected no version key")
				}
				if _, ok := body["_meta"]; ok {
					t.Error("expected no _meta key")
				}
			},
		},
		{
			name: "with version",
			inputs: ComponentTemplateInputs{
				Name:     "my-component",
				Template: `{"mappings":{}}`,
				Version:  intPtr(5),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["version"] != 5 {
					t.Errorf("version = %v, want 5", body["version"])
				}
			},
		},
		{
			name: "with meta",
			inputs: ComponentTemplateInputs{
				Name:     "my-component",
				Template: `{"mappings":{}}`,
				Meta:     strPtr(`{"managed_by":"pulumi"}`),
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
			inputs: ComponentTemplateInputs{
				Name:     "my-component",
				Template: `{"settings":{"number_of_shards":2},"mappings":{"properties":{"field1":{"type":"keyword"}}}}`,
				Version:  intPtr(1),
				Meta:     strPtr(`{"description":"test component"}`),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["template"] == nil {
					t.Error("expected template")
				}
				if body["version"] != 1 {
					t.Errorf("version = %v, want 1", body["version"])
				}
				if body["_meta"] == nil {
					t.Error("expected _meta")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildComponentTemplateBody(tt.inputs)
			tt.check(t, body)
		})
	}
}
