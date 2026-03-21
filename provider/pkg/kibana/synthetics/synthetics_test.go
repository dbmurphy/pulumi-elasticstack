package synthetics

import (
	"testing"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

// ---------------------------------------------------------------------------
// resolveSpaceID
// ---------------------------------------------------------------------------

func TestResolveSpaceID(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil returns default", nil, "default"},
		{"empty returns default", strPtr(""), "default"},
		{"custom space", strPtr("observability"), "observability"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSpaceID(tt.in)
			if got != tt.want {
				t.Errorf("resolveSpaceID() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildMonitorBody
// ---------------------------------------------------------------------------

func TestBuildMonitorBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs MonitorInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - required fields only",
			inputs: MonitorInputs{
				Name:        "My HTTP Monitor",
				MonitorType: "http",
				Schedule:    5,
			},
			check: func(t *testing.T, body map[string]any) {
				if body["type"] != "http" {
					t.Errorf("type = %v, want http", body["type"])
				}
				if body["name"] != "My HTTP Monitor" {
					t.Errorf("name = %v, want My HTTP Monitor", body["name"])
				}

				sched, ok := body["schedule"].(map[string]any)
				if !ok {
					t.Fatalf("schedule should be map[string]any, got %T", body["schedule"])
				}
				if sched["number"] != "5" {
					t.Errorf("schedule.number = %v, want 5", sched["number"])
				}
				if sched["unit"] != "m" {
					t.Errorf("schedule.unit = %v, want m", sched["unit"])
				}

				// Optional keys should not be present
				for _, key := range []string{"locations", "private_locations", "enabled", "tags", "alert", "retest_on_failure"} {
					if _, exists := body[key]; exists {
						t.Errorf("unexpected key %q in minimal body", key)
					}
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: MonitorInputs{
				Name:             "Full Monitor",
				MonitorType:      "browser",
				Schedule:         10,
				Locations:        []string{"us_east", "eu_west"},
				PrivateLocations: []string{"my-private-loc"},
				Enabled:          boolPtr(false),
				Tags:             []string{"prod", "critical"},
				Alert:            strPtr(`{"status":{"enabled":true}}`),
				RetestOnFailure:  boolPtr(true),
				Config:           strPtr(`{"urls":"https://example.com","max_redirects":3}`),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["type"] != "browser" {
					t.Errorf("type = %v, want browser", body["type"])
				}

				locs, ok := body["locations"].([]string)
				if !ok {
					t.Fatalf("locations should be []string, got %T", body["locations"])
				}
				if len(locs) != 2 || locs[0] != "us_east" {
					t.Errorf("locations = %v, want [us_east eu_west]", locs)
				}

				privLocs, ok := body["private_locations"].([]string)
				if !ok {
					t.Fatalf("private_locations should be []string, got %T", body["private_locations"])
				}
				if len(privLocs) != 1 || privLocs[0] != "my-private-loc" {
					t.Errorf("private_locations = %v, want [my-private-loc]", privLocs)
				}

				if body["enabled"] != false {
					t.Errorf("enabled = %v, want false", body["enabled"])
				}

				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags should be []string, got %T", body["tags"])
				}
				if len(tags) != 2 {
					t.Errorf("tags length = %d, want 2", len(tags))
				}

				// alert should be parsed JSON
				alert, ok := body["alert"].(map[string]any)
				if !ok {
					t.Fatalf("alert should be parsed JSON object, got %T", body["alert"])
				}
				if _, ok := alert["status"]; !ok {
					t.Error("alert missing 'status' key")
				}

				if body["retest_on_failure"] != true {
					t.Errorf("retest_on_failure = %v, want true", body["retest_on_failure"])
				}

				// Config keys should be merged into body
				if body["urls"] != "https://example.com" {
					t.Errorf("urls = %v, want https://example.com", body["urls"])
				}
				// max_redirects comes from JSON as float64
				if body["max_redirects"] != float64(3) {
					t.Errorf("max_redirects = %v, want 3", body["max_redirects"])
				}
			},
		},
		{
			name: "empty slices not included",
			inputs: MonitorInputs{
				Name:             "No Locations",
				MonitorType:      "tcp",
				Schedule:         1,
				Locations:        []string{},
				PrivateLocations: []string{},
				Tags:             []string{},
			},
			check: func(t *testing.T, body map[string]any) {
				if _, exists := body["locations"]; exists {
					t.Error("empty locations should not be included")
				}
				if _, exists := body["private_locations"]; exists {
					t.Error("empty private_locations should not be included")
				}
				if _, exists := body["tags"]; exists {
					t.Error("empty tags should not be included")
				}
			},
		},
		{
			name: "invalid config JSON is ignored gracefully",
			inputs: MonitorInputs{
				Name:        "Bad Config",
				MonitorType: "http",
				Schedule:    5,
				Config:      strPtr(`not valid json`),
			},
			check: func(t *testing.T, body map[string]any) {
				// Should still have the required fields
				if body["name"] != "Bad Config" {
					t.Errorf("name = %v, want Bad Config", body["name"])
				}
				// No extra config keys should be merged
				if _, exists := body["urls"]; exists {
					t.Error("invalid config should not merge keys")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildMonitorBody(tt.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildParameterBody
// ---------------------------------------------------------------------------

func TestBuildParameterBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs ParameterInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - required fields only",
			inputs: ParameterInputs{
				Key:   "API_KEY",
				Value: "secret123",
			},
			check: func(t *testing.T, body map[string]any) {
				if body["key"] != "API_KEY" {
					t.Errorf("key = %v, want API_KEY", body["key"])
				}
				if body["value"] != "secret123" {
					t.Errorf("value = %v, want secret123", body["value"])
				}
				for _, key := range []string{"description", "tags", "share_across_spaces"} {
					if _, exists := body[key]; exists {
						t.Errorf("unexpected key %q in minimal body", key)
					}
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: ParameterInputs{
				Key:               "BASE_URL",
				Value:             "https://example.com",
				Description:       strPtr("The base URL for tests"),
				Tags:              []string{"env:prod"},
				ShareAcrossSpaces: boolPtr(true),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["key"] != "BASE_URL" {
					t.Errorf("key = %v, want BASE_URL", body["key"])
				}
				if body["description"] != "The base URL for tests" {
					t.Errorf("description = %v, want The base URL for tests", body["description"])
				}

				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags should be []string, got %T", body["tags"])
				}
				if len(tags) != 1 || tags[0] != "env:prod" {
					t.Errorf("tags = %v, want [env:prod]", tags)
				}

				if body["share_across_spaces"] != true {
					t.Errorf("share_across_spaces = %v, want true", body["share_across_spaces"])
				}
			},
		},
		{
			name: "empty tags not included",
			inputs: ParameterInputs{
				Key:   "K",
				Value: "V",
				Tags:  []string{},
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
			body := buildParameterBody(tt.inputs)
			tt.check(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildPrivateLocationBody
// ---------------------------------------------------------------------------

func TestBuildPrivateLocationBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs PrivateLocationInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - required fields only",
			inputs: PrivateLocationInputs{
				Label:         "My Private Location",
				AgentPolicyID: "policy-abc123",
			},
			check: func(t *testing.T, body map[string]any) {
				if body["label"] != "My Private Location" {
					t.Errorf("label = %v, want My Private Location", body["label"])
				}
				if body["agentPolicyId"] != "policy-abc123" {
					t.Errorf("agentPolicyId = %v, want policy-abc123", body["agentPolicyId"])
				}
				for _, key := range []string{"tags", "geo"} {
					if _, exists := body[key]; exists {
						t.Errorf("unexpected key %q in minimal body", key)
					}
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: PrivateLocationInputs{
				Label:         "Full Location",
				AgentPolicyID: "policy-xyz",
				Tags:          []string{"dc:us-east-1"},
				Geo:           strPtr(`{"lat":40.7128,"lon":-74.0060}`),
			},
			check: func(t *testing.T, body map[string]any) {
				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags should be []string, got %T", body["tags"])
				}
				if len(tags) != 1 || tags[0] != "dc:us-east-1" {
					t.Errorf("tags = %v, want [dc:us-east-1]", tags)
				}

				// geo should be parsed JSON
				geo, ok := body["geo"].(map[string]any)
				if !ok {
					t.Fatalf("geo should be parsed JSON object, got %T", body["geo"])
				}
				if geo["lat"] != 40.7128 {
					t.Errorf("geo.lat = %v, want 40.7128", geo["lat"])
				}
				if geo["lon"] != -74.006 {
					t.Errorf("geo.lon = %v, want -74.006", geo["lon"])
				}
			},
		},
		{
			name: "empty tags not included",
			inputs: PrivateLocationInputs{
				Label:         "No Tags",
				AgentPolicyID: "policy-1",
				Tags:          []string{},
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
			body, err := buildPrivateLocationBody(tt.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, body)
		})
	}
}
