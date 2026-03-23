// Package cloud implements Elastic Cloud resource management.
package cloud

import (
	"testing"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestBuildInvitationBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs OrganizationMemberInputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "minimal",
			inputs: OrganizationMemberInputs{
				OrganizationID: "org-123",
				Email:          "user@example.com",
			},
			checks: func(t *testing.T, body map[string]any) {
				emails := body["emails"].([]string)
				if len(emails) != 1 || emails[0] != "user@example.com" {
					t.Errorf("expected emails=[user@example.com], got %v", emails)
				}
				if _, ok := body["expires_in"]; ok {
					t.Error("expires_in should not be set")
				}
				if _, ok := body["role_assignments"]; ok {
					t.Error("role_assignments should not be set")
				}
			},
		},
		{
			name: "with_expiry",
			inputs: OrganizationMemberInputs{
				OrganizationID: "org-123",
				Email:          "user@example.com",
				ExpiresIn:      strPtr("72h"),
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["expires_in"] != "72h" {
					t.Errorf("expected expires_in=72h, got %v", body["expires_in"])
				}
			},
		},
		{
			name: "org_owner_and_billing_admin",
			inputs: OrganizationMemberInputs{
				OrganizationID:    "org-123",
				Email:             "admin@example.com",
				OrganizationOwner: boolPtr(true),
				BillingAdmin:      boolPtr(true),
			},
			checks: func(t *testing.T, body map[string]any) {
				roles := body["role_assignments"].(map[string]any)
				orgRoles := roles["organization"].([]map[string]any)
				if len(orgRoles) != 2 {
					t.Fatalf("expected 2 org roles, got %d", len(orgRoles))
				}
				if orgRoles[0]["role_id"] != "organization-admin" {
					t.Errorf("expected organization-admin, got %v", orgRoles[0]["role_id"])
				}
				if orgRoles[1]["role_id"] != "billing-admin" {
					t.Errorf("expected billing-admin, got %v", orgRoles[1]["role_id"])
				}
			},
		},
		{
			name: "deployment_role_all",
			inputs: OrganizationMemberInputs{
				OrganizationID:    "org-123",
				Email:             "dev@example.com",
				DeploymentRoleAll: strPtr("editor"),
			},
			checks: func(t *testing.T, body map[string]any) {
				roles := body["role_assignments"].(map[string]any)
				depRoles := roles["deployment"].([]map[string]any)
				if len(depRoles) != 1 {
					t.Fatalf("expected 1 deployment role, got %d", len(depRoles))
				}
				if depRoles[0]["role_id"] != "deployment-editor" {
					t.Errorf("expected deployment-editor, got %v", depRoles[0]["role_id"])
				}
				if depRoles[0]["all"] != true {
					t.Error("expected all=true")
				}
			},
		},
		{
			name: "per_deployment_roles",
			inputs: OrganizationMemberInputs{
				OrganizationID: "org-123",
				Email:          "dev@example.com",
				DeploymentRoles: []DeploymentRoleSpec{
					{DeploymentID: "dep-1", Role: "admin"},
					{DeploymentID: "dep-2", Role: "viewer"},
				},
			},
			checks: func(t *testing.T, body map[string]any) {
				roles := body["role_assignments"].(map[string]any)
				depRoles := roles["deployment"].([]map[string]any)
				if len(depRoles) != 2 {
					t.Fatalf("expected 2 deployment roles, got %d", len(depRoles))
				}
				if depRoles[0]["role_id"] != "deployment-admin" {
					t.Errorf("expected deployment-admin, got %v", depRoles[0]["role_id"])
				}
				ids0 := depRoles[0]["deployment_ids"].([]string)
				if ids0[0] != "dep-1" {
					t.Errorf("expected dep-1, got %v", ids0[0])
				}
				if depRoles[1]["role_id"] != "deployment-viewer" {
					t.Errorf("expected deployment-viewer, got %v", depRoles[1]["role_id"])
				}
			},
		},
		{
			name: "serverless_elasticsearch_roles",
			inputs: OrganizationMemberInputs{
				OrganizationID:       "org-123",
				Email:                "dev@example.com",
				ElasticsearchRoleAll: strPtr("admin"),
				ElasticsearchRoles: []ProjectRoleSpec{
					{ProjectID: "proj-1", Role: "developer"},
				},
			},
			checks: func(t *testing.T, body map[string]any) {
				roles := body["role_assignments"].(map[string]any)
				project := roles["project"].(map[string]any)
				esRoles := project["elasticsearch"].([]map[string]any)
				if len(esRoles) != 2 {
					t.Fatalf("expected 2 elasticsearch roles, got %d", len(esRoles))
				}
				if esRoles[0]["role_id"] != "admin" {
					t.Errorf("expected admin, got %v", esRoles[0]["role_id"])
				}
				if esRoles[0]["all"] != true {
					t.Error("expected all=true for role_all")
				}
				if esRoles[1]["role_id"] != "developer" {
					t.Errorf("expected developer, got %v", esRoles[1]["role_id"])
				}
				ids := esRoles[1]["project_ids"].([]string)
				if ids[0] != "proj-1" {
					t.Errorf("expected proj-1, got %v", ids[0])
				}
			},
		},
		{
			name: "all_serverless_types",
			inputs: OrganizationMemberInputs{
				OrganizationID:       "org-123",
				Email:                "dev@example.com",
				ElasticsearchRoleAll: strPtr("admin"),
				ObservabilityRoleAll: strPtr("editor"),
				SecurityRoleAll:      strPtr("viewer"),
			},
			checks: func(t *testing.T, body map[string]any) {
				roles := body["role_assignments"].(map[string]any)
				project := roles["project"].(map[string]any)
				if _, ok := project["elasticsearch"]; !ok {
					t.Error("elasticsearch key should exist")
				}
				if _, ok := project["observability"]; !ok {
					t.Error("observability key should exist")
				}
				if _, ok := project["security"]; !ok {
					t.Error("security key should exist")
				}
			},
		},
		{
			name: "full_complex_invitation",
			inputs: OrganizationMemberInputs{
				OrganizationID:    "org-456",
				Email:             "team@example.com",
				ExpiresIn:         strPtr("168h"),
				OrganizationOwner: boolPtr(false),
				BillingAdmin:      boolPtr(true),
				DeploymentRoleAll: strPtr("viewer"),
				DeploymentRoles: []DeploymentRoleSpec{
					{DeploymentID: "dep-prod", Role: "admin"},
				},
				ObservabilityRoles: []ProjectRoleSpec{
					{ProjectID: "obs-1", Role: "editor"},
				},
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["expires_in"] != "168h" {
					t.Errorf("expected 168h, got %v", body["expires_in"])
				}
				emails := body["emails"].([]string)
				if emails[0] != "team@example.com" {
					t.Errorf("expected team@example.com, got %s", emails[0])
				}

				roles := body["role_assignments"].(map[string]any)

				// Billing admin only (org owner is false)
				orgRoles := roles["organization"].([]map[string]any)
				if len(orgRoles) != 1 {
					t.Fatalf("expected 1 org role (billing only), got %d", len(orgRoles))
				}
				if orgRoles[0]["role_id"] != "billing-admin" {
					t.Errorf("expected billing-admin, got %v", orgRoles[0]["role_id"])
				}

				// 2 deployment roles: all=viewer + dep-prod=admin
				depRoles := roles["deployment"].([]map[string]any)
				if len(depRoles) != 2 {
					t.Fatalf("expected 2 deployment roles, got %d", len(depRoles))
				}

				// Observability project role
				project := roles["project"].(map[string]any)
				obsRoles := project["observability"].([]map[string]any)
				if len(obsRoles) != 1 {
					t.Fatalf("expected 1 observability role, got %d", len(obsRoles))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildInvitationBody(tt.inputs)
			tt.checks(t, body)
		})
	}
}

