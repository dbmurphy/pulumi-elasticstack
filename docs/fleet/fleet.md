# Fleet

Manage Fleet agent policies, integrations, outputs, and server hosts.

## Resources

- `elasticstack.fleet.AgentPolicy`
- `elasticstack.fleet.Integration`
- `elasticstack.fleet.IntegrationPolicy`
- `elasticstack.fleet.Output`
- `elasticstack.fleet.ServerHost`

## AgentPolicy

Create agent policies that define how Elastic Agents behave.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const monitoring = new elasticstack.fleet.AgentPolicy("monitoring", {
    name: "monitoring-agents",
    namespace: "production",
    description: "Policy for production monitoring agents",
    monitorLogs: true,
    monitorMetrics: true,
});

const webServers = new elasticstack.fleet.AgentPolicy("web-servers", {
    name: "web-server-agents",
    namespace: "production",
    description: "Policy for web server fleet agents",
    monitorLogs: true,
    monitorMetrics: true,
    isProtected: true,
    agentFeatures: JSON.stringify([
        { name: "fqdn", enabled: true },
    ]),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Policy name |
| `namespace` | string | Data namespace |
| `description` | string | Policy description |
| `monitorLogs` | bool | Collect agent logs |
| `monitorMetrics` | bool | Collect agent metrics |
| `isProtected` | bool | Tamper protection |
| `dataOutputId` | string | Output for data |
| `monitoringOutputId` | string | Output for monitoring data |
| `fleetServerHostId` | string | Fleet Server host |

## Integration

Install Fleet integration packages (e.g., Nginx, MySQL, System).

```typescript
const nginx = new elasticstack.fleet.Integration("nginx", {
    name: "nginx",
    version: "1.20.0",
});

const system = new elasticstack.fleet.Integration("system", {
    name: "system",
    version: "1.54.0",
});

const mysql = new elasticstack.fleet.Integration("mysql", {
    name: "mysql",
    version: "1.15.0",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Integration package name |
| `version` | string | Package version |
| `force` | bool | Force install even if already installed |
| `skipDestroy` | bool | Don't uninstall on resource deletion |

## IntegrationPolicy

Attach an integration to an agent policy with specific configuration.

```typescript
const nginxPolicy = new elasticstack.fleet.IntegrationPolicy("nginx-policy", {
    name: "nginx-logs-and-metrics",
    namespace: "production",
    description: "Collect Nginx access/error logs and stub_status metrics",
    agentPolicyId: webServers.id,
    integrationName: "nginx",
    integrationVersion: "1.20.0",
    input: JSON.stringify([
        {
            type: "logfile",
            enabled: true,
            streams: {
                "nginx.access": {
                    enabled: true,
                    vars: {
                        paths: ["/var/log/nginx/access.log"],
                    },
                },
                "nginx.error": {
                    enabled: true,
                    vars: {
                        paths: ["/var/log/nginx/error.log"],
                    },
                },
            },
        },
        {
            type: "nginx/metrics",
            enabled: true,
            streams: {
                "nginx.stubstatus": {
                    enabled: true,
                    vars: {
                        hosts: ["http://127.0.0.1:80/nginx_status"],
                        period: "10s",
                    },
                },
            },
        },
    ]),
});

const systemPolicy = new elasticstack.fleet.IntegrationPolicy("system-policy", {
    name: "system-metrics",
    namespace: "production",
    agentPolicyId: monitoring.id,
    integrationName: "system",
    integrationVersion: "1.54.0",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Policy name |
| `namespace` | string | Data namespace |
| `description` | string | Policy description |
| `agentPolicyId` | string | Parent agent policy ID |
| `integrationName` | string | Integration package name |
| `integrationVersion` | string | Package version |
| `input` | string (JSON) | Input and stream configuration |
| `vars` | string (JSON) | Package-level variables |

## Output

Configure where agents send their data.

```typescript
// Elasticsearch output
const esOutput = new elasticstack.fleet.Output("es-output", {
    name: "production-elasticsearch",
    outputType: "elasticsearch",
    defaultIntegrations: true,
    defaultMonitoring: true,
    hosts: ["https://es-cluster.company.com:9243"],
    ssl: JSON.stringify({
        certificate_authorities: ["/etc/fleet/ca.crt"],
    }),
});

// Logstash output
const logstashOutput = new elasticstack.fleet.Output("logstash-output", {
    name: "logstash-pipeline",
    outputType: "logstash",
    hosts: ["logstash.company.com:5044"],
    ssl: JSON.stringify({
        certificate: "/etc/fleet/client.crt",
        key: "/etc/fleet/client.key",
        certificate_authorities: ["/etc/fleet/ca.crt"],
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Output name |
| `outputType` | string | `"elasticsearch"` or `"logstash"` |
| `defaultIntegrations` | bool | Default output for integration data |
| `defaultMonitoring` | bool | Default output for monitoring data |
| `hosts` | string[] | Output hosts |
| `configYaml` | string | Additional YAML configuration |
| `ssl` | string (JSON) | TLS configuration |

## ServerHost

Configure Fleet Server host URLs.

```typescript
const fleetHost = new elasticstack.fleet.ServerHost("fleet-server", {
    name: "production-fleet-server",
    hosts: ["https://fleet.company.com:8220"],
    isDefault: true,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Host configuration name |
| `hosts` | string[] | Fleet Server URLs |
| `isDefault` | bool | Set as default Fleet Server |
| `proxyId` | string | Proxy configuration ID |

## Complete Fleet Setup

```typescript
// 1. Fleet Server host
const fleetHost = new elasticstack.fleet.ServerHost("fleet", {
    name: "fleet-server",
    hosts: ["https://fleet.company.com:8220"],
    isDefault: true,
});

// 2. Output configuration
const output = new elasticstack.fleet.Output("output", {
    name: "prod-es",
    outputType: "elasticsearch",
    hosts: ["https://es.company.com:9243"],
    defaultIntegrations: true,
});

// 3. Agent policy
const policy = new elasticstack.fleet.AgentPolicy("agents", {
    name: "web-servers",
    namespace: "production",
    monitorLogs: true,
    monitorMetrics: true,
    fleetServerHostId: fleetHost.id,
    dataOutputId: output.id,
});

// 4. Install integration
const nginx = new elasticstack.fleet.Integration("nginx", {
    name: "nginx",
    version: "1.20.0",
});

// 5. Attach integration to policy
const nginxPolicy = new elasticstack.fleet.IntegrationPolicy("nginx-pol", {
    name: "nginx-collection",
    agentPolicyId: policy.id,
    integrationName: nginx.name,
    integrationVersion: "1.20.0",
});
```
