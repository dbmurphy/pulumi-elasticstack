package cloud

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// OrganizationMember invites a user to an Elastic Cloud organization with specified roles.
type OrganizationMember struct{}

// OrganizationMemberInputs defines the input properties for an organization member invitation.
type OrganizationMemberInputs struct {
	OrganizationID string `pulumi:"organizationId"`
	Email          string `pulumi:"email"`

	// Organization-level toggles
	OrganizationOwner *bool `pulumi:"organizationOwner,optional"`
	BillingAdmin      *bool `pulumi:"billingAdmin,optional"`

	// Hosted deployment roles
	DeploymentRoleAll *string              `pulumi:"deploymentRoleAll,optional"`
	DeploymentRoles   []DeploymentRoleSpec `pulumi:"deploymentRoles,optional"`

	// Serverless project roles
	ElasticsearchRoleAll *string           `pulumi:"elasticsearchRoleAll,optional"`
	ElasticsearchRoles   []ProjectRoleSpec `pulumi:"elasticsearchRoles,optional"`
	ObservabilityRoleAll *string           `pulumi:"observabilityRoleAll,optional"`
	ObservabilityRoles   []ProjectRoleSpec `pulumi:"observabilityRoles,optional"`
	SecurityRoleAll      *string           `pulumi:"securityRoleAll,optional"`
	SecurityRoles        []ProjectRoleSpec `pulumi:"securityRoles,optional"`

	ExpiresIn *string `pulumi:"expiresIn,optional"`
}

// DeploymentRoleSpec assigns a role to a specific hosted deployment.
type DeploymentRoleSpec struct {
	DeploymentID string `pulumi:"deploymentId"`
	Role         string `pulumi:"role"`
}

// ProjectRoleSpec assigns a role to a specific serverless project.
type ProjectRoleSpec struct {
	ProjectID string `pulumi:"projectId"`
	Role      string `pulumi:"role"`
}

// OrganizationMemberState defines the output state for an organization member invitation.
type OrganizationMemberState struct {
	OrganizationMemberInputs

	// Outputs
	InvitationToken string `pulumi:"invitationToken"`
	Accepted        bool   `pulumi:"accepted"`
}

var (
	_ infer.CustomDelete[OrganizationMemberState]                           = (*OrganizationMember)(nil)
	_ infer.CustomRead[OrganizationMemberInputs, OrganizationMemberState]   = (*OrganizationMember)(nil)
	_ infer.CustomUpdate[OrganizationMemberInputs, OrganizationMemberState] = (*OrganizationMember)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *OrganizationMember) Annotate(a infer.Annotator) {
	a.Describe(r, "Invites a user to an Elastic Cloud organization with specified role assignments. "+
		"Supports organization owner, billing admin, per-deployment roles (admin/editor/viewer), "+
		"and serverless project roles for Elasticsearch, Observability, and Security.")
	a.SetToken("cloud", "OrganizationMember")
}

// Annotate sets input property descriptions and defaults.
func (i *OrganizationMemberInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.OrganizationID, "The Elastic Cloud organization ID.")
	a.Describe(&i.Email, "The email address to invite.")
	a.Describe(&i.OrganizationOwner,
		"Grant Organization Owner role (highest privilege, can manage members and all resources).")
	a.SetDefault(&i.OrganizationOwner, false)
	a.Describe(&i.BillingAdmin, "Grant Billing Admin role (access to invoices and payment methods).")
	a.SetDefault(&i.BillingAdmin, false)
	a.Describe(&i.DeploymentRoleAll, "Role for ALL hosted deployments: 'admin', 'editor', or 'viewer'.")
	a.Describe(&i.DeploymentRoles, "Per-deployment role assignments for hosted deployments.")
	a.Describe(&i.ElasticsearchRoleAll, "Role for ALL serverless Elasticsearch projects.")
	a.Describe(&i.ElasticsearchRoles, "Per-project role assignments for serverless Elasticsearch projects.")
	a.Describe(&i.ObservabilityRoleAll, "Role for ALL serverless Observability projects.")
	a.Describe(&i.ObservabilityRoles, "Per-project role assignments for serverless Observability projects.")
	a.Describe(&i.SecurityRoleAll, "Role for ALL serverless Security projects.")
	a.Describe(&i.SecurityRoles, "Per-project role assignments for serverless Security projects.")
	a.Describe(&i.ExpiresIn, "Invitation expiration duration. Defaults to 3 days (72h).")
}

