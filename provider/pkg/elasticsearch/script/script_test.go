package script

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestBuildScriptBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     Inputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name: "minimal required fields",
			inputs: Inputs{
				ScriptId: "my-script",
				Lang:     "painless",
				Source:   "doc['field'].value * 2",
			},
			wantKeys: map[string]any{
				"script": map[string]any{
					"lang":   "painless",
					"source": "doc['field'].value * 2",
				},
			},
			absentKeys: []string{"context"},
		},
		{
			name: "mustache language",
			inputs: Inputs{
				ScriptId: "search-tmpl",
				Lang:     "mustache",
				Source:   `{"query":{"match":{"{{field}}":"{{value}}"}}}`,
			},
			wantKeys: map[string]any{
				"script": map[string]any{
					"lang":   "mustache",
					"source": `{"query":{"match":{"{{field}}":"{{value}}"}}}`,
				},
			},
		},
		{
			name: "with context",
			inputs: Inputs{
				ScriptId: "ctx-script",
				Lang:     "painless",
				Source:   "return true",
				Context:  strPtr("score"),
			},
			wantKeys: map[string]any{
				"script": map[string]any{
					"lang":   "painless",
					"source": "return true",
				},
				"context": "score",
			},
		},
		{
			name: "with params JSON",
			inputs: Inputs{
				ScriptId: "param-script",
				Lang:     "painless",
				Source:   "doc['field'].value * params.factor",
				Params:   strPtr(`{"factor":10,"label":"test"}`),
			},
			wantKeys: map[string]any{
				"script": map[string]any{
					"lang":   "painless",
					"source": "doc['field'].value * params.factor",
					"params": map[string]any{"factor": float64(10), "label": "test"},
				},
			},
		},
		{
			name: "with context and params",
			inputs: Inputs{
				ScriptId: "full-script",
				Lang:     "painless",
				Source:   "return params.val",
				Context:  strPtr("filter"),
				Params:   strPtr(`{"val":true}`),
			},
			wantKeys: map[string]any{
				"script": map[string]any{
					"lang":   "painless",
					"source": "return params.val",
					"params": map[string]any{"val": true},
				},
				"context": "filter",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildScriptBody(tc.inputs)
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
