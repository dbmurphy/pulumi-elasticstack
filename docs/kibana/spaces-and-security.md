# Spaces & Security

Manage Kibana spaces and security roles.

## Resources

- `elasticstack.kibana.Space`
- `elasticstack.kibana.SecurityRole`

## Space

Create isolated workspaces in Kibana with custom branding and feature visibility.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const devSpace = new elasticstack.kibana.Space("dev", {
    spaceId: "development",
    name: "Development",
    description: "Development team workspace",
    color: "#00bfb3",
    initials: "DV",
    disabledFeatures: ["ml", "apm", "siem"],
});

const prodSpace = new elasticstack.kibana.Space("prod", {
    spaceId: "production",
    name: "Production",
    description: "Production monitoring and dashboards",
    color: "#ee4056",
    initials: "PR",
    disabledFeatures: [],
});

const securitySpace = new elasticstack.kibana.Space("security", {
    spaceId: "security-ops",
    name: "Security Operations",
    description: "SIEM and threat detection",
    color: "#6092c0",
    disabledFeatures: ["ml", "dev_tools"],
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `spaceId` | string | Unique space identifier (URL-safe) |
| `name` | string | Display name |
| `description` | string | Space description |
| `color` | string | Hex color code for the space avatar |
| `initials` | string | 1-2 character initials for the avatar |
| `disabledFeatures` | string[] | Kibana features to hide in this space |
| `imageUrl` | string | Custom avatar image (data URI) |

## SecurityRole

Define Kibana roles with space-level and feature-level access control.

```typescript
const devRole = new elasticstack.kibana.SecurityRole("dev-role", {
    name: "development_user",
    elasticsearch: JSON.stringify({
        cluster: ["monitor"],
        indices: [
            {
                names: ["dev-*", "logs-dev-*"],
                privileges: ["read", "view_index_metadata"],
            },
        ],
    }),
    kibana: JSON.stringify([
        {
            spaces: ["development"],
            feature: {
                discover: ["all"],
                dashboard: ["all"],
                visualize: ["all"],
                dev_tools: ["all"],
            },
        },
        {
            spaces: ["production"],
            feature: {
                discover: ["read"],
                dashboard: ["read"],
            },
        },
    ]),
});

// Read-only analyst role
const analystRole = new elasticstack.kibana.SecurityRole("analyst", {
    name: "analyst",
    elasticsearch: JSON.stringify({
        cluster: [],
        indices: [
            {
                names: ["logs-*", "metrics-*"],
                privileges: ["read"],
            },
        ],
    }),
    kibana: JSON.stringify([
        {
            spaces: ["*"],
            feature: {
                discover: ["read"],
                dashboard: ["read"],
                visualize: ["read"],
            },
        },
    ]),
});

// Security analyst with SIEM access
const siemRole = new elasticstack.kibana.SecurityRole("siem-analyst", {
    name: "siem_analyst",
    elasticsearch: JSON.stringify({
        cluster: ["monitor"],
        indices: [
            {
                names: [".siem-signals-*", ".alerts-security*", "logs-*"],
                privileges: ["read", "write", "view_index_metadata"],
            },
        ],
    }),
    kibana: JSON.stringify([
        {
            spaces: ["security-ops"],
            feature: {
                siem: ["all"],
                securitySolutionCases: ["all"],
                actions: ["all"],
            },
        },
    ]),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Role name |
| `elasticsearch` | string (JSON) | ES cluster and index privileges |
| `kibana` | string (JSON) | Per-space feature privileges |
| `metadata` | string (JSON) | Role metadata |