// Create provisions a new organization member invitation.
func (r *OrganizationMember) Create(
	ctx context.Context, req infer.CreateRequest[OrganizationMemberInputs],
) (infer.CreateResponse[OrganizationMemberState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.CreateResponse[OrganizationMemberState]{}, err
	}

	body := buildInvitationBody(req.Inputs)
	path := fmt.Sprintf("/organizations/%s/invitations", req.Inputs.OrganizationID)

	var result struct {
		Invitations []struct {
			Token string `json:"token"`
			Email string `json:"email"`
		} `json:"invitations"`
	}

	if err := cloudClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.CreateResponse[OrganizationMemberState]{},
			fmt.Errorf("failed to invite %s to organization: %w", req.Inputs.Email, err)
	}

	token := ""
	if len(result.Invitations) > 0 {
		token = result.Invitations[0].Token
	}

	id := req.Inputs.OrganizationID + "/" + req.Inputs.Email

	return infer.CreateResponse[OrganizationMemberState]{
		ID: id,
		Output: OrganizationMemberState{
			OrganizationMemberInputs: req.Inputs,
			InvitationToken:          token,
			Accepted:                 false,
		},
	}, nil
}

// Read fetches the current state of the organization member.
func (r *OrganizationMember) Read(
	ctx context.Context, req infer.ReadRequest[OrganizationMemberInputs, OrganizationMemberState],
) (infer.ReadResponse[OrganizationMemberInputs, OrganizationMemberState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.ReadResponse[OrganizationMemberInputs, OrganizationMemberState]{}, err
	}

	// Check if the user is now an org member (invitation accepted)
	membersPath := fmt.Sprintf("/organizations/%s/members", req.State.OrganizationID)
	var membersResult struct {
		Members []struct {
			Email string `json:"email"`
		} `json:"members"`
	}
	if err := cloudClient.GetJSON(ctx, membersPath, &membersResult); err == nil {
		for _, m := range membersResult.Members {
			if m.Email == req.State.Email {
				state := req.State
				state.Accepted = true
				return infer.ReadResponse[OrganizationMemberInputs, OrganizationMemberState]{
					ID: req.ID, Inputs: req.Inputs, State: state,
				}, nil
			}
		}
	}

	// Check if invitation is still pending
	invPath := fmt.Sprintf("/organizations/%s/invitations", req.State.OrganizationID)
	var invResult struct {
		Invitations []struct {
			Token   string `json:"token"`
			Email   string `json:"email"`
			Expired bool   `json:"expired"`
		} `json:"invitations"`
	}
	if err := cloudClient.GetJSON(ctx, invPath, &invResult); err == nil {
		for _, inv := range invResult.Invitations {
			if inv.Email == req.State.Email && !inv.Expired {
				return infer.ReadResponse[OrganizationMemberInputs, OrganizationMemberState](req), nil
			}
		}
	}

	return infer.ReadResponse[OrganizationMemberInputs, OrganizationMemberState]{ID: ""}, nil
}

// Update modifies an existing organization member invitation.
func (r *OrganizationMember) Update(
	ctx context.Context, req infer.UpdateRequest[OrganizationMemberInputs, OrganizationMemberState],
) (infer.UpdateResponse[OrganizationMemberState], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.UpdateResponse[OrganizationMemberState]{}, err
	}

	// Delete old invitation if not yet accepted
	if !req.State.Accepted && req.State.InvitationToken != "" {
		deletePath := fmt.Sprintf("/organizations/%s/invitations/%s",
			req.Inputs.OrganizationID, req.State.InvitationToken)
		_ = cloudClient.Delete(ctx, deletePath)
	}

	// Re-invite with new role assignments
	body := buildInvitationBody(req.Inputs)
	path := fmt.Sprintf("/organizations/%s/invitations", req.Inputs.OrganizationID)

	var result struct {
		Invitations []struct {
			Token string `json:"token"`
			Email string `json:"email"`
		} `json:"invitations"`
	}

	if err := cloudClient.PostJSON(ctx, path, body, &result); err != nil {
		return infer.UpdateResponse[OrganizationMemberState]{},
			fmt.Errorf("failed to re-invite %s: %w", req.Inputs.Email, err)
	}

	token := ""
	if len(result.Invitations) > 0 {
		token = result.Invitations[0].Token
	}

	return infer.UpdateResponse[OrganizationMemberState]{
		Output: OrganizationMemberState{
			OrganizationMemberInputs: req.Inputs,
			InvitationToken:          token,
			Accepted:                 false,
		},
	}, nil
}

