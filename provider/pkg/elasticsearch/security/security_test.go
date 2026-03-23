package security

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestBuildUserBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     UserInputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name: "minimal with roles only",
			inputs: UserInputs{
				Username: "alice",
				Roles:    []string{"admin"},
			},
			wantKeys: map[string]any{
				"roles": []any{"admin"},
			},
			absentKeys: []string{"password", "password_hash", "full_name", "email", "metadata", "enabled"},
		},
		{
			name: "with password",
			inputs: UserInputs{
				Username: "bob",
				Password: strPtr("s3cret"),
				Roles:    []string{"viewer"},
			},
			wantKeys: map[string]any{
				"password": "s3cret",
				"roles":    []any{"viewer"},
			},
		},
		{
			name: "with password hash",
			inputs: UserInputs{
				Username:     "carol",
				PasswordHash: strPtr("$2a$10$hash"),
				Roles:        []string{},
			},
			wantKeys: map[string]any{
				"password_hash": "$2a$10$hash",
			},
		},
		{
			name: "with full_name and email",
			inputs: UserInputs{
				Username: "dave",
				Roles:    []string{"user"},
				FullName: strPtr("Dave Smith"),
				Email:    strPtr("dave@example.com"),
			},
			wantKeys: map[string]any{
				"full_name": "Dave Smith",
				"email":     "dave@example.com",
			},
		},
		{
			name: "with metadata JSON",
			inputs: UserInputs{
				Username: "eve",
				Roles:    []string{},
				Metadata: strPtr(`{"team":"security"}`),
			},
			wantKeys: map[string]any{
				"metadata": map[string]any{"team": "security"},
			},
		},
		{
			name: "with enabled true",
			inputs: UserInputs{
				Username: "frank",
				Roles:    []string{"admin"},
				Enabled:  boolPtr(true),
			},
			wantKeys: map[string]any{
				"enabled": true,
			},
		},
		{
			name: "with enabled false",
			inputs: UserInputs{
				Username: "grace",
				Roles:    []string{},
				Enabled:  boolPtr(false),
			},
			wantKeys: map[string]any{
				"enabled": false,
			},
		},
		{
			name: "all fields set",
			inputs: UserInputs{
				Username:     "heidi",
				Password:     strPtr("pw"),
				PasswordHash: strPtr("$2a$10$hash"),
				Roles:        []string{"admin", "user"},
				FullName:     strPtr("Heidi"),
				Email:        strPtr("heidi@example.com"),
				Metadata:     strPtr(`{"org":"eng"}`),
				Enabled:      boolPtr(true),
			},
			wantKeys: map[string]any{
				"password":      "pw",
				"password_hash": "$2a$10$hash",
				"roles":         []any{"admin", "user"},
				"full_name":     "Heidi",
				"email":         "heidi@example.com",
				"metadata":      map[string]any{"org": "eng"},
				"enabled":       true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildUserBody(tc.inputs)
			if err != nil {
				t.Fatal(err)
			}

			// Always expect roles key
			if _, ok := body["roles"]; !ok {
				t.Fatal("expected 'roles' key in body")
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

func TestBuildRoleBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     RoleInputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name:       "empty role",
			inputs:     RoleInputs{Name: "empty"},
			wantKeys:   map[string]any{},
			absentKeys: []string{"cluster", "indices", "applications", "run_as", "metadata", "global"},
		},
		{
			name: "with cluster privileges",
			inputs: RoleInputs{
				Name:    "monitor-role",
				Cluster: []string{"monitor", "manage"},
			},
			wantKeys: map[string]any{
				"cluster": []any{"monitor", "manage"},
			},
		},
		{
			name: "with indices JSON",
			inputs: RoleInputs{
				Name:    "idx-role",
				Indices: strPtr(`[{"names":["logs-*"],"privileges":["read"]}]`),
			},
			wantKeys: map[string]any{
				"indices": []any{map[string]any{"names": []any{"logs-*"}, "privileges": []any{"read"}}},
			},
		},
		{
			name: "with run_as",
			inputs: RoleInputs{
				Name:  "impersonate",
				RunAs: []string{"admin", "deploy"},
			},
			wantKeys: map[string]any{
				"run_as": []any{"admin", "deploy"},
			},
		},
		{
			name: "with metadata JSON",
			inputs: RoleInputs{
				Name:     "meta-role",
				Metadata: strPtr(`{"version":1}`),
			},
			wantKeys: map[string]any{
				"metadata": map[string]any{"version": float64(1)},
			},
		},
		{
			name: "with applications JSON",
			inputs: RoleInputs{
				Name:         "app-role",
				Applications: strPtr(`[{"application":"myapp","privileges":["read"],"resources":["*"]}]`),
			},
			wantKeys: map[string]any{
				"applications": []any{map[string]any{
					"application": "myapp",
					"privileges":  []any{"read"},
					"resources":   []any{"*"},
				}},
			},
		},
		{
			name: "with global JSON",
			inputs: RoleInputs{
				Name:   "global-role",
				Global: strPtr(`{"application":{"manage":{"applications":["myapp"]}}}`),
			},
			wantKeys: map[string]any{
				"global": map[string]any{
					"application": map[string]any{
						"manage": map[string]any{
							"applications": []any{"myapp"},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildRoleBody(tc.inputs)
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

func TestBuildRoleMappingBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     RoleMappingInputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name: "minimal with rules only",
			inputs: RoleMappingInputs{
				Name:  "basic",
				Rules: `{"field":{"username":"*"}}`,
			},
			wantKeys: map[string]any{
				"rules": map[string]any{"field": map[string]any{"username": "*"}},
			},
			absentKeys: []string{"enabled", "roles", "role_templates", "metadata"},
		},
		{
			name: "with enabled and roles",
			inputs: RoleMappingInputs{
				Name:    "with-roles",
				Enabled: boolPtr(true),
				Roles:   []string{"admin", "user"},
				Rules:   `{"field":{"groups":"cn=admins"}}`,
			},
			wantKeys: map[string]any{
				"enabled": true,
				"roles":   []any{"admin", "user"},
				"rules":   map[string]any{"field": map[string]any{"groups": "cn=admins"}},
			},
		},
		{
			name: "with enabled false",
			inputs: RoleMappingInputs{
				Name:    "disabled",
				Enabled: boolPtr(false),
				Rules:   `{"match_all":{}}`,
			},
			wantKeys: map[string]any{
				"enabled": false,
			},
		},
		{
			name: "with role_templates JSON",
			inputs: RoleMappingInputs{
				Name:          "tmpl",
				RoleTemplates: strPtr(`[{"template":{"source":"{{#tojson}}groups{{/tojson}}"}}]`),
				Rules:         `{"field":{"realm.name":"native"}}`,
			},
			wantKeys: map[string]any{
				"role_templates": []any{map[string]any{
					"template": map[string]any{"source": "{{#tojson}}groups{{/tojson}}"},
				}},
			},
		},
		{
			name: "with metadata JSON",
			inputs: RoleMappingInputs{
				Name:     "meta",
				Rules:    `{"match_all":{}}`,
				Metadata: strPtr(`{"version":2}`),
			},
			wantKeys: map[string]any{
				"metadata": map[string]any{"version": float64(2)},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildRoleMappingBody(tc.inputs)
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

func TestBuildApiKeyBody(t *testing.T) {
	tests := []struct {
		name       string
		inputs     ApiKeyInputs
		wantKeys   map[string]any
		absentKeys []string
	}{
		{
			name: "name only",
			inputs: ApiKeyInputs{
				Name: "my-key",
			},
			wantKeys: map[string]any{
				"name": "my-key",
			},
			absentKeys: []string{"role_descriptors", "expiration", "metadata"},
		},
		{
			name: "with role_descriptors JSON",
			inputs: ApiKeyInputs{
				Name:            "rd-key",
				RoleDescriptors: strPtr(`{"role1":{"cluster":["monitor"]}}`),
			},
			wantKeys: map[string]any{
				"name":             "rd-key",
				"role_descriptors": map[string]any{"role1": map[string]any{"cluster": []any{"monitor"}}},
			},
		},
		{
			name: "with expiration",
			inputs: ApiKeyInputs{
				Name:       "exp-key",
				Expiration: strPtr("7d"),
			},
			wantKeys: map[string]any{
				"name":       "exp-key",
				"expiration": "7d",
			},
		},
		{
			name: "with metadata JSON",
			inputs: ApiKeyInputs{
				Name:     "meta-key",
				Metadata: strPtr(`{"app":"test"}`),
			},
			wantKeys: map[string]any{
				"name":     "meta-key",
				"metadata": map[string]any{"app": "test"},
			},
		},
		{
			name: "all fields set",
			inputs: ApiKeyInputs{
				Name:            "full-key",
				RoleDescriptors: strPtr(`{"r":{}}`),
				Expiration:      strPtr("1d"),
				Metadata:        strPtr(`{"env":"prod"}`),
			},
			wantKeys: map[string]any{
				"name":             "full-key",
				"role_descriptors": map[string]any{"r": map[string]any{}},
				"expiration":       "1d",
				"metadata":         map[string]any{"env": "prod"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildApiKeyBody(tc.inputs)
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

func TestPtrStringChanged(t *testing.T) {
	tests := []struct {
		name string
		a    *string
		b    *string
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: false,
		},
		{
			name: "a nil b set",
			a:    nil,
			b:    strPtr("hello"),
			want: true,
		},
		{
			name: "a set b nil",
			a:    strPtr("hello"),
			b:    nil,
			want: true,
		},
		{
			name: "equal values",
			a:    strPtr("same"),
			b:    strPtr("same"),
			want: false,
		},
		{
			name: "different values",
			a:    strPtr("foo"),
			b:    strPtr("bar"),
			want: true,
		},
		{
			name: "empty strings equal",
			a:    strPtr(""),
			b:    strPtr(""),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ptrStringChanged(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("ptrStringChanged() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBoolPtr(t *testing.T) {
	tests := []struct {
		name  string
		input bool
		want  bool
	}{
		{name: "true", input: true, want: true},
		{name: "false", input: false, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := boolPtr(tc.input)
			if got == nil {
				t.Fatal("boolPtr returned nil")
			}
			if *got != tc.want {
				t.Errorf("boolPtr(%v) = %v, want %v", tc.input, *got, tc.want)
			}
		})
	}
}