func TestResolveDeploymentRoleID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"admin", "deployment-admin"},
		{"editor", "deployment-editor"},
		{"viewer", "deployment-viewer"},
		{"custom-role-id", "custom-role-id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveDeploymentRoleID(tt.input)
			if got != tt.expected {
				t.Errorf("resolveDeploymentRoleID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildRulesetBody(t *testing.T) {
	tests := []struct {
		name   string
		inputs TrafficFilterInputs
		checks func(t *testing.T, body map[string]any)
	}{
		{
			name: "ip_allowlist",
			inputs: TrafficFilterInputs{
				Name:             "office-ips",
				Type:             "ip",
				Region:           "azure-eastus2",
				IncludeByDefault: false,
				Rules: []TrafficFilterRule{
					{Source: strPtr("10.0.0.0/8"), Description: strPtr("Private network")},
					{Source: strPtr("203.0.113.42")},
				},
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["name"] != "office-ips" {
					t.Errorf("expected name=office-ips, got %v", body["name"])
				}
				if body["type"] != "ip" {
					t.Errorf("expected type=ip, got %v", body["type"])
				}
				if body["region"] != "azure-eastus2" {
					t.Errorf("expected region=azure-eastus2, got %v", body["region"])
				}
				if body["include_by_default"] != false {
					t.Error("expected include_by_default=false")
				}
				rules := body["rules"].([]map[string]any)
				if len(rules) != 2 {
					t.Fatalf("expected 2 rules, got %d", len(rules))
				}
				if rules[0]["source"] != "10.0.0.0/8" {
					t.Errorf("expected source=10.0.0.0/8, got %v", rules[0]["source"])
				}
				if rules[0]["description"] != "Private network" {
					t.Errorf("expected description='Private network', got %v", rules[0]["description"])
				}
				if rules[1]["source"] != "203.0.113.42" {
					t.Errorf("expected source=203.0.113.42, got %v", rules[1]["source"])
				}
			},
		},
		{
			name: "azure_private_endpoint",
			inputs: TrafficFilterInputs{
				Name:   "azure-pl",
				Type:   "azure_private_endpoint",
				Region: "azure-eastus2",
				Rules: []TrafficFilterRule{
					{
						AzureEndpointName: strPtr("my-private-endpoint"),
						AzureEndpointGUID: strPtr("7c0f05e4-e32b-4b10-a246-7b77f7dcc63c"),
					},
				},
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["type"] != "azure_private_endpoint" {
					t.Errorf("expected type=azure_private_endpoint, got %v", body["type"])
				}
				rules := body["rules"].([]map[string]any)
				if len(rules) != 1 {
					t.Fatalf("expected 1 rule, got %d", len(rules))
				}
				if rules[0]["azure_endpoint_name"] != "my-private-endpoint" {
					t.Errorf("unexpected azure_endpoint_name: %v", rules[0]["azure_endpoint_name"])
				}
				if rules[0]["azure_endpoint_guid"] != "7c0f05e4-e32b-4b10-a246-7b77f7dcc63c" {
					t.Errorf("unexpected azure_endpoint_guid: %v", rules[0]["azure_endpoint_guid"])
				}
			},
		},
		{
			name: "vpce",
			inputs: TrafficFilterInputs{
				Name:   "aws-pl",
				Type:   "vpce",
				Region: "us-east-1",
				Rules: []TrafficFilterRule{
					{Source: strPtr("vpce-00000000000")},
				},
			},
			checks: func(t *testing.T, body map[string]any) {
				rules := body["rules"].([]map[string]any)
				if rules[0]["source"] != "vpce-00000000000" {
					t.Errorf("expected vpce source, got %v", rules[0]["source"])
				}
			},
		},
		{
			name: "egress_firewall",
			inputs: TrafficFilterInputs{
				Name:   "egress",
				Type:   "egress_firewall",
				Region: "azure-eastus2",
				Rules: []TrafficFilterRule{
					{
						Description: strPtr("Allow HTTPS to backend"),
						EgressRule: &TrafficEgressRule{
							Target:   "10.0.1.0/24",
							Protocol: "tcp",
							Ports:    []int{443, 9243},
						},
					},
				},
			},
			checks: func(t *testing.T, body map[string]any) {
				rules := body["rules"].([]map[string]any)
				if len(rules) != 1 {
					t.Fatalf("expected 1 rule, got %d", len(rules))
				}
				egress := rules[0]["egress_rule"].(map[string]any)
				if egress["target"] != "10.0.1.0/24" {
					t.Errorf("expected target=10.0.1.0/24, got %v", egress["target"])
				}
				if egress["protocol"] != "tcp" {
					t.Errorf("expected protocol=tcp, got %v", egress["protocol"])
				}
				ports := egress["ports"].([]int)
				if len(ports) != 2 || ports[0] != 443 || ports[1] != 9243 {
					t.Errorf("expected ports=[443,9243], got %v", ports)
				}
			},
		},
		{
			name: "with_description_and_include_by_default",
			inputs: TrafficFilterInputs{
				Name:             "default-filter",
				Type:             "ip",
				Region:           "us-east-1",
				Description:      strPtr("Auto-applied to all deployments"),
				IncludeByDefault: true,
				Rules: []TrafficFilterRule{
					{Source: strPtr("0.0.0.0/0")},
				},
			},
			checks: func(t *testing.T, body map[string]any) {
				if body["description"] != "Auto-applied to all deployments" {
					t.Errorf("unexpected description: %v", body["description"])
				}
				if body["include_by_default"] != true {
					t.Error("expected include_by_default=true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := buildRulesetBody(tt.inputs)
			tt.checks(t, body)
		})
	}
}

func TestBuildProjectRoles(t *testing.T) {
	tests := []struct {
		name       string
		roleAll    *string
		perProject []ProjectRoleSpec
		orgID      string
		wantLen    int
	}{
		{"nil_all_no_projects", nil, nil, "org-1", 0},
		{"empty_all_no_projects", strPtr(""), nil, "org-1", 0},
		{"role_all_only", strPtr("admin"), nil, "org-1", 1},
		{"per_project_only", nil, []ProjectRoleSpec{{ProjectID: "p1", Role: "editor"}}, "org-1", 1},
		{"both", strPtr("viewer"), []ProjectRoleSpec{{ProjectID: "p1", Role: "admin"}}, "org-1", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := buildProjectRoles(tt.roleAll, tt.perProject, tt.orgID)
			if len(roles) != tt.wantLen {
				t.Errorf("expected %d roles, got %d", tt.wantLen, len(roles))
			}
		})
	}
}
