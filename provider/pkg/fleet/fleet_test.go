package fleet

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

// --- AgentPolicy body builder tests ---

func TestBuildAgentPolicyBody_Minimal(t *testing.T) {
	body, err := buildAgentPolicyBody(AgentPolicyInputs{
		Name: "my-policy",
	})
	if err != nil {
		t.Fatal(err)
	}

	if body["name"] != "my-policy" {
		t.Errorf("name = %v, want %q", body["name"], "my-policy")
	}
	for _, key := range []string{
		"namespace", "description", "monitoring_enabled",
		"data_output_id", "monitoring_output_id", "fleet_server_host_id",
		"agent_features", "is_protected", "keep_monitoring_alive",
		"global_data_tags",
	} {
		if _, ok := body[key]; ok {
			t.Errorf("%s should not be present when nil/unset", key)
		}
	}
}

func TestBuildAgentPolicyBody_AllFields(t *testing.T) {
	body, err := buildAgentPolicyBody(AgentPolicyInputs{
		Name:                "full-policy",
		Namespace:           strPtr("custom"),
		Description:         strPtr("A test policy"),
		MonitorLogs:         boolPtr(true),
		MonitorMetrics:      boolPtr(true),
		DataOutputID:        strPtr("out-1"),
		MonitoringOutputID:  strPtr("out-2"),
		FleetServerHostID:   strPtr("host-1"),
		AgentFeatures:       strPtr(`[{"name":"fqdn","enabled":true}]`),
		IsProtected:         boolPtr(true),
		KeepMonitoringAlive: boolPtr(false),
		GlobalDataTags:      strPtr(`[{"name":"env","value":"prod"}]`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if body["name"] != "full-policy" {
		t.Errorf("name = %v, want %q", body["name"], "full-policy")
	}
	if body["namespace"] != "custom" {
		t.Errorf("namespace = %v, want %q", body["namespace"], "custom")
	}
	if body["description"] != "A test policy" {
		t.Errorf("description = %v, want %q", body["description"], "A test policy")
	}
	if body["data_output_id"] != "out-1" {
		t.Errorf("data_output_id = %v, want %q", body["data_output_id"], "out-1")
	}
	if body["monitoring_output_id"] != "out-2" {
		t.Errorf("monitoring_output_id = %v, want %q", body["monitoring_output_id"], "out-2")
	}
	if body["fleet_server_host_id"] != "host-1" {
		t.Errorf("fleet_server_host_id = %v, want %q", body["fleet_server_host_id"], "host-1")
	}
	if body["is_protected"] != true {
		t.Errorf("is_protected = %v, want true", body["is_protected"])
	}
	if body["keep_monitoring_alive"] != false {
		t.Errorf("keep_monitoring_alive = %v, want false", body["keep_monitoring_alive"])
	}

	// Check monitoring_enabled
	monitoring, ok := body["monitoring_enabled"].([]string)
	if !ok {
		t.Fatalf("monitoring_enabled is not []string, got %T", body["monitoring_enabled"])
	}
	if len(monitoring) != 2 {
		t.Fatalf("monitoring_enabled length = %d, want 2", len(monitoring))
	}

	// Check agent_features is parsed JSON
	if body["agent_features"] == nil {
		t.Fatal("agent_features should be present")
	}

	// Check global_data_tags is parsed JSON
	if body["global_data_tags"] == nil {
		t.Fatal("global_data_tags should be present")
	}
}

func TestBuildMonitoringEnabled(t *testing.T) {
	tests := []struct {
		name    string
		logs    *bool
		metrics *bool
		want    []string
	}{
		{
			name: "both nil",
			want: nil,
		},
		{
			name: "logs only",
			logs: boolPtr(true),
			want: []string{"logs"},
		},
		{
			name:    "metrics only",
			metrics: boolPtr(true),
			want:    []string{"metrics"},
		},
		{
			name:    "both enabled",
			logs:    boolPtr(true),
			metrics: boolPtr(true),
			want:    []string{"logs", "metrics"},
		},
		{
			name:    "both disabled",
			logs:    boolPtr(false),
			metrics: boolPtr(false),
			want:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildMonitoringEnabled(tc.logs, tc.metrics)
			if len(got) != len(tc.want) {
				t.Fatalf("buildMonitoringEnabled() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("buildMonitoringEnabled()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// --- IntegrationPolicy body builder tests ---

func TestBuildIntegrationPolicyBody_Minimal(t *testing.T) {
	body, err := buildIntegrationPolicyBody(IntegrationPolicyInputs{
		Name:               "my-integration",
		AgentPolicyID:      "agent-1",
		IntegrationName:    "system",
		IntegrationVersion: "1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	if body["name"] != "my-integration" {
		t.Errorf("name = %v, want %q", body["name"], "my-integration")
	}
	if body["policy_id"] != "agent-1" {
		t.Errorf("policy_id = %v, want %q", body["policy_id"], "agent-1")
	}

	pkg, ok := body["package"].(map[string]any)
	if !ok {
		t.Fatalf("package is not map[string]any, got %T", body["package"])
	}
	if pkg["name"] != "system" {
		t.Errorf("package.name = %v, want %q", pkg["name"], "system")
	}
	if pkg["version"] != "1.0.0" {
		t.Errorf("package.version = %v, want %q", pkg["version"], "1.0.0")
	}

	for _, key := range []string{"namespace", "description", "inputs", "vars"} {
		if _, ok := body[key]; ok {
			t.Errorf("%s should not be present when nil", key)
		}
	}
}

func TestBuildIntegrationPolicyBody_AllFields(t *testing.T) {
	body, err := buildIntegrationPolicyBody(IntegrationPolicyInputs{
		Name:               "full-integration",
		Namespace:          strPtr("custom"),
		Description:        strPtr("Integration description"),
		AgentPolicyID:      "agent-2",
		IntegrationName:    "nginx",
		IntegrationVersion: "2.0.0",
		Input:              strPtr(`[{"type":"logfile","enabled":true}]`),
		Vars:               strPtr(`{"paths":["/var/log/nginx/*.log"]}`),
		Force:              boolPtr(true),
	})
	if err != nil {
		t.Fatal(err)
	}

	if body["name"] != "full-integration" {
		t.Errorf("name = %v, want %q", body["name"], "full-integration")
	}
	if body["namespace"] != "custom" {
		t.Errorf("namespace = %v, want %q", body["namespace"], "custom")
	}
	if body["description"] != "Integration description" {
		t.Errorf("description = %v, want %q", body["description"], "Integration description")
	}
	if body["force"] != true {
		t.Errorf("force = %v, want true", body["force"])
	}

	// inputs should be parsed JSON
	if body["inputs"] == nil {
		t.Fatal("inputs should be present")
	}

	// vars should be parsed JSON
	if body["vars"] == nil {
		t.Fatal("vars should be present")
	}
}

// --- Output body builder tests ---

func TestBuildOutputBody_Minimal(t *testing.T) {
	body, err := buildOutputBody(OutputInputs{
		Name:       "my-output",
		OutputType: "elasticsearch",
	})
	if err != nil {
		t.Fatal(err)
	}

	if body["name"] != "my-output" {
		t.Errorf("name = %v, want %q", body["name"], "my-output")
	}
	if body["type"] != "elasticsearch" {
		t.Errorf("type = %v, want %q", body["type"], "elasticsearch")
	}

	for _, key := range []string{"is_default", "is_default_monitoring", "hosts", "config_yaml", "ssl"} {
		if _, ok := body[key]; ok {
			t.Errorf("%s should not be present when nil/empty", key)
		}
	}
}

func TestBuildOutputBody_AllFields(t *testing.T) {
	body, err := buildOutputBody(OutputInputs{
		Name:                "full-output",
		OutputType:          "logstash",
		DefaultIntegrations: boolPtr(true),
		DefaultMonitoring:   boolPtr(false),
		Hosts:               []string{"logstash.example.com:5044"},
		ConfigYaml:          strPtr("timeout: 30s"),
		Ssl:                 strPtr(`{"verification_mode":"full"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if body["name"] != "full-output" {
		t.Errorf("name = %v, want %q", body["name"], "full-output")
	}
	if body["type"] != "logstash" {
		t.Errorf("type = %v, want %q", body["type"], "logstash")
	}
	if body["is_default"] != true {
		t.Errorf("is_default = %v, want true", body["is_default"])
	}
	if body["is_default_monitoring"] != false {
		t.Errorf("is_default_monitoring = %v, want false", body["is_default_monitoring"])
	}
	if body["config_yaml"] != "timeout: 30s" {
		t.Errorf("config_yaml = %v, want %q", body["config_yaml"], "timeout: 30s")
	}

	hosts, ok := body["hosts"].([]string)
	if !ok {
		t.Fatalf("hosts is not []string, got %T", body["hosts"])
	}
	if len(hosts) != 1 || hosts[0] != "logstash.example.com:5044" {
		t.Errorf("hosts = %v, want [logstash.example.com:5044]", hosts)
	}

	// ssl should be parsed JSON
	if body["ssl"] == nil {
		t.Fatal("ssl should be present")
	}
}

func TestBuildOutputBody_SslRoundTrip(t *testing.T) {
	body, err := buildOutputBody(OutputInputs{
		Name:       "ssl-output",
		OutputType: "elasticsearch",
		Ssl:        strPtr(`{"verification_mode":"certificate","certificate":"/path/to/cert"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	ssl, ok := decoded["ssl"].(map[string]any)
	if !ok {
		t.Fatalf("ssl is not map[string]any after round-trip, got %T", decoded["ssl"])
	}
	if ssl["verification_mode"] != "certificate" {
		t.Errorf("ssl.verification_mode = %v, want %q", ssl["verification_mode"], "certificate")
	}
}
