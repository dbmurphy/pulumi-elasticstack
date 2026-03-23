package lifecycle

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func TestBuildILMBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs IndexLifecycleInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name:   "empty inputs produces empty phases",
			inputs: IndexLifecycleInputs{Name: "my-policy"},
			check: func(t *testing.T, body map[string]any) {
				policy, ok := body["policy"].(map[string]any)
				if !ok {
					t.Fatal("expected policy key")
				}
				phases, ok := policy["phases"].(map[string]any)
				if !ok {
					t.Fatal("expected phases key")
				}
				if len(phases) != 0 {
					t.Errorf("expected empty phases, got %d entries", len(phases))
				}
			},
		},
		{
			name: "with hot phase only",
			inputs: IndexLifecycleInputs{
				Name: "my-policy",
				Hot:  strPtr(`{"actions":{"rollover":{"max_age":"7d"}}}`),
			},
			check: func(t *testing.T, body map[string]any) {
				policy := body["policy"].(map[string]any)
				phases := policy["phases"].(map[string]any)
				hot, ok := phases["hot"]
				if !ok {
					t.Fatal("expected hot phase")
				}
				hotMap, ok := hot.(map[string]any)
				if !ok {
					t.Fatal("expected hot to be a map")
				}
				if hotMap["actions"] == nil {
					t.Error("expected actions in hot phase")
				}
				if _, ok := phases["warm"]; ok {
					t.Error("expected no warm phase")
				}
			},
		},
		{
			name: "with hot and warm phases",
			inputs: IndexLifecycleInputs{
				Name: "my-policy",
				Hot:  strPtr(`{"actions":{"rollover":{"max_age":"7d"}}}`),
				Warm: strPtr(`{"min_age":"30d","actions":{"shrink":{"number_of_shards":1}}}`),
			},
			check: func(t *testing.T, body map[string]any) {
				policy := body["policy"].(map[string]any)
				phases := policy["phases"].(map[string]any)
				if phases["hot"] == nil {
					t.Error("expected hot phase")
				}
				if phases["warm"] == nil {
					t.Error("expected warm phase")
				}
				if len(phases) != 2 {
					t.Errorf("expected 2 phases, got %d", len(phases))
				}
			},
		},
		{
			name: "with cold and delete phases",
			inputs: IndexLifecycleInputs{
				Name:   "my-policy",
				Cold:   strPtr(`{"min_age":"60d","actions":{"freeze":{}}}`),
				Delete: strPtr(`{"min_age":"90d","actions":{"delete":{}}}`),
			},
			check: func(t *testing.T, body map[string]any) {
				policy := body["policy"].(map[string]any)
				phases := policy["phases"].(map[string]any)
				if phases["cold"] == nil {
					t.Error("expected cold phase")
				}
				if phases["delete"] == nil {
					t.Error("expected delete phase")
				}
				if _, ok := phases["hot"]; ok {
					t.Error("expected no hot phase")
				}
			},
		},
		{
			name: "with frozen phase",
			inputs: IndexLifecycleInputs{
				Name: "my-policy",
				Frozen: strPtr(
					`{"min_age":"90d","actions":{"searchable_snapshot":{"snapshot_repository":"found-snapshots"}}}`,
				),
			},
			check: func(t *testing.T, body map[string]any) {
				policy := body["policy"].(map[string]any)
				phases := policy["phases"].(map[string]any)
				if phases["frozen"] == nil {
					t.Error("expected frozen phase")
				}
			},
		},
		{
			name: "with all phases",
			inputs: IndexLifecycleInputs{
				Name:   "my-policy",
				Hot:    strPtr(`{"actions":{"rollover":{"max_age":"7d"}}}`),
				Warm:   strPtr(`{"min_age":"30d","actions":{}}`),
				Cold:   strPtr(`{"min_age":"60d","actions":{}}`),
				Frozen: strPtr(`{"min_age":"90d","actions":{}}`),
				Delete: strPtr(`{"min_age":"120d","actions":{"delete":{}}}`),
			},
			check: func(t *testing.T, body map[string]any) {
				policy := body["policy"].(map[string]any)
				phases := policy["phases"].(map[string]any)
				if len(phases) != 5 {
					t.Errorf("expected 5 phases, got %d", len(phases))
				}
				for _, phase := range []string{"hot", "warm", "cold", "frozen", "delete"} {
					if phases[phase] == nil {
						t.Errorf("expected %s phase", phase)
					}
				}
			},
		},
		{
			name: "with metadata",
			inputs: IndexLifecycleInputs{
				Name:     "my-policy",
				Hot:      strPtr(`{"actions":{}}`),
				Metadata: strPtr(`{"managed_by":"pulumi"}`),
			},
			check: func(t *testing.T, body map[string]any) {
				policy := body["policy"].(map[string]any)
				meta, ok := policy["_meta"]
				if !ok {
					t.Fatal("expected _meta key in policy")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildILMBody(tt.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, body)
		})
	}
}
