package enrich

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestBuildEnrichPolicyBody(t *testing.T) {
	tests := []struct {
		name     string
		inputs   PolicyInputs
		wantJSON string
	}{
		{
			name: "match type minimal",
			inputs: PolicyInputs{
				Name:         "match-policy",
				PolicyType:   "match",
				Indices:      []string{"users"},
				MatchField:   "email",
				EnrichFields: []string{"first_name", "last_name"},
			},
			wantJSON: `{"match":{"indices":["users"],"match_field":"email","enrich_fields":["first_name","last_name"]}}`,
		},
		{
			name: "geo_match type",
			inputs: PolicyInputs{
				Name:         "geo-policy",
				PolicyType:   "geo_match",
				Indices:      []string{"geo-data"},
				MatchField:   "location",
				EnrichFields: []string{"city", "country"},
			},
			wantJSON: `{"geo_match":{"indices":["geo-data"],"match_field":"location","enrich_fields":["city","country"]}}`,
		},
		{
			name: "range type",
			inputs: PolicyInputs{
				Name:         "range-policy",
				PolicyType:   "range",
				Indices:      []string{"ranges"},
				MatchField:   "ip_range",
				EnrichFields: []string{"network_name"},
			},
			wantJSON: `{"range":{"indices":["ranges"],"match_field":"ip_range","enrich_fields":["network_name"]}}`,
		},
		{
			name: "with multiple indices",
			inputs: PolicyInputs{
				Name:         "multi-idx",
				PolicyType:   "match",
				Indices:      []string{"idx-a", "idx-b", "idx-c"},
				MatchField:   "id",
				EnrichFields: []string{"value"},
			},
			wantJSON: `{"match":{"indices":["idx-a","idx-b","idx-c"],"match_field":"id","enrich_fields":["value"]}}`,
		},
		{
			name: "with query JSON",
			inputs: PolicyInputs{
				Name:         "query-policy",
				PolicyType:   "match",
				Indices:      []string{"data"},
				MatchField:   "key",
				EnrichFields: []string{"val"},
				Query:        strPtr(`{"match_all":{}}`),
			},
			wantJSON: `{"match":{"indices":["data"],"match_field":"key","enrich_fields":["val"],"query":{"match_all":{}}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildEnrichPolicyBody(tc.inputs)
			gotBytes, _ := json.Marshal(body)

			// Normalize both by unmarshalling into map[string]any
			var gotMap, wantMap map[string]any
			if err := json.Unmarshal(gotBytes, &gotMap); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.wantJSON), &wantMap); err != nil {
				t.Fatal(err)
			}

			gotNorm, _ := json.Marshal(gotMap)
			wantNorm, _ := json.Marshal(wantMap)

			if string(gotNorm) != string(wantNorm) {
				t.Errorf("got  %s\nwant %s", gotNorm, wantNorm)
			}
		})
	}
}

func TestShouldExecute(t *testing.T) {
	tests := []struct {
		name     string
		specific *bool
		general  *bool
		want     bool
	}{
		{
			name:     "both nil",
			specific: nil,
			general:  nil,
			want:     false,
		},
		{
			name:     "specific true general nil",
			specific: boolPtr(true),
			general:  nil,
			want:     true,
		},
		{
			name:     "specific false general nil",
			specific: boolPtr(false),
			general:  nil,
			want:     false,
		},
		{
			name:     "specific nil general true",
			specific: nil,
			general:  boolPtr(true),
			want:     true,
		},
		{
			name:     "specific nil general false",
			specific: nil,
			general:  boolPtr(false),
			want:     false,
		},
		{
			name:     "specific true general false",
			specific: boolPtr(true),
			general:  boolPtr(false),
			want:     true,
		},
		{
			name:     "specific false general true",
			specific: boolPtr(false),
			general:  boolPtr(true),
			want:     true,
		},
		{
			name:     "both true",
			specific: boolPtr(true),
			general:  boolPtr(true),
			want:     true,
		},
		{
			name:     "both false",
			specific: boolPtr(false),
			general:  boolPtr(false),
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldExecute(tc.specific, tc.general)
			if got != tc.want {
				t.Errorf("shouldExecute() = %v, want %v", got, tc.want)
			}
		})
	}
}
