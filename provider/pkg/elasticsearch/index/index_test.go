package index

import (
	"encoding/json"
	"reflect"
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestBuildIndexBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs Inputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name:   "minimal inputs produces empty body",
			inputs: Inputs{Name: "test-index"},
			check: func(t *testing.T, body map[string]any) {
				if _, ok := body["settings"]; ok {
					t.Error("expected no settings key for minimal inputs")
				}
				if _, ok := body["mappings"]; ok {
					t.Error("expected no mappings key for minimal inputs")
				}
				if _, ok := body["aliases"]; ok {
					t.Error("expected no aliases key for minimal inputs")
				}
			},
		},
		{
			name: "with numberOfShards and numberOfReplicas",
			inputs: Inputs{
				Name:             "test-index",
				NumberOfShards:   intPtr(3),
				NumberOfReplicas: intPtr(1),
			},
			check: func(t *testing.T, body map[string]any) {
				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatal("expected settings to be present")
				}
				if settings["number_of_shards"] != 3 {
					t.Errorf("number_of_shards = %v, want 3", settings["number_of_shards"])
				}
				if settings["number_of_replicas"] != 1 {
					t.Errorf("number_of_replicas = %v, want 1", settings["number_of_replicas"])
				}
			},
		},
		{
			name: "with settings JSON merged",
			inputs: Inputs{
				Name:     "test-index",
				Settings: strPtr(`{"refresh_interval":"5s","number_of_replicas":2}`),
			},
			check: func(t *testing.T, body map[string]any) {
				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatal("expected settings to be present")
				}
				if settings["refresh_interval"] != "5s" {
					t.Errorf("refresh_interval = %v, want 5s", settings["refresh_interval"])
				}
				// JSON numbers unmarshal as float64
				if settings["number_of_replicas"] != float64(2) {
					t.Errorf("number_of_replicas = %v, want 2", settings["number_of_replicas"])
				}
			},
		},
		{
			name: "with mappings JSON",
			inputs: Inputs{
				Name:     "test-index",
				Mappings: strPtr(`{"properties":{"title":{"type":"text"}}}`),
			},
			check: func(t *testing.T, body map[string]any) {
				mappings, ok := body["mappings"]
				if !ok {
					t.Fatal("expected mappings to be present")
				}
				m, ok := mappings.(map[string]any)
				if !ok {
					t.Fatal("expected mappings to be a map")
				}
				if _, ok := m["properties"]; !ok {
					t.Error("expected properties in mappings")
				}
			},
		},
		{
			name: "with aliases",
			inputs: Inputs{
				Name: "test-index",
				Aliases: []Alias{
					{
						Name:    "my-alias",
						Routing: strPtr("shard-1"),
					},
				},
			},
			check: func(t *testing.T, body map[string]any) {
				aliases, ok := body["aliases"].(map[string]any)
				if !ok {
					t.Fatal("expected aliases to be present")
				}
				aliasBody, ok := aliases["my-alias"].(map[string]any)
				if !ok {
					t.Fatal("expected my-alias to be present")
				}
				if aliasBody["routing"] != "shard-1" {
					t.Errorf("routing = %v, want shard-1", aliasBody["routing"])
				}
			},
		},
		{
			name: "with codec and hidden",
			inputs: Inputs{
				Name:   "test-index",
				Codec:  strPtr("best_compression"),
				Hidden: boolPtr(true),
			},
			check: func(t *testing.T, body map[string]any) {
				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatal("expected settings to be present")
				}
				if settings["codec"] != "best_compression" {
					t.Errorf("codec = %v, want best_compression", settings["codec"])
				}
				if settings["hidden"] != true {
					t.Errorf("hidden = %v, want true", settings["hidden"])
				}
			},
		},
		{
			name: "with sort fields",
			inputs: Inputs{
				Name:      "test-index",
				SortField: []string{"timestamp"},
				SortOrder: []string{"desc"},
			},
			check: func(t *testing.T, body map[string]any) {
				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatal("expected settings to be present")
				}
				sf, ok := settings["sort.field"].([]string)
				if !ok {
					t.Fatal("expected sort.field to be []string")
				}
				if len(sf) != 1 || sf[0] != "timestamp" {
					t.Errorf("sort.field = %v, want [timestamp]", sf)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildIndexBody(tt.inputs)
			tt.check(t, body)
		})
	}
}

