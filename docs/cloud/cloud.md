# Elastic Cloud

Manage Elastic Cloud deployment passwords, organization member invitations, and network security (traffic filters).

## Prerequisites

The Cloud module requires an Elastic Cloud API key. Generate one at [cloud.elastic.co/account/keys](https://cloud.elastic.co/account/keys).

Set the `EC_API_KEY` environment variable:

```bash
export EC_API_KEY="your-elastic-cloud-api-key"
```

Optionally override the API endpoint (defaults to `https://api.elastic-cloud.com`):

```bash
export EC_ENDPOINT="https://api.elastic-cloud.com"
```

## Resources

- `elasticstack.cloud.DeploymentPassword`
- `elasticstack.cloud.OrganizationMember`
- `elasticstack.cloud.TrafficFilter`
- `elasticstack.cloud.TrafficFilterAssociation`

## DeploymentPassword

Reset the `elastic` user password for an Elastic Cloud deployment. The Cloud API generates a new password and returns it as a secret output. Useful when deploying clusters via Azure/GCP marketplace where the initial password isn't returned.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const password = new elasticstack.cloud.DeploymentPassword("cluster-pw", {
    deploymentId: "my-deployment-id",
    refId: "main-elasticsearch",
});

// Use the password to configure the provider
const provider = new elasticstack.Provider("es", {
    elasticsearch: {
        endpoints: ["https://my-cluster.es.eastus2.azure.elastic-cloud.com:9243"],
        username: "elastic",
        password: password.password,
    },
});

export const elasticPassword = password.password;
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `deploymentId` | string | Elastic Cloud deployment ID |
| `refId` | string | Elasticsearch resource ref ID (usually `"main-elasticsearch"`) |

### Key Outputs

| Output | Type | Description |
|--------|------|-------------|
| `password` | string (secret) | The generated password |
| `username` | string | Always `"elastic"` |

### Behavior