// Delete removes the organization member invitation.
func (r *OrganizationMember) Delete(
	ctx context.Context, req infer.DeleteRequest[OrganizationMemberState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	cloudClient, err := cfg.CloudClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if !req.State.Accepted && req.State.InvitationToken != "" {
		path := fmt.Sprintf("/organizations/%s/invitations/%s",
			req.State.OrganizationID, req.State.InvitationToken)
		if err := cloudClient.Delete(ctx, path); err != nil {
			return infer.DeleteResponse{}, fmt.Errorf("failed to delete invitation: %w", err)
		}
	}

	return infer.DeleteResponse{}, nil
}

// resolveDeploymentRoleID maps user-friendly role names to Cloud API role_id values.
func resolveDeploymentRoleID(role string) string {
	switch role {
	case "admin":
		return "deployment-admin"
	case "editor":
		return "deployment-editor"
	case "viewer":
		return "deployment-viewer"
	default:
		return role // allow raw role_id passthrough
	}
}

func buildInvitationBody(inputs OrganizationMemberInputs) map[string]any {
	body := map[string]any{
		"emails": []string{inputs.Email},
	}

	if inputs.ExpiresIn != nil {
		body["expires_in"] = *inputs.ExpiresIn
	}

	roles := buildRoleAssignments(inputs)
	if len(roles) > 0 {
		body["role_assignments"] = roles
	}

	return body
}

func buildRoleAssignments(inputs OrganizationMemberInputs) map[string]any {
	roles := map[string]any{}

	// Organization-level roles
	var orgRoles []map[string]any
	if inputs.OrganizationOwner != nil && *inputs.OrganizationOwner {
		orgRoles = append(orgRoles, map[string]any{
			"role_id":         "organization-admin",
			"organization_id": inputs.OrganizationID,
		})
	}
	if inputs.BillingAdmin != nil && *inputs.BillingAdmin {
		orgRoles = append(orgRoles, map[string]any{
			"role_id":         "billing-admin",
			"organization_id": inputs.OrganizationID,
		})
	}
	if len(orgRoles) > 0 {
		roles["organization"] = orgRoles
	}

	// Hosted deployment roles
	var deploymentRoles []map[string]any
	if inputs.DeploymentRoleAll != nil && *inputs.DeploymentRoleAll != "" {
		deploymentRoles = append(deploymentRoles, map[string]any{
			"role_id":         resolveDeploymentRoleID(*inputs.DeploymentRoleAll),
			"organization_id": inputs.OrganizationID,
			"all":             true,
		})
	}
	for _, dr := range inputs.DeploymentRoles {
		deploymentRoles = append(deploymentRoles, map[string]any{
			"role_id":         resolveDeploymentRoleID(dr.Role),
			"organization_id": inputs.OrganizationID,
			"deployment_ids":  []string{dr.DeploymentID},
		})
	}
	if len(deploymentRoles) > 0 {
		roles["deployment"] = deploymentRoles
	}

	// Serverless project roles
	project := map[string]any{}

	esRoles := buildProjectRoles(inputs.ElasticsearchRoleAll, inputs.ElasticsearchRoles, inputs.OrganizationID)
	if len(esRoles) > 0 {
		project["elasticsearch"] = esRoles
	}

	obsRoles := buildProjectRoles(inputs.ObservabilityRoleAll, inputs.ObservabilityRoles, inputs.OrganizationID)
	if len(obsRoles) > 0 {
		project["observability"] = obsRoles
	}

	secRoles := buildProjectRoles(inputs.SecurityRoleAll, inputs.SecurityRoles, inputs.OrganizationID)
	if len(secRoles) > 0 {
		project["security"] = secRoles
	}

	if len(project) > 0 {
		roles["project"] = project
	}

	return roles
}

func buildProjectRoles(roleAll *string, perProject []ProjectRoleSpec, orgID string) []map[string]any {
	var roles []map[string]any

	if roleAll != nil && *roleAll != "" {
		roles = append(roles, map[string]any{
			"role_id":         *roleAll,
			"organization_id": orgID,
			"all":             true,
		})
	}

	for _, pr := range perProject {
		roles = append(roles, map[string]any{
			"role_id":         pr.Role,
			"organization_id": orgID,
			"project_ids":     []string{pr.ProjectID},
		})
	}

	return roles
}
