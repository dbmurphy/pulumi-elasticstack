package space

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func TestBuildSpaceBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs Inputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - just spaceId and name",
			inputs: Inputs{
				SpaceID: "my-space",
				Name:    "My Space",
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["id"] != "my-space" {
					t.Errorf("id = %v, want %q", body["id"], "my-space")
				}
				if body["name"] != "My Space" {
					t.Errorf("name = %v, want %q", body["name"], "My Space")
				}
				if _, ok := body["description"]; ok {
					t.Error("description should not be present when nil")
				}
				if _, ok := body["color"]; ok {
					t.Error("color should not be present when nil")
				}
				if _, ok := body["initials"]; ok {
					t.Error("initials should not be present when nil")
				}
				if _, ok := body["disabledFeatures"]; ok {
					t.Error("disabledFeatures should not be present when empty")
				}
				if _, ok := body["imageUrl"]; ok {
					t.Error("imageUrl should not be present when nil")
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: Inputs{
				SpaceID:          "eng-space",
				Name:             "Engineering",
				Description:      strPtr("The engineering team space"),
				Color:            strPtr("#FF5733"),
				Initials:         strPtr("EN"),
				DisabledFeatures: []string{"ml", "apm"},
				ImageUrl:         strPtr("data:image/png;base64,abc123"),
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["id"] != "eng-space" {
					t.Errorf("id = %v, want %q", body["id"], "eng-space")
				}
				if body["name"] != "Engineering" {
					t.Errorf("name = %v, want %q", body["name"], "Engineering")
				}
				if body["description"] != "The engineering team space" {
					t.Errorf("description = %v, want %q", body["description"], "The engineering team space")
				}
				if body["color"] != "#FF5733" {
					t.Errorf("color = %v, want %q", body["color"], "#FF5733")
				}
				if body["initials"] != "EN" {
					t.Errorf("initials = %v, want %q", body["initials"], "EN")
				}

				features, ok := body["disabledFeatures"].([]string)
				if !ok {
					t.Fatalf("disabledFeatures is not []string, got %T", body["disabledFeatures"])
				}
				if len(features) != 2 || features[0] != "ml" || features[1] != "apm" {
					t.Errorf("disabledFeatures = %v, want [ml apm]", features)
				}

				if body["imageUrl"] != "data:image/png;base64,abc123" {
					t.Errorf("imageUrl = %v, want %q", body["imageUrl"], "data:image/png;base64,abc123")
				}
			},
		},
		{
			name: "partial optional fields - only description and color",
			inputs: Inputs{
				SpaceID:     "ops",
				Name:        "Operations",
				Description: strPtr("Ops space"),
				Color:       strPtr("#00FF00"),
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["id"] != "ops" {
					t.Errorf("id = %v, want %q", body["id"], "ops")
				}
				if body["description"] != "Ops space" {
					t.Errorf("description = %v, want %q", body["description"], "Ops space")
				}
				if body["color"] != "#00FF00" {
					t.Errorf("color = %v, want %q", body["color"], "#00FF00")
				}
				if _, ok := body["initials"]; ok {
					t.Error("initials should not be present when nil")
				}
				if _, ok := body["disabledFeatures"]; ok {
					t.Error("disabledFeatures should not be present when empty")
				}
				if _, ok := body["imageUrl"]; ok {
					t.Error("imageUrl should not be present when nil")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildSpaceBody(tc.inputs)
			tc.checks(t, body)
		})
	}
}
