package watcher

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestBuildWatchBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     WatchInputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name: "minimal required fields",
			inputs: WatchInputs{
				WatchId: "watch-1",
				Trigger: `{"schedule":{"interval":"10m"}}`,
				Input:   `{"search":{"request":{"indices":["logs"],"body":{"query":{"match_all":{}}}}}}`,
				Actions: `{"log":{"logging":{"text":"found results"}}}`,
			},
			wantKeys: map[string]any{
				"trigger": map[string]any{"schedule": map[string]any{"interval": "10m"}},
				"input": map[string]any{
					"search": map[string]any{
						"request": map[string]any{
							"indices": []any{"logs"},
							"body":    map[string]any{"query": map[string]any{"match_all": map[string]any{}}},
						},
					},
				},
				"actions": map[string]any{"log": map[string]any{"logging": map[string]any{"text": "found results"}}},
			},
			absentKeys: []string{"condition", "transform", "throttle_period", "metadata"},
		},
		{
			name: "with condition JSON",
			inputs: WatchInputs{
				WatchId:   "watch-cond",
				Trigger:   `{"schedule":{"interval":"1h"}}`,
				Input:     `{"none":{}}`,
				Actions:   `{"notify":{}}`,
				Condition: strPtr(`{"compare":{"ctx.payload.hits.total.value":{"gt":0}}}`),
			},
			wantKeys: map[string]any{
				"condition": map[string]any{
					"compare": map[string]any{"ctx.payload.hits.total.value": map[string]any{"gt": float64(0)}},
				},
			},
		},
		{
			name: "with transform JSON",
			inputs: WatchInputs{
				WatchId:   "watch-xform",
				Trigger:   `{"schedule":{"interval":"1h"}}`,
				Input:     `{"none":{}}`,
				Actions:   `{"notify":{}}`,
				Transform: strPtr(`{"script":{"source":"return ['count': ctx.payload.hits.total.value]"}}`),
			},
			wantKeys: map[string]any{
				"transform": map[string]any{
					"script": map[string]any{"source": "return ['count': ctx.payload.hits.total.value]"},
				},
			},
		},
		{
			name: "with throttle_period",
			inputs: WatchInputs{
				WatchId:        "watch-throttle",
				Trigger:        `{"schedule":{"interval":"1h"}}`,
				Input:          `{"none":{}}`,
				Actions:        `{"notify":{}}`,
				ThrottlePeriod: strPtr("5m"),
			},
			wantKeys: map[string]any{
				"throttle_period": "5m",
			},
		},
		{
			name: "with metadata JSON",
			inputs: WatchInputs{
				WatchId:  "watch-meta",
				Trigger:  `{"schedule":{"interval":"1h"}}`,
				Input:    `{"none":{}}`,
				Actions:  `{"notify":{}}`,
				Metadata: strPtr(`{"severity":"high"}`),
			},
			wantKeys: map[string]any{
				"metadata": map[string]any{"severity": "high"},
			},
		},
		{
			name: "all optional fields set",
			inputs: WatchInputs{
				WatchId:        "watch-full",
				Trigger:        `{"schedule":{"cron":"0 0 * * *"}}`,
				Input:          `{"simple":{"count":5}}`,
				Actions:        `{"email":{"email":{"to":"admin@example.com"}}}`,
				Condition:      strPtr(`{"always":{}}`),
				Transform:      strPtr(`{"search":{"request":{"indices":["data"]}}}`),
				ThrottlePeriod: strPtr("10m"),
				Metadata:       strPtr(`{"env":"prod"}`),
			},
			wantKeys: map[string]any{
				"trigger": map[string]any{"schedule": map[string]any{"cron": "0 0 * * *"}},
				"input":   map[string]any{"simple": map[string]any{"count": float64(5)}},
				"actions": map[string]any{
					"email": map[string]any{"email": map[string]any{"to": "admin@example.com"}},
				},
				"condition": map[string]any{"always": map[string]any{}},
				"transform": map[string]any{
					"search": map[string]any{"request": map[string]any{"indices": []any{"data"}}},
				},
				"throttle_period": "10m",
				"metadata":        map[string]any{"env": "prod"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildWatchBody(tc.inputs)
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
