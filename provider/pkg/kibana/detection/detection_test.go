package detection

import (
	"testing"
)

const namespaceTypeSingle = "single"

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
		{"custom space", strPtr("security"), "security"},
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
// resolveNamespaceType
// ---------------------------------------------------------------------------

func TestResolveNamespaceType(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil returns single", nil, namespaceTypeSingle},
		{"empty returns single", strPtr(""), namespaceTypeSingle},
		{"agnostic", strPtr("agnostic"), "agnostic"},
		{"single explicit", strPtr(namespaceTypeSingle), namespaceTypeSingle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveNamespaceType(tt.in)
			if got != tt.want {
				t.Errorf("resolveNamespaceType() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildDetectionRuleBody
// ---------------------------------------------------------------------------

func TestBuildDetectionRuleBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs SecurityDetectionRuleInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - required fields only",
			inputs: SecurityDetectionRuleInputs{
				Name:        "Test Rule",
				Description: "Detects bad things",
				RiskScore:   50,
				Severity:    "medium",
				RuleType:    "query",
			},
			check: func(t *testing.T, body map[string]any) {
				if body["name"] != "Test Rule" {
					t.Errorf("name = %v, want Test Rule", body["name"])
				}
				if body["description"] != "Detects bad things" {
					t.Errorf("description = %v, want Detects bad things", body["description"])
				}
				if body["risk_score"] != 50 {
					t.Errorf("risk_score = %v, want 50", body["risk_score"])
				}
				if body["severity"] != "medium" {
					t.Errorf("severity = %v, want medium", body["severity"])
				}
				if body["type"] != "query" {
					t.Errorf("type = %v, want query", body["type"])
				}
				// Optional keys should not be present
				for _, key := range []string{"query", "language", "index", "filters", "enabled", "interval", "from", "to", "tags", "actions", "exceptions_list"} {
					if _, exists := body[key]; exists {
						t.Errorf("unexpected key %q in minimal body", key)
					}
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: SecurityDetectionRuleInputs{
				Name:          "Full Rule",
				Description:   "Full detection rule",
				RiskScore:     75,
				Severity:      "high",
				RuleType:      "eql",
				Query:         strPtr("process where process.name == \"cmd.exe\""),
				Language:      strPtr("eql"),
				IndexPatterns: []string{"logs-*", "winlogbeat-*"},
				Filters:       strPtr(`[{"meta":{"disabled":false}}]`),
				Enabled:       boolPtr(true),
				Interval:      strPtr("5m"),
				FromTime:      strPtr("now-360s"),
				ToTime:        strPtr("now"),
				Tags:          []string{"windows", "process"},
				Actions:       strPtr(`[{"group":"default","id":"my-action","action_type_id":".slack"}]`),
				ExceptionsList: strPtr(
					`[{"id":"my-list","list_id":"my-list","type":"detection","namespace_type":"single"}]`,
				),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["name"] != "Full Rule" {
					t.Errorf("name = %v, want Full Rule", body["name"])
				}
				if body["type"] != "eql" {
					t.Errorf("type = %v, want eql", body["type"])
				}
				if body["query"] != `process where process.name == "cmd.exe"` {
					t.Errorf("query = %v, want process where...", body["query"])
				}
				if body["language"] != "eql" {
					t.Errorf("language = %v, want eql", body["language"])
				}

				idx, ok := body["index"].([]string)
				if !ok {
					t.Fatalf("index should be []string, got %T", body["index"])
				}
				if len(idx) != 2 || idx[0] != "logs-*" {
					t.Errorf("index = %v, want [logs-* winlogbeat-*]", idx)
				}

				// filters should be parsed JSON array
				filters, ok := body["filters"].([]any)
				if !ok {
					t.Fatalf("filters should be parsed JSON array, got %T", body["filters"])
				}
				if len(filters) != 1 {
					t.Errorf("filters length = %d, want 1", len(filters))
				}

				if body["enabled"] != true {
					t.Errorf("enabled = %v, want true", body["enabled"])
				}
				if body["interval"] != "5m" {
					t.Errorf("interval = %v, want 5m", body["interval"])
				}
				if body["from"] != "now-360s" {
					t.Errorf("from = %v, want now-360s", body["from"])
				}
				if body["to"] != "now" {
					t.Errorf("to = %v, want now", body["to"])
				}

				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags should be []string, got %T", body["tags"])
				}
				if len(tags) != 2 {
					t.Errorf("tags length = %d, want 2", len(tags))
				}

				// actions should be parsed JSON
				actions, ok := body["actions"].([]any)
				if !ok {
					t.Fatalf("actions should be parsed JSON array, got %T", body["actions"])
				}
				if len(actions) != 1 {
					t.Errorf("actions length = %d, want 1", len(actions))
				}

				// exceptions_list should be parsed JSON
				exceptions, ok := body["exceptions_list"].([]any)
				if !ok {
					t.Fatalf("exceptions_list should be parsed JSON array, got %T", body["exceptions_list"])
				}
				if len(exceptions) != 1 {
					t.Errorf("exceptions_list length = %d, want 1", len(exceptions))
				}
			},
		},
		{
			name: "empty slices not included",
			inputs: SecurityDetectionRuleInputs{
				Name:          "Minimal Rule",
				Description:   "desc",
				RiskScore:     10,
				Severity:      "low",
				RuleType:      "query",
				IndexPatterns: []string{},
				Tags:          []string{},
			},
			check: func(t *testing.T, body map[string]any) {
				if _, exists := body["index"]; exists {
					t.Error("empty IndexPatterns should not produce index key")
				}
				if _, exists := body["tags"]; exists {
					t.Error("empty Tags should not produce tags key")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildDetectionRuleBody(tt.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildExceptionListBody
// ---------------------------------------------------------------------------

func TestBuildExceptionListBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs SecurityExceptionListInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - required fields only",
			inputs: SecurityExceptionListInputs{
				Name:        "My Exception List",
				Description: "Test exceptions",
				ListType:    "detection",
			},
			check: func(t *testing.T, body map[string]any) {
				if body["name"] != "My Exception List" {
					t.Errorf("name = %v, want My Exception List", body["name"])
				}
				if body["description"] != "Test exceptions" {
					t.Errorf("description = %v, want Test exceptions", body["description"])
				}
				if body["type"] != "detection" {
					t.Errorf("type = %v, want detection", body["type"])
				}
				// namespace_type should default to single
				if body["namespace_type"] != namespaceTypeSingle {
					t.Errorf("namespace_type = %v, want single", body["namespace_type"])
				}
				if _, exists := body["tags"]; exists {
					t.Error("tags should not be in minimal body")
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: SecurityExceptionListInputs{
				Name:          "Full List",
				Description:   "Full exception list",
				ListType:      "endpoint",
				NamespaceType: strPtr("agnostic"),
				Tags:          []string{"tag1", "tag2"},
			},
			check: func(t *testing.T, body map[string]any) {
				if body["type"] != "endpoint" {
					t.Errorf("type = %v, want endpoint", body["type"])
				}
				if body["namespace_type"] != "agnostic" {
					t.Errorf("namespace_type = %v, want agnostic", body["namespace_type"])
				}
				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags should be []string, got %T", body["tags"])
				}
				if len(tags) != 2 || tags[0] != "tag1" {
					t.Errorf("tags = %v, want [tag1 tag2]", tags)
				}
			},
		},
		{
			name: "empty tags not included",
			inputs: SecurityExceptionListInputs{
				Name:        "No Tags",
				Description: "desc",
				ListType:    "detection",
				Tags:        []string{},
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
			body := buildExceptionListBody(tt.inputs)
			tt.check(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildExceptionItemBody
// ---------------------------------------------------------------------------

func TestBuildExceptionItemBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs SecurityExceptionItemInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - required fields only",
			inputs: SecurityExceptionItemInputs{
				ListID:      "my-list",
				Name:        "My Item",
				Description: "An exception item",
				ItemType:    "simple",
				Entries:     `[{"field":"host.name","operator":"included","type":"match","value":"safe-host"}]`,
			},
			check: func(t *testing.T, body map[string]any) {
				if body["list_id"] != "my-list" {
					t.Errorf("list_id = %v, want my-list", body["list_id"])
				}
				if body["name"] != "My Item" {
					t.Errorf("name = %v, want My Item", body["name"])
				}
				if body["description"] != "An exception item" {
					t.Errorf("description = %v, want An exception item", body["description"])
				}
				if body["type"] != "simple" {
					t.Errorf("type = %v, want simple", body["type"])
				}
				// namespace_type defaults to single
				if body["namespace_type"] != namespaceTypeSingle {
					t.Errorf("namespace_type = %v, want single", body["namespace_type"])
				}
				// entries should be parsed JSON
				entries, ok := body["entries"].([]any)
				if !ok {
					t.Fatalf("entries should be parsed JSON array, got %T", body["entries"])
				}
				if len(entries) != 1 {
					t.Errorf("entries length = %d, want 1", len(entries))
				}
				// Optional keys should not be present
				for _, key := range []string{"item_id", "tags", "expire_time", "os_types", "comments"} {
					if _, exists := body[key]; exists {
						t.Errorf("unexpected key %q in minimal body", key)
					}
				}
			},
		},
		{
			name: "all optional fields populated",
			inputs: SecurityExceptionItemInputs{
				ListID:        "my-list",
				Name:          "Full Item",
				Description:   "All fields set",
				ItemType:      "simple",
				NamespaceType: strPtr("agnostic"),
				Entries:       `[{"field":"host.name","operator":"included","type":"match","value":"safe-host"}]`,
				ItemID:        strPtr("my-item-id"),
				Tags:          []string{"tag-a", "tag-b"},
				ExpireTime:    strPtr("2025-12-31T23:59:59Z"),
				OsTypes:       []string{"windows", "linux"},
				Comments:      strPtr(`[{"comment":"This is safe"}]`),
			},
			check: func(t *testing.T, body map[string]any) {
				if body["namespace_type"] != "agnostic" {
					t.Errorf("namespace_type = %v, want agnostic", body["namespace_type"])
				}
				if body["item_id"] != "my-item-id" {
					t.Errorf("item_id = %v, want my-item-id", body["item_id"])
				}

				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags should be []string, got %T", body["tags"])
				}
				if len(tags) != 2 {
					t.Errorf("tags length = %d, want 2", len(tags))
				}

				if body["expire_time"] != "2025-12-31T23:59:59Z" {
					t.Errorf("expire_time = %v, want 2025-12-31T23:59:59Z", body["expire_time"])
				}

				osTypes, ok := body["os_types"].([]string)
				if !ok {
					t.Fatalf("os_types should be []string, got %T", body["os_types"])
				}
				if len(osTypes) != 2 {
					t.Errorf("os_types length = %d, want 2", len(osTypes))
				}

				// comments should be parsed JSON
				comments, ok := body["comments"].([]any)
				if !ok {
					t.Fatalf("comments should be parsed JSON array, got %T", body["comments"])
				}
				if len(comments) != 1 {
					t.Errorf("comments length = %d, want 1", len(comments))
				}
			},
		},
		{
			name: "empty slices not included",
			inputs: SecurityExceptionItemInputs{
				ListID:      "my-list",
				Name:        "No Optionals",
				Description: "desc",
				ItemType:    "simple",
				Entries:     `[]`,
				Tags:        []string{},
				OsTypes:     []string{},
			},
			check: func(t *testing.T, body map[string]any) {
				if _, exists := body["tags"]; exists {
					t.Error("empty tags should not be included")
				}
				if _, exists := body["os_types"]; exists {
					t.Error("empty os_types should not be included")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildExceptionItemBody(tt.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildSecurityListBody
// ---------------------------------------------------------------------------

func TestBuildSecurityListBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs SecurityListInputs
		check  func(t *testing.T, body map[string]any)
	}{
		{
			name: "all required fields",
			inputs: SecurityListInputs{
				Name:        "IP Blocklist",
				Description: "Known bad IPs",
				ListType:    "ip",
			},
			check: func(t *testing.T, body map[string]any) {
				if body["name"] != "IP Blocklist" {
					t.Errorf("name = %v, want IP Blocklist", body["name"])
				}
				if body["description"] != "Known bad IPs" {
					t.Errorf("description = %v, want Known bad IPs", body["description"])
				}
				if body["type"] != "ip" {
					t.Errorf("type = %v, want ip", body["type"])
				}
				// Should have exactly 3 keys
				if len(body) != 3 {
					t.Errorf("body has %d keys, want 3", len(body))
				}
			},
		},
		{
			name: "keyword type",
			inputs: SecurityListInputs{
				Name:        "Keyword List",
				Description: "Keywords",
				ListType:    "keyword",
			},
			check: func(t *testing.T, body map[string]any) {
				if body["type"] != "keyword" {
					t.Errorf("type = %v, want keyword", body["type"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildSecurityListBody(tt.inputs)
			tt.check(t, body)
		})
	}
}
