# Getting Started

## Prerequisites

- [Go 1.26.1+](https://golang.org/dl/)
- [Pulumi CLI](https://www.pulumi.com/docs/install/)
- Access to an Elasticsearch cluster (self-hosted or Elastic Cloud)

## Installation

Build the provider from source:

```bash
git clone https://github.com/dbmurphy/pulumi-elasticstack.git
cd pulumi-elasticstack
make build
```

This produces `bin/pulumi-resource-elasticstack`.

## Provider Configuration

The provider supports four connection blocks, each configurable via Pulumi config or environment variables.

### Elasticsearch

| Property | Env Var | Description |
|----------|---------|-------------|
| `endpoints` | `ELASTICSEARCH_ENDPOINTS` | List of ES endpoint URLs |
| `username` | `ELASTICSEARCH_USERNAME` | Basic auth username |
| `password` | `ELASTICSEARCH_PASSWORD` | Basic auth password |
| `apiKey` | `ELASTICSEARCH_API_KEY` | API key (base64 encoded) |
| `bearerToken` | `ELASTICSEARCH_BEARER_TOKEN` | JWT bearer token |
| `insecure` | `ELASTICSEARCH_INSECURE` | Skip TLS verification |
| `caFile` | — | Path to CA certificate file |
| `caData` | — | PEM-encoded CA certificate |
| `certFile` | — | Client certificate file (mTLS) |
| `certData` | — | PEM-encoded client certificate |
| `keyFile` | — | Client private key file |
| `keyData` | — | PEM-encoded client private key |

### Kibana

Inherits ES credentials (username, password, apiKey, bearerToken) if not set.

| Property | Env Var | Description |
|----------|---------|-------------|
| `endpoints` | `KIBANA_ENDPOINT` | List of Kibana endpoint URLs |
| `username` | `KIBANA_USERNAME` | Basic auth username |
| `password` | `KIBANA_PASSWORD` | Basic auth password |
| `apiKey` | `KIBANA_API_KEY` | API key |
| `bearerToken` | `KIBANA_BEARER_TOKEN` | JWT bearer token |
| `caCerts` | `KIBANA_CA_CERTS` | CA certificate file paths |
| `insecure` | `KIBANA_INSECURE` | Skip TLS verification |

### Fleet

Inherits from Kibana, which inherits from ES.

| Property | Env Var | Description |
|----------|---------|-------------|
| `endpoint` | `FLEET_ENDPOINT` | Fleet/Kibana endpoint URL |
| `username` | `FLEET_USERNAME` | Basic auth username |
| `password` | `FLEET_PASSWORD` | Basic auth password |
| `apiKey` | `FLEET_API_KEY` | API key |
| `bearerToken` | `FLEET_BEARER_TOKEN` | JWT bearer token |
| `caCerts` | `FLEET_CA_CERTS` | CA certificate file paths |

### Elastic Cloud

| Property | Env Var | Default | Description |
|----------|---------|---------|-------------|
| `apiKey` | `EC_API_KEY` | — | **Required.** API key from [cloud.elastic.co/account/keys](https://cloud.elastic.co/account/keys) |
| `endpoint` | `EC_ENDPOINT` | `https://api.elastic-cloud.com` | Cloud API endpoint |

## Configuration Examples

### Using Environment Variables

```bash
export ELASTICSEARCH_ENDPOINTS='["https://my-cluster.es.eastus2.azure.elastic-cloud.com:9243"]'
export ELASTICSEARCH_USERNAME="elastic"
export ELASTICSEARCH_PASSWORD="my-password"
export KIBANA_ENDPOINT='["https://my-cluster.kb.eastus2.azure.elastic-cloud.com:9243"]'
export EC_API_KEY="my-cloud-api-key"
```

### Using Pulumi Config

```yaml
# Pulumi.dev.yaml
config:
  elasticstack:elasticsearch:
    endpoints:
      - https://localhost:9200
    username: elastic
    password:
      secure: v1:encrypted-password
  elasticstack:kibana:
    endpoints:
      - https://localhost:5601
  elasticstack:cloud:
    apiKey:
      secure: v1:encrypted-api-key
```

## Verify Connectivity

Use the `getInfo` function to verify your Elasticsearch connection:

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const info = elasticstack.elasticsearch.getInfo({});

export const clusterName = info.then(i => i.clusterName);
export const esVersion = info.then(i => i.versionNumber);
```

## Your First Program

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as elasticstack from "@pulumi/elasticstack";

// Create an index
const logs = new elasticstack.elasticsearch.Index("logs", {
    name: "application-logs",
    numberOfShards: 1,
    numberOfReplicas: 1,
    mappings: JSON.stringify({
        properties: {
            "@timestamp": { type: "date" },
            message: { type: "text" },
            level: { type: "keyword" },
        },
    }),
});

// Create a Kibana space
const devSpace = new elasticstack.kibana.Space("dev", {
    spaceId: "development",
    name: "Development",
    description: "Development workspace",
});

// Create a data view in that space
const logsView = new elasticstack.kibana.DataView("logs-view", {
    title: "application-logs*",
    name: "Application Logs",
    timeFieldName: "@timestamp",
    spaceId: devSpace.spaceId,
});

export const indexName = logs.name;
export const spaceName = devSpace.name;
```

## Retry and Timeout Behavior

The provider automatically handles transient errors and rate limiting across all four API clients (Elasticsearch, Kibana, Fleet, Cloud).

### Context-Deadline-Based Retry

Unlike simple retry-count approaches, this provider retries operations until the **context deadline** expires — not after a fixed number of attempts. This is critical for long-running operations and high-throughput deployments where API rate limits (429) or transient service unavailability (503/504) are common.

| Behavior | Details |
|----------|---------|
| **Default operation timeout** | **10 minutes** — applied automatically if no deadline is set |
| **Transient errors (503, 504)** | Exponential backoff with jitter (1s base, 30s cap) |
| **Rate limiting (429)** | Parses `Retry-After` header; falls back to 10s + 5s per attempt (capped at 120s) |
| **Network errors** | Same exponential backoff as transient errors |
| **Context cancellation** | All retries respect `context.Done()` — sleeps are interruptible |

### How It Works

1. Each CRUD operation receives a `context.Context` from the Pulumi engine
2. If the context has no deadline, the provider adds a **10-minute** default (`DefaultOperationTimeout`)
3. The retry loop continues until the context expires, using appropriate backoff between attempts
4. HTTP requests use `http.NewRequestWithContext` so even in-flight requests are cancelled when the context expires

### Rate Limiting (429)

Any API call — Elasticsearch, Kibana, Fleet, or Cloud — can return `429 Too Many Requests` when the cluster is under heavy load, when many resources are being created/updated simultaneously, or when a long-running process (transforms, enrich policies, ML jobs, snapshot operations) is consuming cluster resources.

When a 429 occurs:

1. Provider reads the `Retry-After` header if present (capped at 120s)
2. If no header, waits `10s + (attempt × 5s)` with 0–5s jitter
3. Keeps retrying until the 10-minute deadline expires — **not** after a fixed number of retries
4. Adds small jitter to prevent thundering herd when multiple resources retry simultaneously

This approach handles real-world scenarios where deploying many resources at once saturates the API, or where a single long-running operation holds cluster resources for minutes at a time.

## Common Patterns

### adoptOnCreate

Most resources support `adoptOnCreate: true`, which adopts an existing resource instead of failing if it already exists. Useful for managing pre-existing infrastructure:

```typescript
const existingIndex = new elasticstack.elasticsearch.Index("existing", {
    name: "my-existing-index",
    adoptOnCreate: true,
});
```

### deletionProtection

Some resources support `deletionProtection: true` to prevent accidental deletion:

```typescript
const prodIndex = new elasticstack.elasticsearch.Index("prod", {
    name: "production-data",
    deletionProtection: true,
});
```

### Explicit Provider

When managing multiple clusters, use explicit provider instances:

```typescript
const prod = new elasticstack.Provider("prod", {
    elasticsearch: {
        endpoints: ["https://prod-cluster:9200"],
        username: "elastic",
        password: prodPassword,
    },
});

const index = new elasticstack.elasticsearch.Index("prod-index", {
    name: "prod-logs",
}, { provider: prod });
```
