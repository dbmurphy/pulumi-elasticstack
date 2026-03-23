# Synthetics

Manage Kibana synthetics monitors, parameters, and private locations.

## Resources

- `elasticstack.kibana.Monitor`
- `elasticstack.kibana.Parameter`
- `elasticstack.kibana.SyntheticsPrivateLocation`

## Monitor

Create uptime monitors that proactively check service availability.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

// HTTP monitor
const apiCheck = new elasticstack.kibana.Monitor("api-check", {
    name: "API Health Check",
    monitorType: "http",
    schedule: 5,
    locations: ["us_east", "eu_west"],
    enabled: true,
    tags: ["api", "production"],
    config: JSON.stringify({
        url: "https://api.company.com/health",
        method: "GET",
        max_response_bytes: 1048576,
        response: {
            status: [200],
        },
        check: {
            response: {
                body: { positive: ["\"status\":\"ok\""] },
            },
        },
    }),
    alert: JSON.stringify({
        status: { enabled: true },
        tls: { enabled: true },
    }),
    retestOnFailure: true,
});

// TCP monitor
const dbCheck = new elasticstack.kibana.Monitor("db-check", {
    name: "Database Connectivity",
    monitorType: "tcp",
    schedule: 3,
    locations: ["us_east"],
    enabled: true,
    config: JSON.stringify({
        host: "db.company.com:5432",
    }),
    tags: ["database", "infrastructure"],
});

// Browser monitor (synthetic journey)
const loginFlow = new elasticstack.kibana.Monitor("login-flow", {
    name: "Login Flow",
    monitorType: "browser",
    schedule: 10,
    locations: ["us_east"],
    enabled: true,
    config: JSON.stringify({
        inline_script: `
step('Go to login page', async () => {
    await page.goto('https://app.company.com/login');
    await page.waitForSelector('#login-form');
});

step('Enter credentials', async () => {
    await page.fill('#username', params.username);
    await page.fill('#password', params.password);
    await page.click('#submit');
});

step('Verify dashboard', async () => {
    await page.waitForSelector('.dashboard');
});
        `,
    }),
    tags: ["browser", "login", "critical"],
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Monitor name |
| `monitorType` | string | `"http"`, `"tcp"`, `"icmp"`, or `"browser"` |
| `schedule` | int | Check interval in minutes |
| `locations` | string[] | Elastic global locations |
| `privateLocations` | string[] | Private location names |
| `enabled` | bool | Whether the monitor is active |
| `tags` | string[] | Tags for filtering |
| `config` | string (JSON) | Monitor-type-specific configuration |
| `alert` | string (JSON) | Alert settings (status, TLS) |
| `retestOnFailure` | bool | Automatically retest on failure |
| `spaceId` | string | Kibana space |

## Parameter

Create reusable parameters for synthetics monitors (e.g., credentials, URLs).

```typescript
const apiUrl = new elasticstack.kibana.Parameter("api-url", {
    key: "api_base_url",
    value: "https://api.company.com",
    description: "Base URL for API monitors",
    tags: ["api"],
    spaceId: "default",
});

const testUser = new elasticstack.kibana.Parameter("test-user", {
    key: "username",
    value: "synthetic-test-user@company.com",
    description: "Test user for browser monitors",
    tags: ["credentials"],
});

const testPassword = new elasticstack.kibana.Parameter("test-password", {
    key: "password",
    value: "test-password-from-vault",
    description: "Test password for browser monitors",
    tags: ["credentials"],
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `key` | string | Parameter key (referenced as `params.key` in monitors) |
| `value` | string | Parameter value |
| `description` | string | Description |
| `tags` | string[] | Tags for organization |
| `spaceId` | string | Kibana space |

## SyntheticsPrivateLocation

Create private locations for running monitors inside your network.

```typescript
const internalLoc = new elasticstack.kibana.SyntheticsPrivateLocation("internal", {
    name: "Internal Datacenter",
    agentPolicyId: "agent-policy-id",
    tags: ["internal", "datacenter"],
    spaceId: "default",
});

// Use the private location in a monitor
const internalCheck = new elasticstack.kibana.Monitor("internal-api", {
    name: "Internal API Check",
    monitorType: "http",
    schedule: 5,
    privateLocations: [internalLoc.name],
    enabled: true,
    config: JSON.stringify({
        url: "http://internal-api.corp:8080/health",
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Location name |
| `agentPolicyId` | string | Fleet agent policy for the private location |
| `tags` | string[] | Tags |
| `spaceId` | string | Kibana space |
