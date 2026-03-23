package security

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func TestBuildKibanaRoleBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs KibanaSecurityRoleInputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "with elasticsearch, kibana and metadata JSON",
			inputs: KibanaSecurityRoleInputs{
				Name:          "full-role",
				Elasticsearch: strPtr(`{"cluster":["monitor"],"indices":[{"names":["logs-*"],"privileges":["read"]}]}`),
				Kibana:        strPtr(`[{"base":["all"],"spaces":["default"]}]`),
				Metadata:      strPtr(`{"version":1}`),
			},
			checks: func(t *testing.T, body map[string]any) {
				es, ok := body["elasticsearch"].(map[string]any)
				if !ok {
					t.Fatalf("elasticsearch is not a map, got %T", body["elasticsearch"])
				}
				cluster, ok := es["cluster"].([]any)
				if !ok {
					t.Fatalf("cluster is not a slice, got %T", es["cluster"])
				}
				if len(cluster) != 1 || cluster[0] != "monitor" {
					t.Errorf("cluster = %v, want [monitor]", cluster)
				}
				indices, ok := es["indices"].([]any)
				if !ok {
					t.Fatalf("indices is not a slice, got %T", es["indices"])
				}
				if len(indices) != 1 {
					t.Errorf("len(indices) = %d, want 1", len(indices))
				}

				kb, ok := body["kibana"].([]any)
				if !ok {
					t.Fatalf("kibana is not a slice, got %T", body["kibana"])
				}
				if len(kb) != 1 {
					t.Errorf("len(kibana) = %d, want 1", len(kb))
				}

				meta, ok := body["metadata"].(map[string]any)
				if !ok {
					t.Fatalf("metadata is not a map, got %T", body["metadata"])
				}
				if meta["version"] != float64(1) {
					t.Errorf("metadata.version = %v, want 1", meta["version"])
				}
			},
		},
		{
			name: "minimal - just name, no optional fields",
			inputs: KibanaSecurityRoleInputs{
				Name: "empty-role",
			},
			checks: func(t *testing.T, body map[string]any) {
				if _, ok := body["elasticsearch"]; ok {
					t.Error("elasticsearch should not be present when nil")
				}
				if _, ok := body["kibana"]; ok {
					t.Error("kibana should not be present when nil")
				}
				if _, ok := body["metadata"]; ok {
					t.Error("metadata should not be present when nil")
				}
				// body should be empty
				if len(body) != 0 {
					t.Errorf("body should be empty for minimal role, got %d keys: %v", len(body), body)
				}
			},
		},
		{
			name: "only elasticsearch privileges",
			inputs: KibanaSecurityRoleInputs{
				Name:          "es-only-role",
				Elasticsearch: strPtr(`{"cluster":["all"]}`),
			},
			checks: func(t *testing.T, body map[string]any) {
				es, ok := body["elasticsearch"].(map[string]any)
				if !ok {
					t.Fatalf("elasticsearch is not a map, got %T", body["elasticsearch"])
				}
				cluster, ok := es["cluster"].([]any)
				if !ok {
					t.Fatalf("cluster is not a slice, got %T", es["cluster"])
				}
				if len(cluster) != 1 || cluster[0] != "all" {
					t.Errorf("cluster = %v, want [all]", cluster)
				}

				if _, ok := body["kibana"]; ok {
					t.Error("kibana should not be present when nil")
				}
				if _, ok := body["metadata"]; ok {
					t.Error("metadata should not be present when nil")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildKibanaRoleBody(tc.inputs)
			tc.checks(t, body)
		})
	}
}
