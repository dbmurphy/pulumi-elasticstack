package snapshot

import (
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestBuildRepoBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs RepositoryInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "with fs type and settings",
			inputs: RepositoryInputs{
				Name:     "my-repo",
				Type:     "fs",
				Settings: `{"location":"/mnt/snapshots"}`,
			},
			check: func(t *testing.T, body map[string]any) {
				if body["type"] != "fs" {
					t.Errorf("type = %v, want fs", body["type"])
				}
				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatal("expected settings to be a map")
				}
				if settings["location"] != "/mnt/snapshots" {
					t.Errorf("settings.location = %v, want /mnt/snapshots", settings["location"])
				}
			},
		},
		{
			name: "with s3 type and settings",
			inputs: RepositoryInputs{
				Name:     "s3-repo",
				Type:     "s3",
				Settings: `{"bucket":"my-bucket","region":"us-east-1"}`,
			},
			check: func(t *testing.T, body map[string]any) {
				if body["type"] != "s3" {
					t.Errorf("type = %v, want s3", body["type"])
				}
				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatal("expected settings to be a map")
				}
				if settings["bucket"] != "my-bucket" {
					t.Errorf("settings.bucket = %v, want my-bucket", settings["bucket"])
				}
				if settings["region"] != "us-east-1" {
					t.Errorf("settings.region = %v, want us-east-1", settings["region"])
				}
			},
		},
		{
			name: "with empty settings JSON object",
			inputs: RepositoryInputs{
				Name:     "minimal-repo",
				Type:     "url",
				Settings: `{}`,
			},
			check: func(t *testing.T, body map[string]any) {
				if body["type"] != "url" {
					t.Errorf("type = %v, want url", body["type"])
				}
				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatal("expected settings to be a map")
				}
				if len(settings) != 0 {
					t.Errorf("expected empty settings, got %v", settings)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildRepoBody(tt.inputs)
			tt.check(t, body)
		})
	}
}

func TestBuildSLMBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs LifecycleInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal required fields",
			inputs: LifecycleInputs{
				Name:         "daily-snap",
				Schedule:     "0 30 1 * * ?",
				SnapshotName: "<daily-snap-{now/d}>",
				Repository:   "my-repo",
			},
			check: func(t *testing.T, body map[string]any) {
				if body["schedule"] != "0 30 1 * * ?" {
					t.Errorf("schedule = %v, want '0 30 1 * * ?'", body["schedule"])
				}
				if body["name"] != "<daily-snap-{now/d}>" {
					t.Errorf("name = %v, want '<daily-snap-{now/d}>'", body["name"])
				}
				if body["repository"] != "my-repo" {
					t.Errorf("repository = %v, want my-repo", body["repository"])
				}
				if _, ok := body["config"]; ok {
					t.Error("expected no config key for minimal inputs")
				}
				if _, ok := body["retention"]; ok {
					t.Error("expected no retention key for minimal inputs")
				}
			},
		},
		{
			name: "with indices and config options",
			inputs: LifecycleInputs{
				Name:               "daily-snap",
				Schedule:           "0 30 1 * * ?",
				SnapshotName:       "<daily-snap-{now/d}>",
				Repository:         "my-repo",
				Indices:            []string{"logs-*", "metrics-*"},
				IgnoreUnavailable:  boolPtr(true),
				IncludeGlobalState: boolPtr(false),
				Partial:            boolPtr(true),
			},
			check: func(t *testing.T, body map[string]any) {
				config, ok := body["config"].(map[string]any)
				if !ok {
					t.Fatal("expected config to be a map")
				}
				indices, ok := config["indices"].([]string)
				if !ok {
					t.Fatal("expected indices to be []string")
				}
				if len(indices) != 2 {
					t.Errorf("expected 2 indices, got %d", len(indices))
				}
				if config["ignore_unavailable"] != true {
					t.Errorf("ignore_unavailable = %v, want true", config["ignore_unavailable"])
				}
				if config["include_global_state"] != false {
					t.Errorf("include_global_state = %v, want false", config["include_global_state"])
				}
				if config["partial"] != true {
					t.Errorf("partial = %v, want true", config["partial"])
				}
			},
		},
		{
			name: "with retention settings",
			inputs: LifecycleInputs{
				Name:         "daily-snap",
				Schedule:     "0 30 1 * * ?",
				SnapshotName: "<daily-snap-{now/d}>",
				Repository:   "my-repo",
				ExpireAfter:  strPtr("30d"),
				MaxCount:     intPtr(50),
				MinCount:     intPtr(5),
			},
			check: func(t *testing.T, body map[string]any) {
				retention, ok := body["retention"].(map[string]any)
				if !ok {
					t.Fatal("expected retention to be a map")
				}
				if retention["expire_after"] != "30d" {
					t.Errorf("expire_after = %v, want 30d", retention["expire_after"])
				}
				if retention["max_count"] != 50 {
					t.Errorf("max_count = %v, want 50", retention["max_count"])
				}
				if retention["min_count"] != 5 {
					t.Errorf("min_count = %v, want 5", retention["min_count"])
				}
			},
		},
		{
			name: "with feature states",
			inputs: LifecycleInputs{
				Name:          "daily-snap",
				Schedule:      "0 30 1 * * ?",
				SnapshotName:  "<daily-snap-{now/d}>",
				Repository:    "my-repo",
				FeatureStates: []string{"geoip", "security"},
			},
			check: func(t *testing.T, body map[string]any) {
				config, ok := body["config"].(map[string]any)
				if !ok {
					t.Fatal("expected config to be a map")
				}
				features, ok := config["feature_states"].([]string)
				if !ok {
					t.Fatal("expected feature_states to be []string")
				}
				if len(features) != 2 {
					t.Errorf("expected 2 feature states, got %d", len(features))
				}
			},
		},
		{
			name: "with all fields",
			inputs: LifecycleInputs{
				Name:               "full-snap",
				Schedule:           "0 0 * * * ?",
				SnapshotName:       "<hourly-{now/H}>",
				Repository:         "s3-repo",
				Indices:            []string{"*"},
				ExpireAfter:        strPtr("7d"),
				MaxCount:           intPtr(168),
				MinCount:           intPtr(24),
				IgnoreUnavailable:  boolPtr(true),
				IncludeGlobalState: boolPtr(true),
				Partial:            boolPtr(false),
				FeatureStates:      []string{"geoip"},
			},
			check: func(t *testing.T, body map[string]any) {
				if body["schedule"] == nil {
					t.Error("expected schedule")
				}
				if body["name"] == nil {
					t.Error("expected name")
				}
				if body["repository"] == nil {
					t.Error("expected repository")
				}
				if body["config"] == nil {
					t.Error("expected config")
				}
				if body["retention"] == nil {
					t.Error("expected retention")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildSLMBody(tt.inputs)
			tt.check(t, body)
		})
	}
}
