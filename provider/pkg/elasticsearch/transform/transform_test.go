package transform

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestBuildTransformBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     Inputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name:     "empty transform",
			inputs:   Inputs{Name: "empty"},
			wantKeys: map[string]any{},
			absentKeys: []string{
				"source",
				"dest",
				"pivot",
				"latest",
				"frequency",
				"sync",
				"retention_policy",
				"description",
				"_meta",
			},
		},
		{
			name: "with source JSON",
			inputs: Inputs{
				Name:   "src",
				Source: strPtr(`{"index":["logs-*"]}`),
			},
			wantKeys: map[string]any{
				"source": map[string]any{"index": []any{"logs-*"}},
			},
		},
		{
			name: "with destination JSON",
			inputs: Inputs{
				Name:        "dest",
				Destination: strPtr(`{"index":"results"}`),
			},
			wantKeys: map[string]any{
				"dest": map[string]any{"index": "results"},
			},
		},
		{
			name: "with pivot JSON",
			inputs: Inputs{
				Name: "pvt",
				Pivot: strPtr(
					`{"group_by":{"host":{"terms":{"field":"host"}}},"aggregations":{"avg_cpu":{"avg":{"field":"cpu"}}}}`,
				),
			},
			wantKeys: map[string]any{
				"pivot": map[string]any{
					"group_by": map[string]any{
						"host": map[string]any{"terms": map[string]any{"field": "host"}},
					},
					"aggregations": map[string]any{
						"avg_cpu": map[string]any{"avg": map[string]any{"field": "cpu"}},
					},
				},
			},
		},
		{
			name: "with latest JSON",
			inputs: Inputs{
				Name:   "lat",
				Latest: strPtr(`{"unique_key":["host"],"sort":"@timestamp"}`),
			},
			wantKeys: map[string]any{
				"latest": map[string]any{"unique_key": []any{"host"}, "sort": "@timestamp"},
			},
		},
		{
			name: "with frequency string",
			inputs: Inputs{
				Name:      "freq",
				Frequency: strPtr("5m"),
			},
			wantKeys: map[string]any{
				"frequency": "5m",
			},
		},
		{
			name: "with sync JSON",
			inputs: Inputs{
				Name: "synced",
				Sync: strPtr(`{"time":{"field":"@timestamp","delay":"60s"}}`),
			},
			wantKeys: map[string]any{
				"sync": map[string]any{"time": map[string]any{"field": "@timestamp", "delay": "60s"}},
			},
		},
		{
			name: "with retention_policy JSON",
			inputs: Inputs{
				Name:            "rp",
				RetentionPolicy: strPtr(`{"time":{"field":"@timestamp","max_age":"30d"}}`),
			},
			wantKeys: map[string]any{
				"retention_policy": map[string]any{"time": map[string]any{"field": "@timestamp", "max_age": "30d"}},
			},
		},
		{
			name: "with description",
			inputs: Inputs{
				Name:        "desc",
				Description: strPtr("My transform"),
			},
			wantKeys: map[string]any{
				"description": "My transform",
			},
		},
		{
			name: "with metadata JSON (stored as _meta)",
			inputs: Inputs{
				Name:     "meta",
				Metadata: strPtr(`{"version":1}`),
			},
			wantKeys: map[string]any{
				"_meta": map[string]any{"version": float64(1)},
			},
		},
		{
			name: "all optional fields set",
			inputs: Inputs{
				Name:            "full",
				Source:          strPtr(`{"index":["src"]}`),
				Destination:     strPtr(`{"index":"dst"}`),
				Pivot:           strPtr(`{"group_by":{},"aggregations":{}}`),
				Frequency:       strPtr("1m"),
				Sync:            strPtr(`{"time":{"field":"ts"}}`),
				RetentionPolicy: strPtr(`{"time":{"field":"ts","max_age":"7d"}}`),
				Description:     strPtr("full transform"),
				Metadata:        strPtr(`{"v":2}`),
			},
			wantKeys: map[string]any{
				"source":           map[string]any{"index": []any{"src"}},
				"dest":             map[string]any{"index": "dst"},
				"pivot":            map[string]any{"group_by": map[string]any{}, "aggregations": map[string]any{}},
				"frequency":        "1m",
				"sync":             map[string]any{"time": map[string]any{"field": "ts"}},
				"retention_policy": map[string]any{"time": map[string]any{"field": "ts", "max_age": "7d"}},
				"description":      "full transform",
				"_meta":            map[string]any{"v": float64(2)},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildTransformBody(tc.inputs)
			if err != nil {
				t.Fatal(err)
			}
			bodyJSON, _ := json.Marshal(body)
			wantJSON, _ := json.Marshal(tc.wantKeys)
			var bodyMap, wantMap map[string]any
			if err := json.Unmarshal(bodyJSON, &bodyMap); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(wantJSON, &wantMap); err != nil {
				t.Fatal(err)
			}

			for key, wantVal := range wantMap {
				gotVal, ok := bodyMap[key]
				if !ok {
					t.Errorf("missing key %q", key)
					continue
				}
				gotBytes, _ := json.Marshal(gotVal)
				wantBytes, _ := json.Marshal(wantVal)
				if string(gotBytes) != string(wantBytes) {
					t.Errorf("key %q: got %s, want %s", key, gotBytes, wantBytes)
				}
			}

			for _, key := range tc.absentKeys {
				if _, ok := body[key]; ok {
					t.Errorf("unexpected key %q in body", key)
				}
			}
		})
	}
}

func TestBuildTransformUpdateBody(t *testing.T) {
	// buildTransformUpdateBody delegates to buildTransformBody, so verify it returns the same result
	inputs := Inputs{
		Name:        "update-test",
		Source:      strPtr(`{"index":["data"]}`),
		Description: strPtr("updated"),
	}

	body, err := buildTransformUpdateBody(inputs)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := body["source"]; !ok {
		t.Error("expected 'source' key in update body")
	}
	if _, ok := body["description"]; !ok {
		t.Error("expected 'description' key in update body")
	}
}