- **Create**: Calls the Cloud API to reset the password and stores the result.
- **Read**: Verifies the deployment still exists (password itself isn't re-fetched).
- **Update**: Any input change triggers replacement (new password reset).
- **Delete**: No-op (passwords can't be "un-reset").

## OrganizationMember

Invite users to your Elastic Cloud organization with fine-grained role assignments. Supports organization-level roles, per-deployment access, and serverless project roles.

### Basic Invitation

```typescript
const viewer = new elasticstack.cloud.OrganizationMember("viewer", {
    organizationId: "my-org-id",
    email: "viewer@company.com",
    deploymentRoleAll: "viewer",
});
```

### Organization Admin with Billing

```typescript
const admin = new elasticstack.cloud.OrganizationMember("admin", {
    organizationId: "my-org-id",
    email: "admin@company.com",
    organizationOwner: true,
    billingAdmin: true,
    deploymentRoleAll: "admin",
});
```

### Per-Deployment Roles

```typescript
const developer = new elasticstack.cloud.OrganizationMember("developer", {
    organizationId: "my-org-id",
    email: "dev@company.com",
    deploymentRoles: [
        { deploymentId: "prod-deployment-id", role: "viewer" },
        { deploymentId: "staging-deployment-id", role: "admin" },
        { deploymentId: "dev-deployment-id", role: "admin" },
    ],
});
```

### Serverless Project Roles

```typescript
const projectUser = new elasticstack.cloud.OrganizationMember("project-user", {
    organizationId: "my-org-id",
    email: "analyst@company.com",
    elasticsearchRoleAll: "viewer",
    observabilityRoleAll: "editor",
    securityRoles: [
        { projectId: "security-project-1", role: "admin" },
    ],
});
```

### Full Complex Invitation

```typescript
const teamLead = new elasticstack.cloud.OrganizationMember("team-lead", {
    organizationId: "my-org-id",
    email: "lead@company.com",
    billingAdmin: true,
    deploymentRoleAll: "editor",
    deploymentRoles: [
        { deploymentId: "prod-deployment-id", role: "admin" },
    ],
    elasticsearchRoleAll: "admin",
    observabilityRoleAll: "editor",
    securityRoleAll: "viewer",
    expiresIn: "168h",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `organizationId` | string | Elastic Cloud organization ID |
| `email` | string | Email address to invite |
| `organizationOwner` | bool | Grant organization owner role |
| `billingAdmin` | bool | Grant billing admin role |
| `deploymentRoleAll` | string | Role for ALL deployments: `"admin"`, `"editor"`, `"viewer"` |
| `deploymentRoles` | array | Per-deployment role assignments |
| `deploymentRoles[].deploymentId` | string | Deployment ID |
| `deploymentRoles[].role` | string | `"admin"`, `"editor"`, or `"viewer"` |
| `elasticsearchRoleAll` | string | Role for all Elasticsearch serverless projects |
| `elasticsearchRoles` | array | Per-project Elasticsearch roles |
| `observabilityRoleAll` | string | Role for all Observability serverless projects |
| `observabilityRoles` | array | Per-project Observability roles |
| `securityRoleAll` | string | Role for all Security serverless projects |
| `securityRoles` | array | Per-project Security roles |
| `expiresIn` | string | Invitation expiry (e.g., `"72h"`, `"168h"`) |

### Key Outputs

| Output | Type | Description |
|--------|------|-------------|
| `memberID` | string | Member user ID (set after invitation is accepted) |
| `invitationToken` | string | Invitation token (while pending) |
| `accepted` | bool | Whether the invitation has been accepted |

### Role Hierarchy

**Organization-level:**
- `organizationOwner` — Full org admin access
- `billingAdmin` — Manage billing and subscriptions

**Deployment-level (hosted clusters):**
- `admin` — Full deployment management
- `editor` — Modify deployment configuration
- `viewer` — Read-only access

**Serverless project-level:**
- Applies to Elasticsearch, Observability, and Security serverless projects
- Use `*RoleAll` for blanket access or `*Roles` for per-project control
- Role values are project-type-specific (e.g., `"admin"`, `"editor"`, `"viewer"`, `"developer"`)

## TrafficFilter

Manage network security rulesets that control which traffic can reach your Elastic Cloud deployments. Corresponds to the [Network Security](https://cloud.elastic.co/access-security/network-security) page in the Elastic Cloud console.

### IP Allowlist

Whitelist specific IPs or CIDR ranges (e.g., Azure NAT Gateway egress IPs, office networks).

```typescript
const azureNat = new elasticstack.cloud.TrafficFilter("azure-nat", {
    name: "Azure NAT Gateway Egress",
    type: "ip",
    region: "azure-eastus2",
    description: "Allow traffic from our Azure NAT Gateway",
    includeByDefault: false,
    rules: [
        { source: "20.62.134.50", description: "NAT Gateway primary IP" },
        { source: "20.62.134.51", description: "NAT Gateway secondary IP" },
    ],
});

// Attach the filter to a deployment
const natAssoc = new elasticstack.cloud.TrafficFilterAssociation("azure-nat-assoc", {
    rulesetId: azureNat.rulesetId,
    deploymentId: "my-deployment-id",
});
```

### Azure Private Link

```typescript
const privateLinkFilter = new elasticstack.cloud.TrafficFilter("azure-pl", {
    name: "Azure Private Link",
    type: "azure_private_endpoint",
    region: "azure-eastus2",
    rules: [
        {
            azureEndpointName: "my-private-endpoint",
            azureEndpointGuid: "7c0f05e4-e32b-4b10-a246-7b77f7dcc63c",
        },
    ],
});

const plAssoc = new elasticstack.cloud.TrafficFilterAssociation("pl-assoc", {
    rulesetId: privateLinkFilter.rulesetId,
    deploymentId: "my-deployment-id",
});
```

### AWS PrivateLink (VPC Endpoint)

```typescript
const vpcFilter = new elasticstack.cloud.TrafficFilter("aws-vpc", {
    name: "AWS VPC Endpoint",
    type: "vpce",
    region: "us-east-1",
    rules: [
        { source: "vpce-0123456789abcdef0" },
    ],
});
```

### GCP Private Service Connect

```typescript
const gpcFilter = new elasticstack.cloud.TrafficFilter("gcp-psc", {
    name: "GCP PSC Endpoint",
    type: "gcp_private_service_connect_endpoint",
    region: "gcp-us-central1",
    rules: [
        { source: "18446744072646845332" },
    ],
});
```

### Egress Firewall

Control outbound traffic from your deployment.

```typescript
const egressFilter = new elasticstack.cloud.TrafficFilter("egress", {
    name: "Egress to Backend Services",
    type: "egress_firewall",
    region: "azure-eastus2",
    rules: [
        {
            description: "Allow HTTPS to internal API",
            egressRule: { target: "10.0.1.0/24", protocol: "tcp", ports: [443] },
        },
        {
            description: "Allow ES cross-cluster",
            egressRule: { target: "10.0.2.0/24", protocol: "tcp", ports: [9243, 9300] },
        },
    ],
});
```

### Auto-Apply to New Deployments

```typescript
const defaultFilter = new elasticstack.cloud.TrafficFilter("default", {
    name: "Default Allowlist",
    type: "ip",
    region: "azure-eastus2",
    includeByDefault: true,  // Automatically applied to all new deployments
    rules: [
        { source: "10.0.0.0/8", description: "Internal network" },
    ],
});
```

### Key Inputs — TrafficFilter

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Ruleset name |
| `type` | string | `"ip"`, `"vpce"`, `"azure_private_endpoint"`, `"gcp_private_service_connect_endpoint"`, or `"egress_firewall"` |
| `region` | string | Cloud region (e.g., `"azure-eastus2"`, `"us-east-1"`). Immutable. |
| `description` | string | Optional description |
| `includeByDefault` | bool | Auto-attach to new deployments in this region |
| `rules` | array | Traffic filter rules (see below) |

### Rule Fields by Type

| Field | Used By | Description |
|-------|---------|-------------|
| `source` | ip, vpce, gcp_private_service_connect_endpoint | IP/CIDR, VPC endpoint ID, or PSC connection ID |
| `description` | all | Rule description |
| `azureEndpointName` | azure_private_endpoint | Azure Private Link endpoint name |
| `azureEndpointGuid` | azure_private_endpoint | Azure Private Link endpoint GUID |
| `egressRule.target` | egress_firewall | Target IP or CIDR |
| `egressRule.protocol` | egress_firewall | `"all"`, `"tcp"`, or `"udp"` |
| `egressRule.ports` | egress_firewall | Port numbers (optional, empty = all ports) |

## TrafficFilterAssociation

Attach a traffic filter ruleset to a deployment. This is a separate resource so you can manage which deployments use which filters independently.

```typescript
const assoc = new elasticstack.cloud.TrafficFilterAssociation("my-assoc", {
    rulesetId: myFilter.rulesetId,
    deploymentId: "my-deployment-id",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `rulesetId` | string | Traffic filter ruleset ID |
| `deploymentId` | string | Deployment ID to attach to |

Changing either input triggers replacement (delete old association + create new one).
