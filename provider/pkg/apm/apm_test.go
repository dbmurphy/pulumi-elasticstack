package apm

import (
	"encoding/json"
	"testing"
)

const testServiceName = "my-service"

func strPtr(s string) *string { return &s }

func TestBuildAgentConfigID(t *testing.T) {
	tests := []struct {
		name               string
		serviceName        string
		serviceEnvironment *string
		want               string
	}{
		{
			name:        "service name only",
			serviceName: testServiceName,
			want:        testServiceName,
		},
		{
			name:               "service name with environment",
			serviceName:        testServiceName,
			serviceEnvironment: strPtr("production"),
			want:               "my-service/production",
		},
		{
			name:               "empty environment treated as absent",
			serviceName:        testServiceName,
			serviceEnvironment: strPtr(""),
			want:               testServiceName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildAgentConfigID(tc.serviceName, tc.serviceEnvironment)
			if got != tc.want {
				t.Errorf("buildAgentConfigID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildAgentConfigBody(t *testing.T) {
	tests := []struct {
		name    string
		inputs  AgentConfigurationInputs
		checks  func(t *testing.T, body map[string]any)
		wantErr bool
	}{
		{
			name: "minimal - required fields only",
			inputs: AgentConfigurationInputs{
				ServiceName: testServiceName,
				Settings:    `{"transaction_sample_rate": "0.5"}`,
			},
			checks: func(t *testing.T, body map[string]any) {
				service, ok := body["service"].(map[string]any)
				if !ok {
					t.Fatalf("service is not map[string]any, got %T", body["service"])
				}
				if service["name"] != testServiceName {
					t.Errorf("service.name = %v, want %q", service["name"], testServiceName)
				}
				if _, ok := service["environment"]; ok {
					t.Error("service.environment should not be present when nil")
				}

				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatalf("settings is not map[string]any, got %T", body["settings"])
				}
				if settings["transaction_sample_rate"] != "0.5" {
					t.Errorf(
						"settings.transaction_sample_rate = %v, want %q",
						settings["transaction_sample_rate"],
						"0.5",
					)
				}

				if _, ok := body["agent_name"]; ok {
					t.Error("agent_name should not be present when nil")
				}
			},
		},
		{
			name: "all fields populated",
			inputs: AgentConfigurationInputs{
				ServiceName:        "frontend",
				ServiceEnvironment: strPtr("production"),
				AgentName:          strPtr("nodejs"),
				Settings:           `{"transaction_sample_rate": "1.0", "capture_body": "all"}`,
			},
			checks: func(t *testing.T, body map[string]any) {
				service, ok := body["service"].(map[string]any)
				if !ok {
					t.Fatalf("service is not map[string]any, got %T", body["service"])
				}
				if service["name"] != "frontend" {
					t.Errorf("service.name = %v, want %q", service["name"], "frontend")
				}
				if service["environment"] != "production" {
					t.Errorf("service.environment = %v, want %q", service["environment"], "production")
				}

				if body["agent_name"] != "nodejs" {
					t.Errorf("agent_name = %v, want %q", body["agent_name"], "nodejs")
				}

				settings, ok := body["settings"].(map[string]any)
				if !ok {
					t.Fatalf("settings is not map[string]any, got %T", body["settings"])
				}
				if settings["transaction_sample_rate"] != "1.0" {
					t.Errorf(
						"settings.transaction_sample_rate = %v, want %q",
						settings["transaction_sample_rate"],
						"1.0",
					)
				}
				if settings["capture_body"] != "all" {
					t.Errorf("settings.capture_body = %v, want %q", settings["capture_body"], "all")
				}
			},
		},
		{
			name: "invalid settings JSON returns error",
			inputs: AgentConfigurationInputs{
				ServiceName: "bad-service",
				Settings:    `{not valid json}`,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildAgentConfigBody(tc.inputs)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.checks(t, body)
		})
	}
}

func TestBuildAgentConfigBody_SettingsRoundTrip(t *testing.T) {
	inputs := AgentConfigurationInputs{
		ServiceName: "roundtrip-svc",
		Settings:    `{"transaction_sample_rate":"0.25","capture_headers":"false"}`,
	}

	body, err := buildAgentConfigBody(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	settings, ok := decoded["settings"].(map[string]any)
	if !ok {
		t.Fatalf("settings is not map[string]any after round-trip, got %T", decoded["settings"])
	}
	if settings["transaction_sample_rate"] != "0.25" {
		t.Errorf("transaction_sample_rate = %v, want %q", settings["transaction_sample_rate"], "0.25")
	}
}

func TestBuildAgentConfigDeleteBody(t *testing.T) {
	tests := []struct {
		name               string
		serviceName        string
		serviceEnvironment *string
		checks             func(t *testing.T, body map[string]any)
	}{
		{
			name:        "without environment",
			serviceName: testServiceName,
			checks: func(t *testing.T, body map[string]any) {
				service, ok := body["service"].(map[string]any)
				if !ok {
					t.Fatalf("service is not map[string]any, got %T", body["service"])
				}
				if service["name"] != testServiceName {
					t.Errorf("service.name = %v, want %q", service["name"], testServiceName)
				}
				if _, ok := service["environment"]; ok {
					t.Error("service.environment should not be present when nil")
				}
			},
		},
		{
			name:               "with environment",
			serviceName:        testServiceName,
			serviceEnvironment: strPtr("staging"),
			checks: func(t *testing.T, body map[string]any) {
				service, ok := body["service"].(map[string]any)
				if !ok {
					t.Fatalf("service is not map[string]any, got %T", body["service"])
				}
				if service["name"] != testServiceName {
					t.Errorf("service.name = %v, want %q", service["name"], testServiceName)
				}
				if service["environment"] != "staging" {
					t.Errorf("service.environment = %v, want %q", service["environment"], "staging")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildAgentConfigDeleteBody(tc.serviceName, tc.serviceEnvironment)
			tc.checks(t, body)
		})
	}
}
