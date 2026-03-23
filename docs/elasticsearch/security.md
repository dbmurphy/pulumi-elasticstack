# Security

Manage Elasticsearch users, roles, role mappings, and API keys.

## Resources

- `elasticstack.elasticsearch.User`
- `elasticstack.elasticsearch.SystemUser`
- `elasticstack.elasticsearch.Role`
- `elasticstack.elasticsearch.RoleMapping`
- `elasticstack.elasticsearch.ApiKey`

## User

Create and manage native Elasticsearch users.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const appUser = new elasticstack.elasticsearch.User("app-user", {
    username: "app_service",
    password: "secure-password-here",
    roles: ["app_reader", "kibana_user"],
    fullName: "Application Service Account",
    email: "app@company.com",
    enabled: true,
    metadata: JSON.stringify({
        team: "platform",
        environment: "production",
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `username` | string | Username |
| `password` | string | Password (secret) |
| `passwordHash` | string | Bcrypt password hash (alternative to password) |
| `roles` | string[] | Assigned role names |
| `fullName` | string | Display name |
| `email` | string | Email address |
| `metadata` | string (JSON) | User metadata |
| `enabled` | bool | Whether the user is enabled |

## SystemUser

Change passwords for built-in system users (elastic, kibana_system, etc.).

```typescript
const elasticPassword = new elasticstack.elasticsearch.SystemUser("elastic-pw", {
    username: "elastic",
    password: "new-secure-password",
});

const kibanaSystem = new elasticstack.elasticsearch.SystemUser("kibana-system", {
    username: "kibana_system",
    password: "kibana-system-password",
});
```

## Role

Define custom roles with cluster and index privileges.

```typescript
const readerRole = new elasticstack.elasticsearch.Role("log-reader", {
    name: "log_reader",
    cluster: ["monitor"],
    indices: JSON.stringify([
        {
            names: ["logs-*", "metrics-*"],
            privileges: ["read", "view_index_metadata"],
        },
    ]),
});

const adminRole = new elasticstack.elasticsearch.Role("app-admin", {
    name: "app_admin",
    cluster: ["manage_index_templates", "monitor", "manage_ilm"],
    indices: JSON.stringify([
        {
            names: ["app-*"],
            privileges: ["all"],
        },
        {
            names: ["logs-*"],
            privileges: ["read", "write", "create_index"],
            field_security: {
                grant: ["*"],
                except: ["sensitive_*"],
            },
        },
    ]),
    applications: JSON.stringify([
        {
            application: "kibana-.kibana",
            privileges: ["feature_discover.all", "feature_dashboard.all"],
            resources: ["space:development"],
        },
    ]),
    runAs: ["reporting_user"],
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Role name |
| `cluster` | string[] | Cluster-level privileges |
| `indices` | string (JSON) | Index privilege specifications |
| `applications` | string (JSON) | Application privileges |
| `runAs` | string[] | Users this role can impersonate |
| `metadata` | string (JSON) | Role metadata |

## RoleMapping

Map external identities (LDAP, SAML, etc.) to Elasticsearch roles.

```typescript
const ldapMapping = new elasticstack.elasticsearch.RoleMapping("ldap-admins", {
    name: "ldap_admins",
    enabled: true,
    roles: ["superuser"],
    rules: JSON.stringify({
        all: [
            { field: { "groups": "cn=admins,ou=groups,dc=company,dc=com" } },
            { field: { "realm.name": "ldap1" } },
        ],
    }),
    metadata: JSON.stringify({ source: "ldap" }),
});

// Template-based role assignment
const samlMapping = new elasticstack.elasticsearch.RoleMapping("saml-users", {
    name: "saml_users",
    enabled: true,
    roleTemplates: JSON.stringify([
        {
            template: { source: "{{#tojson}}groups{{/tojson}}" },
            format: "json",
        },
    ]),
    rules: JSON.stringify({
        field: { "realm.name": "saml1" },
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Role mapping name |
| `enabled` | bool | Whether the mapping is active |
| `roles` | string[] | Static role names to assign |
| `roleTemplates` | string (JSON) | Dynamic role templates |
| `rules` | string (JSON) | Matching rules for the mapping |
| `metadata` | string (JSON) | Mapping metadata |

## ApiKey

Create API keys for programmatic access. API keys are immutable — changes trigger replacement.

```typescript
const apiKey = new elasticstack.elasticsearch.ApiKey("service-key", {
    name: "my-service-key",
    roleDescriptors: JSON.stringify({
        log_writer: {
            cluster: [],
            index: [
                {
                    names: ["logs-*"],
                    privileges: ["write", "create_index"],
                },
            ],
        },
    }),
    expiration: "90d",
    metadata: JSON.stringify({
        application: "log-shipper",
        team: "platform",
    }),
});

export const apiKeyEncoded = apiKey.encoded;
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | API key name |
| `roleDescriptors` | string (JSON) | Role-based access restrictions |
| `expiration` | string | Key expiration (e.g., "90d") |
| `metadata` | string (JSON) | Key metadata |
