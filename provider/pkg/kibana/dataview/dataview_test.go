package dataview

import (
	"testing"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestDerefString(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil returns empty", nil, ""},
		{"non-nil returns value", strPtr("hello"), "hello"},
		{"empty string returns empty", strPtr(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derefString(tt.in)
			if got != tt.want {
				t.Errorf("derefString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildDataViewBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs Inputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - title only",
			inputs: Inputs{
				Title: "logs-*",
			},
			check: func(t *testing.T, body map[string]any) {
				dv, ok := body["data_view"].(map[string]any)
				if !ok {
					t.Fatal("expected data_view key in body")
				}
				if dv["title"] != "logs-*" {
					t.Errorf("title = %v, want logs-*", dv["title"])
				}
				// Should not have optional keys
				for _, key := range []string{"name", "timeFieldName", "sourceFilters", "fieldFormats", "fieldAttrs", "runtimeFieldMap", "allowNoIndex", "namespaces"} {
					if _, exists := dv[key]; exists {
						t.Errorf("unexpected key %q in minimal body", key)
					}
				}
				// Override should not be present
				if _, exists := body["override"]; exists {
					t.Error("unexpected override key in body")
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: Inputs{
				Title:         "metrics-*",
				Name:          strPtr("My Metrics View"),
				TimeFieldName: strPtr("@timestamp"),
				SourceFilters: strPtr(`[{"value":"host.name"}]`),
				FieldFormats:  strPtr(`{"bytes":{"id":"bytes"}}`),
				FieldAttrs:    strPtr(`{"host.name":{"customLabel":"Host"}}`),
				RuntimeFieldMap: strPtr(
					`{"day_of_week":{"type":"keyword","script":{"source":"emit(doc['@timestamp'].value.dayOfWeekEnum.getDisplayName(TextStyle.FULL, Locale.ROOT))"}}}`,
				),
				AllowNoIndex: boolPtr(true),
				Namespaces:   []string{"default", "security"},
				Override:     boolPtr(true),
			},
			check: func(t *testing.T, body map[string]any) {
				dv, ok := body["data_view"].(map[string]any)
				if !ok {
					t.Fatal("expected data_view key in body")
				}
				if dv["title"] != "metrics-*" {
					t.Errorf("title = %v, want metrics-*", dv["title"])
				}
				if dv["name"] != "My Metrics View" {
					t.Errorf("name = %v, want My Metrics View", dv["name"])
				}
				if dv["timeFieldName"] != "@timestamp" {
					t.Errorf("timeFieldName = %v, want @timestamp", dv["timeFieldName"])
				}

				// sourceFilters should be parsed JSON
				sf, ok := dv["sourceFilters"].([]any)
				if !ok {
					t.Fatalf("sourceFilters should be a parsed JSON array, got %T", dv["sourceFilters"])
				}
				if len(sf) != 1 {
					t.Errorf("sourceFilters length = %d, want 1", len(sf))
				}

				// fieldFormats should be parsed JSON
				ff, ok := dv["fieldFormats"].(map[string]any)
				if !ok {
					t.Fatalf("fieldFormats should be a parsed JSON object, got %T", dv["fieldFormats"])
				}
				if _, ok := ff["bytes"]; !ok {
					t.Error("fieldFormats missing 'bytes' key")
				}

				// fieldAttrs should be parsed JSON
				fa, ok := dv["fieldAttrs"].(map[string]any)
				if !ok {
					t.Fatalf("fieldAttrs should be a parsed JSON object, got %T", dv["fieldAttrs"])
				}
				if _, ok := fa["host.name"]; !ok {
					t.Error("fieldAttrs missing 'host.name' key")
				}

				// runtimeFieldMap should be parsed JSON
				rfm, ok := dv["runtimeFieldMap"].(map[string]any)
				if !ok {
					t.Fatalf("runtimeFieldMap should be a parsed JSON object, got %T", dv["runtimeFieldMap"])
				}
				if _, ok := rfm["day_of_week"]; !ok {
					t.Error("runtimeFieldMap missing 'day_of_week' key")
				}

				if dv["allowNoIndex"] != true {
					t.Errorf("allowNoIndex = %v, want true", dv["allowNoIndex"])
				}

				ns, ok := dv["namespaces"].([]string)
				if !ok {
					t.Fatalf("namespaces should be []string, got %T", dv["namespaces"])
				}
				if len(ns) != 2 {
					t.Errorf("namespaces length = %d, want 2", len(ns))
				}

				// override should be set at top level
				if body["override"] != true {
					t.Errorf("override = %v, want true", body["override"])
				}
			},
		},
		{
			name: "override false is not included",
			inputs: Inputs{
				Title:    "logs-*",
				Override: boolPtr(false),
			},
			check: func(t *testing.T, body map[string]any) {
				if _, exists := body["override"]; exists {
					t.Error("override=false should not be included in body")
				}
			},
		},
		{
			name: "allowNoIndex false",
			inputs: Inputs{
				Title:        "logs-*",
				AllowNoIndex: boolPtr(false),
			},
			check: func(t *testing.T, body map[string]any) {
				dv := body["data_view"].(map[string]any)
				if dv["allowNoIndex"] != false {
					t.Errorf("allowNoIndex = %v, want false", dv["allowNoIndex"])
				}
			},
		},
		{
			name: "empty namespaces not included",
			inputs: Inputs{
				Title:      "logs-*",
				Namespaces: []string{},
			},
			check: func(t *testing.T, body map[string]any) {
				dv := body["data_view"].(map[string]any)
				if _, exists := dv["namespaces"]; exists {
					t.Error("empty namespaces should not be included")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildDataViewBody(tt.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, body)
		})
	}
}
