package alerting

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
		name     string
		input    *string
		expected string
	}{
		{name: "nil returns default", input: nil, expected: "default"},
		{name: "empty string returns default", input: strPtr(""), expected: "default"},
		{name: "explicit default returns default", input: strPtr("default"), expected: "default"},
		{name: "custom space returned as-is", input: strPtr("my-space"), expected: "my-space"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveSpaceID(tc.input)
			if got != tc.expected {
				t.Errorf("resolveSpaceID() = %q, want %q", got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildConnectorBody
// ---------------------------------------------------------------------------

func TestBuildConnectorBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs ActionConnectorInputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal - name and connector_type_id only",
			inputs: ActionConnectorInputs{
				Name:            "my-connector",
				ConnectorTypeID: ".slack",
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["name"] != "my-connector" {
					t.Errorf("name = %v, want %q", body["name"], "my-connector")
				}
				if body["connector_type_id"] != ".slack" {
					t.Errorf("connector_type_id = %v, want %q", body["connector_type_id"], ".slack")
				}
				if _, ok := body["config"]; ok {
					t.Error("config should not be present when nil")
				}
				if _, ok := body["secrets"]; ok {
					t.Error("secrets should not be present when nil")
				}
			},
		},
		{
			name: "with config and secrets JSON",
			inputs: ActionConnectorInputs{
				Name:            "webhook-conn",
				ConnectorTypeID: ".webhook",
				Config:          strPtr(`{"url":"https://example.com"}`),
				Secrets:         strPtr(`{"token":"abc123"}`),
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["name"] != "webhook-conn" {
					t.Errorf("name = %v, want %q", body["name"], "webhook-conn")
				}
				if body["connector_type_id"] != ".webhook" {
					t.Errorf("connector_type_id = %v, want %q", body["connector_type_id"], ".webhook")
				}
				cfg, ok := body["config"].(map[string]any)
				if !ok {
					t.Fatalf("config is not a map, got %T", body["config"])
				}
				if cfg["url"] != "https://example.com" {
					t.Errorf("config.url = %v, want %q", cfg["url"], "https://example.com")
				}
				sec, ok := body["secrets"].(map[string]any)
				if !ok {
					t.Fatalf("secrets is not a map, got %T", body["secrets"])
				}
				if sec["token"] != "abc123" {
					t.Errorf("secrets.token = %v, want %q", sec["token"], "abc123")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildConnectorBody(tc.inputs)
			tc.checks(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildConnectorUpdateBody
// ---------------------------------------------------------------------------

func TestBuildConnectorUpdateBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs ActionConnectorInputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "should NOT include connector_type_id",
			inputs: ActionConnectorInputs{
				Name:            "my-connector",
				ConnectorTypeID: ".slack",
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["name"] != "my-connector" {
					t.Errorf("name = %v, want %q", body["name"], "my-connector")
				}
				if _, ok := body["connector_type_id"]; ok {
					t.Error("connector_type_id should NOT be present in update body")
				}
			},
		},
		{
			name: "with config and secrets but no connector_type_id",
			inputs: ActionConnectorInputs{
				Name:            "updated-conn",
				ConnectorTypeID: ".webhook",
				Config:          strPtr(`{"url":"https://new.example.com"}`),
				Secrets:         strPtr(`{"token":"new-secret"}`),
			},
			checks: func(t *testing.T, body map[string]any) {
				if _, ok := body["connector_type_id"]; ok {
					t.Error("connector_type_id should NOT be present in update body")
				}
				if body["name"] != "updated-conn" {
					t.Errorf("name = %v, want %q", body["name"], "updated-conn")
				}
				cfg, ok := body["config"].(map[string]any)
				if !ok {
					t.Fatalf("config is not a map, got %T", body["config"])
				}
				if cfg["url"] != "https://new.example.com" {
					t.Errorf("config.url = %v, want %q", cfg["url"], "https://new.example.com")
				}
				sec, ok := body["secrets"].(map[string]any)
				if !ok {
					t.Fatalf("secrets is not a map, got %T", body["secrets"])
				}
				if sec["token"] != "new-secret" {
					t.Errorf("secrets.token = %v, want %q", sec["token"], "new-secret")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildConnectorUpdateBody(tc.inputs)
			tc.checks(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildRuleCreateBody
// ---------------------------------------------------------------------------

func TestBuildRuleCreateBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs RuleInputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "full inputs with all fields",
			inputs: RuleInputs{
				Name:       "my-rule",
				Consumer:   "alerts",
				RuleTypeID: ".es-query",
				Schedule:   `{"interval":"1m"}`,
				Params:     `{"threshold":100}`,
				Actions:    strPtr(`[{"group":"default","id":"conn-1","params":{}}]`),
				Enabled:    boolPtr(true),
				Tags:       []string{"production", "critical"},
				Throttle:   strPtr("5m"),
				NotifyWhen: strPtr("onActiveAlert"),
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["name"] != "my-rule" {
					t.Errorf("name = %v, want %q", body["name"], "my-rule")
				}
				if body["consumer"] != "alerts" {
					t.Errorf("consumer = %v, want %q", body["consumer"], "alerts")
				}
				if body["rule_type_id"] != ".es-query" {
					t.Errorf("rule_type_id = %v, want %q", body["rule_type_id"], ".es-query")
				}

				schedule, ok := body["schedule"].(map[string]any)
				if !ok {
					t.Fatalf("schedule is not a map, got %T", body["schedule"])
				}
				if schedule["interval"] != "1m" {
					t.Errorf("schedule.interval = %v, want %q", schedule["interval"], "1m")
				}

				params, ok := body["params"].(map[string]any)
				if !ok {
					t.Fatalf("params is not a map, got %T", body["params"])
				}
				if params["threshold"] != float64(100) {
					t.Errorf("params.threshold = %v, want %v", params["threshold"], 100)
				}

				actions, ok := body["actions"].([]any)
				if !ok {
					t.Fatalf("actions is not a slice, got %T", body["actions"])
				}
				if len(actions) != 1 {
					t.Errorf("len(actions) = %d, want 1", len(actions))
				}

				if body["enabled"] != true {
					t.Errorf("enabled = %v, want true", body["enabled"])
				}

				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags is not []string, got %T", body["tags"])
				}
				if len(tags) != 2 || tags[0] != "production" || tags[1] != "critical" {
					t.Errorf("tags = %v, want [production critical]", tags)
				}

				if body["throttle"] != "5m" {
					t.Errorf("throttle = %v, want %q", body["throttle"], "5m")
				}
				if body["notify_when"] != "onActiveAlert" {
					t.Errorf("notify_when = %v, want %q", body["notify_when"], "onActiveAlert")
				}
			},
		},
		{
			name: "minimal inputs - no optional fields",
			inputs: RuleInputs{
				Name:       "minimal-rule",
				Consumer:   "siem",
				RuleTypeID: ".index-threshold",
				Schedule:   `{"interval":"5m"}`,
				Params:     `{"index":"logs-*"}`,
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["name"] != "minimal-rule" {
					t.Errorf("name = %v, want %q", body["name"], "minimal-rule")
				}
				if body["consumer"] != "siem" {
					t.Errorf("consumer = %v, want %q", body["consumer"], "siem")
				}
				if body["rule_type_id"] != ".index-threshold" {
					t.Errorf("rule_type_id = %v, want %q", body["rule_type_id"], ".index-threshold")
				}
				if _, ok := body["actions"]; ok {
					t.Error("actions should not be present when nil")
				}
				if _, ok := body["enabled"]; ok {
					t.Error("enabled should not be present when nil")
				}
				if _, ok := body["tags"]; ok {
					t.Error("tags should not be present when empty")
				}
				if _, ok := body["throttle"]; ok {
					t.Error("throttle should not be present when nil")
				}
				if _, ok := body["notify_when"]; ok {
					t.Error("notify_when should not be present when nil")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildRuleCreateBody(tc.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tc.checks(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildRuleUpdateBody
// ---------------------------------------------------------------------------

func TestBuildRuleUpdateBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs RuleInputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "should NOT include consumer or rule_type_id",
			inputs: RuleInputs{
				Name:       "my-rule",
				Consumer:   "alerts",
				RuleTypeID: ".es-query",
				Schedule:   `{"interval":"1m"}`,
				Params:     `{"threshold":100}`,
				Enabled:    boolPtr(false),
				Tags:       []string{"updated"},
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["name"] != "my-rule" {
					t.Errorf("name = %v, want %q", body["name"], "my-rule")
				}
				if _, ok := body["consumer"]; ok {
					t.Error("consumer should NOT be present in update body (immutable)")
				}
				if _, ok := body["rule_type_id"]; ok {
					t.Error("rule_type_id should NOT be present in update body (immutable)")
				}

				// enabled should NOT be in update body (not set by buildRuleUpdateBody)
				if _, ok := body["enabled"]; ok {
					t.Error("enabled should NOT be present in update body")
				}

				// schedule and params should still be present
				if _, ok := body["schedule"]; !ok {
					t.Error("schedule should be present in update body")
				}
				if _, ok := body["params"]; !ok {
					t.Error("params should be present in update body")
				}

				tags, ok := body["tags"].([]string)
				if !ok {
					t.Fatalf("tags is not []string, got %T", body["tags"])
				}
				if len(tags) != 1 || tags[0] != "updated" {
					t.Errorf("tags = %v, want [updated]", tags)
				}
			},
		},
		{
			name: "with actions, throttle and notify_when",
			inputs: RuleInputs{
				Name:       "rule-2",
				Consumer:   "siem",
				RuleTypeID: ".index-threshold",
				Schedule:   `{"interval":"10m"}`,
				Params:     `{}`,
				Actions:    strPtr(`[{"id":"act-1"}]`),
				Throttle:   strPtr("1h"),
				NotifyWhen: strPtr("onThrottleInterval"),
			},
			checks: func(t *testing.T, body map[string]any) {
				if _, ok := body["consumer"]; ok {
					t.Error("consumer should NOT be present in update body")
				}
				if _, ok := body["rule_type_id"]; ok {
					t.Error("rule_type_id should NOT be present in update body")
				}
				if body["throttle"] != "1h" {
					t.Errorf("throttle = %v, want %q", body["throttle"], "1h")
				}
				if body["notify_when"] != "onThrottleInterval" {
					t.Errorf("notify_when = %v, want %q", body["notify_when"], "onThrottleInterval")
				}
				actions, ok := body["actions"].([]any)
				if !ok {
					t.Fatalf("actions is not a slice, got %T", body["actions"])
				}
				if len(actions) != 1 {
					t.Errorf("len(actions) = %d, want 1", len(actions))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildRuleUpdateBody(tc.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tc.checks(t, body)
		})
	}
}

// ---------------------------------------------------------------------------
// buildMaintenanceWindowBody
// ---------------------------------------------------------------------------

func TestBuildMaintenanceWindowBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs MaintenanceWindowInputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "full inputs with all fields",
			inputs: MaintenanceWindowInputs{
				Title:       "Nightly Maintenance",
				Enabled:     boolPtr(true),
				Schedule:    `{"rrule":"FREQ=DAILY;INTERVAL=1","duration":"2h"}`,
				ScopedQuery: strPtr(`[{"kql":"host.name:web-*"}]`),
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["title"] != "Nightly Maintenance" {
					t.Errorf("title = %v, want %q", body["title"], "Nightly Maintenance")
				}
				if body["enabled"] != true {
					t.Errorf("enabled = %v, want true", body["enabled"])
				}

				schedule, ok := body["schedule"].(map[string]any)
				if !ok {
					t.Fatalf("schedule is not a map, got %T", body["schedule"])
				}
				if schedule["rrule"] != "FREQ=DAILY;INTERVAL=1" {
					t.Errorf("schedule.rrule = %v, want %q", schedule["rrule"], "FREQ=DAILY;INTERVAL=1")
				}

				scopedQuery, ok := body["scoped_query"].([]any)
				if !ok {
					t.Fatalf("scoped_query is not a slice, got %T", body["scoped_query"])
				}
				if len(scopedQuery) != 1 {
					t.Errorf("len(scoped_query) = %d, want 1", len(scopedQuery))
				}
			},
		},
		{
			name: "minimal - no optional fields",
			inputs: MaintenanceWindowInputs{
				Title:    "Simple Window",
				Schedule: `{"rrule":"FREQ=WEEKLY"}`,
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["title"] != "Simple Window" {
					t.Errorf("title = %v, want %q", body["title"], "Simple Window")
				}
				if _, ok := body["enabled"]; ok {
					t.Error("enabled should not be present when nil")
				}
				if _, ok := body["scoped_query"]; ok {
					t.Error("scoped_query should not be present when nil")
				}
				// schedule should always be present
				if _, ok := body["schedule"]; !ok {
					t.Error("schedule should always be present")
				}
			},
		},
		{
			name: "enabled set to false",
			inputs: MaintenanceWindowInputs{
				Title:    "Disabled Window",
				Enabled:  boolPtr(false),
				Schedule: `{"rrule":"FREQ=MONTHLY"}`,
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["enabled"] != false {
					t.Errorf("enabled = %v, want false", body["enabled"])
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildMaintenanceWindowBody(tc.inputs)
			if err != nil {
				t.Fatal(err)
			}
			tc.checks(t, body)
		})
	}
}
