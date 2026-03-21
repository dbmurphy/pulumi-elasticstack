package slo

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func TestDerefString(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil returns empty", nil, ""},
		{"non-nil returns value", strPtr("space1"), "space1"},
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

func TestBuildSloBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs Inputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - required fields only",
			inputs: Inputs{
				Name:            "My SLO",
				Indicator:       `{"type":"sli.kql.custom","params":{"index":"logs-*","good":"response < 500","total":"*"}}`,
				TimeWindow:      `{"duration":"30d","type":"rolling"}`,
				BudgetingMethod: "occurrences",
				Objective:       `{"target":0.99}`,
			},
			check: func(t *testing.T, body map[string]any) {
				if body["name"] != "My SLO" {
					t.Errorf("name = %v, want My SLO", body["name"])
				}
				if body["budgetingMethod"] != "occurrences" {
					t.Errorf("budgetingMethod = %v, want occurrences", body["budgetingMethod"])
				}

				// indicator should be parsed JSON
				ind, ok := body["indicator"].(map[string]any)
				if !ok {
					t.Fatalf("indicator should be parsed JSON object, got %T", body["indicator"])
				}
				if ind["type"] != "sli.kql.custom" {
					t.Errorf("indicator.type = %v, want sli.kql.custom", ind["type"])
				}

				// timeWindow should be parsed JSON
				tw, ok := body["timeWindow"].(map[string]any)
				if !ok {
					t.Fatalf("timeWindow should be parsed JSON object, got %T", body["timeWindow"])
				}
				if tw["duration"] != "30d" {
					t.Errorf("timeWindow.duration = %v, want 30d", tw["duration"])
				}

				// objective should be parsed JSON
				obj, ok := body["objective"].(map[string]any)
				if !ok {
					t.Fatalf("objective should be parsed JSON object, got %T", body["objective"])
				}
				if obj["target"] != 0.99 {
					t.Errorf("objective.target = %v, want 0.99", obj["target"])
				}

				// Optional fields should not be present
				for _, key := range []string{"description", "settings", "tags", "groupBy"} {
					if _, exists := body[key]; exists {
						t.Errorf("unexpected key %q in minimal body", key)
					}
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: Inputs{
				Name:            "Full SLO",
				Description:     strPtr("A comprehensive SLO"),
				Indicator:       `{"type":"sli.kql.custom","params":{}}`,
				TimeWindow:      `{"duration":"7d","type":"rolling"}`,
				BudgetingMethod: "timeslices",
				Objective:       `{"target":0.95,"timesliceTarget":0.9,"timesliceWindow":"5m"}`,
				Settings:        strPtr(`{"syncDelay":"5m","frequency":"1m"}`),
				Tags:            []string{"production", "critical"},
				GroupBy:         strPtr("service.name"),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["name"] != "Full SLO" {
					t.Errorf("name = %v, want Full SLO", body["name"])
				}
				if body["description"] != "A comprehensive SLO" {
					t.Errorf("description = %v, want A comprehensive SLO", body["description"])
				}
				if body["budgetingMethod"] != "timeslices" {
					t.Errorf("budgetingMethod = %v, want timeslices", body["budgetingMethod"])
				}

				// settings should be parsed JSON
				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatalf("settings should be parsed JSON object, got %T", body["settings"])
				}
				if settings["syncDelay"] != "5m" {
					t.Errorf("settings.syncDelay = %v, want 5m", settings["syncDelay"])
				}

				// tags
				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags should be []string, got %T", body["tags"])
				}
				if len(tags) != 2 || tags[0] != "production" {
					t.Errorf("tags = %v, want [production critical]", tags)
				}

				if body["groupBy"] != "service.name" {
					t.Errorf("groupBy = %v, want service.name", body["groupBy"])
				}
			},
		},
		{
			name: "empty tags not included",
			inputs: Inputs{
				Name:            "No Tags SLO",
				Indicator:       `{}`,
				TimeWindow:      `{}`,
				BudgetingMethod: "occurrences",
				Objective:       `{}`,
				Tags:            []string{},
			},
			check: func(t *testing.T, body map[string]any) {
				if _, exists := body["tags"]; exists {
					t.Error("empty tags should not be included")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildSloBody(tt.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, body)
		})
	}
}