func TestBuildMutableSettings(t *testing.T) {
	tests := []struct {
		name   string
		inputs Inputs
		want   map[string]any
	}{
		{
			name:   "empty inputs",
			inputs: Inputs{Name: "test"},
			want:   map[string]any{},
		},
		{
			name:   "with numberOfReplicas",
			inputs: Inputs{Name: "test", NumberOfReplicas: intPtr(2)},
			want:   map[string]any{"number_of_replicas": 2},
		},
		{
			name:   "with settings JSON",
			inputs: Inputs{Name: "test", Settings: strPtr(`{"refresh_interval":"10s"}`)},
			want:   map[string]any{"refresh_interval": "10s"},
		},
		{
			name: "with numberOfReplicas and settings JSON",
			inputs: Inputs{
				Name:             "test",
				NumberOfReplicas: intPtr(3),
				Settings:         strPtr(`{"refresh_interval":"30s"}`),
			},
			want: map[string]any{
				"number_of_replicas": 3,
				"refresh_interval":   "30s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMutableSettings(tt.inputs)
			// Compare by marshalling to JSON to handle type differences
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			var gotParsed, wantParsed any
			if err := json.Unmarshal(gotJSON, &gotParsed); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(wantJSON, &wantParsed); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotParsed, wantParsed) {
				t.Errorf("buildMutableSettings() = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestBoolPtr(t *testing.T) {
	tests := []struct {
		name  string
		input bool
	}{
		{name: "true", input: true},
		{name: "false", input: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boolPtr(tt.input)
			if got == nil {
				t.Fatal("boolPtr returned nil")
			}
			if *got != tt.input {
				t.Errorf("boolPtr(%v) = %v, want %v", tt.input, *got, tt.input)
			}
		})
	}
}

func TestBuildAliasActions(t *testing.T) {
	tests := []struct {
		name   string
		action string
		inputs AliasInputs
		check  func(t *testing.T, actions []any)
	}{
		{
			name:   "add action with routing and filter",
			action: "add",
			inputs: AliasInputs{
				Name:    "my-alias",
				Indices: []string{"index-1"},
				Routing: strPtr("shard-1"),
				Filter:  strPtr(`{"term":{"status":"published"}}`),
			},
			check: func(t *testing.T, actions []any) {
				if len(actions) != 1 {
					t.Fatalf("expected 1 action, got %d", len(actions))
				}
				action, ok := actions[0].(map[string]any)
				if !ok {
					t.Fatal("expected action to be a map")
				}
				addBody, ok := action["add"].(map[string]any)
				if !ok {
					t.Fatal("expected add key in action")
				}
				if addBody["index"] != "index-1" {
					t.Errorf("index = %v, want index-1", addBody["index"])
				}
				if addBody["alias"] != "my-alias" {
					t.Errorf("alias = %v, want my-alias", addBody["alias"])
				}
				if addBody["routing"] != "shard-1" {
					t.Errorf("routing = %v, want shard-1", addBody["routing"])
				}
				if addBody["filter"] == nil {
					t.Error("expected filter to be present")
				}
			},
		},
		{
			name:   "remove action",
			action: "remove",
			inputs: AliasInputs{
				Name:    "my-alias",
				Indices: []string{"index-1", "index-2"},
			},
			check: func(t *testing.T, actions []any) {
				if len(actions) != 2 {
					t.Fatalf("expected 2 actions, got %d", len(actions))
				}
				for i, a := range actions {
					action, ok := a.(map[string]any)
					if !ok {
						t.Fatalf("action[%d] is not a map", i)
					}
					removeBody, ok := action["remove"].(map[string]any)
					if !ok {
						t.Fatalf("action[%d] missing remove key", i)
					}
					if removeBody["alias"] != "my-alias" {
						t.Errorf("action[%d] alias = %v, want my-alias", i, removeBody["alias"])
					}
				}
			},
		},
		{
			name:   "add action with isWriteIndex and isHidden",
			action: "add",
			inputs: AliasInputs{
				Name:         "my-alias",
				Indices:      []string{"index-1"},
				IsWriteIndex: boolPtr(true),
				IsHidden:     boolPtr(false),
			},
			check: func(t *testing.T, actions []any) {
				if len(actions) != 1 {
					t.Fatalf("expected 1 action, got %d", len(actions))
				}
				action := actions[0].(map[string]any)
				addBody := action["add"].(map[string]any)
				if addBody["is_write_index"] != true {
					t.Errorf("is_write_index = %v, want true", addBody["is_write_index"])
				}
				if addBody["is_hidden"] != false {
					t.Errorf("is_hidden = %v, want false", addBody["is_hidden"])
				}
			},
		},
		{
			name:   "empty indices produces no actions",
			action: "add",
			inputs: AliasInputs{
				Name:    "my-alias",
				Indices: []string{},
			},
			check: func(t *testing.T, actions []any) {
				if len(actions) != 0 {
					t.Errorf("expected 0 actions, got %d", len(actions))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := buildAliasActions(tt.action, tt.inputs)
			tt.check(t, actions)
		})
	}
}

func TestBuildLifecycleBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs DataStreamLifecycleInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name:   "empty inputs produces lifecycle key",
			inputs: DataStreamLifecycleInputs{Name: "ds-test"},
			check: func(t *testing.T, body map[string]any) {
				lc, ok := body["lifecycle"].(map[string]any)
				if !ok {
					t.Fatal("expected lifecycle key")
				}
				if _, ok := lc["data_retention"]; ok {
					t.Error("expected no data_retention")
				}
			},
		},
		{
			name: "with dataRetention",
			inputs: DataStreamLifecycleInputs{
				Name:          "ds-test",
				DataRetention: strPtr("30d"),
			},
			check: func(t *testing.T, body map[string]any) {
				lc := body["lifecycle"].(map[string]any)
				if lc["data_retention"] != "30d" {
					t.Errorf("data_retention = %v, want 30d", lc["data_retention"])
				}
			},
		},
		{
			name: "with enabled flag",
			inputs: DataStreamLifecycleInputs{
				Name:    "ds-test",
				Enabled: boolPtr(false),
			},
			check: func(t *testing.T, body map[string]any) {
				lc := body["lifecycle"].(map[string]any)
				if lc["enabled"] != false {
					t.Errorf("enabled = %v, want false", lc["enabled"])
				}
			},
		},
		{
			name: "with downsampling rounds",
			inputs: DataStreamLifecycleInputs{
				Name: "ds-test",
				Downsampling: []DownsamplingRound{
					{After: "7d", FixedInterval: "1h"},
					{After: "30d", FixedInterval: "1d"},
				},
			},
			check: func(t *testing.T, body map[string]any) {
				lc := body["lifecycle"].(map[string]any)
				ds, ok := lc["downsampling"].(map[string]any)
				if !ok {
					t.Fatal("expected downsampling key")
				}
				rounds, ok := ds["rounds"].([]map[string]any)
				if !ok {
					t.Fatal("expected rounds to be []map[string]any")
				}
				if len(rounds) != 2 {
					t.Fatalf("expected 2 rounds, got %d", len(rounds))
				}
				if rounds[0]["after"] != "7d" {
					t.Errorf("round[0].after = %v, want 7d", rounds[0]["after"])
				}
				if rounds[0]["fixed_interval"] != "1h" {
					t.Errorf("round[0].fixed_interval = %v, want 1h", rounds[0]["fixed_interval"])
				}
				if rounds[1]["after"] != "30d" {
					t.Errorf("round[1].after = %v, want 30d", rounds[1]["after"])
				}
			},
		},
		{
			name: "with all fields",
			inputs: DataStreamLifecycleInputs{
				Name:          "ds-test",
				DataRetention: strPtr("90d"),
				Enabled:       boolPtr(true),
				Downsampling: []DownsamplingRound{
					{After: "1d", FixedInterval: "10m"},
				},
			},
			check: func(t *testing.T, body map[string]any) {
				lc := body["lifecycle"].(map[string]any)
				if lc["data_retention"] != "90d" {
					t.Errorf("data_retention = %v, want 90d", lc["data_retention"])
				}
				if lc["enabled"] != true {
					t.Errorf("enabled = %v, want true", lc["enabled"])
				}
				if lc["downsampling"] == nil {
					t.Error("expected downsampling to be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildLifecycleBody(tt.inputs)
			tt.check(t, body)
		})
	}
}
