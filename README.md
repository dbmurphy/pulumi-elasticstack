# pulumi-elasticstack

A native Pulumi provider for managing Elastic Stack resources — Elasticsearch, Kibana, Fleet, APM, and Elastic Cloud.

## Overview

This provider lets you manage your entire Elastic Stack as infrastructure-as-code using Pulumi. It covers:

| Module | Resources | Description |
|--------|-----------|-------------|
| **Elasticsearch** | 26 resources, 1 function | Indices, templates, ILM, security, ML, ingest, transforms, watchers |
| **Kibana** | 21 resources | Spaces, alerting, data views, SLOs, detection rules, synthetics, dashboards |
| **Fleet** | 5 resources | Agent policies, integrations, outputs, server hosts |
| **APM** | 1 resource | Agent configuration |
| **Cloud** | 4 resources | Deployment password reset, organization members, network security (traffic filters) |

**57 resources + 1 function** across 5 modules.

## Quick Start

### Prerequisites

- [Go 1.26.1+](https://golang.org/dl/)
- [Pulumi CLI](https://www.pulumi.com/docs/install/)
- Access to an Elasticsearch cluster (self-hosted or Elastic Cloud)

### Build

```bash
make build
```

### Configure

Set environment variables for your Elasticsearch cluster:

```bash
export ELASTICSEARCH_ENDPOINTS='["https://localhost:9200"]'
export ELASTICSEARCH_USERNAME="elastic"
export ELASTICSEARCH_PASSWORD="changeme"
```

For Kibana and Fleet, set their respective endpoints (or they'll inherit ES credentials):

```bash
export KIBANA_ENDPOINT='["https://localhost:5601"]'
export FLEET_ENDPOINT="https://localhost:5601"
```

For Elastic Cloud operations (password reset, org member invitations):

```bash
export EC_API_KEY="your-elastic-cloud-api-key"  # from https://cloud.elastic.co/account/keys
```

### First Program

```typescript
import * as elasticstack from "@pulumi/elasticstack";

// Verify connectivity
const info = elasticstack.elasticsearch.getInfo({});

// Create an index
const logs = new elasticstack.elasticsearch.Index("logs", {
    name: "application-logs",
    numberOfShards: 1,
    numberOfReplicas: 1,
});

export const clusterName = info.then(i => i.clusterName);
export const indexName = logs.name;
```

## Documentation

Detailed documentation with examples for every resource is in the [`docs/`](docs/) folder:

### Getting Started
- **[Getting Started Guide](docs/getting-started.md)** — Installation, configuration, and your first program
- **[Resource Reference](docs/resource-reference.md)** — Quick-reference table of all 57 resources

### Elasticsearch
- **[Index Management](docs/elasticsearch/index-management.md)** — Index, IndexAlias, DataStream, DataStreamLifecycle
- **[Templates](docs/elasticsearch/templates.md)** — IndexTemplate, ComponentTemplate, ILM attachment
- **[Lifecycle Management](docs/elasticsearch/lifecycle.md)** — ILM policies, snapshot lifecycle, snapshot repositories
- **[Security](docs/elasticsearch/security.md)** — Users, roles, role mappings, API keys
- **[Ingest & Transform](docs/elasticsearch/ingest-and-transform.md)** — Ingest pipelines, transforms
- **[Machine Learning](docs/elasticsearch/ml.md)** — Anomaly detection jobs, datafeeds
- **[Advanced](docs/elasticsearch/advanced.md)** — Watchers, enrich policies, scripts, Logstash pipelines, cluster settings

### Kibana
- **[Spaces & Security](docs/kibana/spaces-and-security.md)** — Spaces, Kibana security roles
- **[Alerting](docs/kibana/alerting.md)** — Connectors, alerting rules, maintenance windows
- **[Data Views](docs/kibana/data-views.md)** — Data views, default data view, saved object import
- **[SLOs](docs/kibana/slo.md)** — Service Level Objectives
- **[Detection Rules](docs/kibana/detection.md)** — Security detection rules, exceptions, value lists
- **[Synthetics](docs/kibana/synthetics.md)** — Monitors, parameters, private locations
- **[Dashboards](docs/kibana/dashboards.md)** — Dashboard management

### Fleet & APM
- **[Fleet](docs/fleet/fleet.md)** — Agent policies, integrations, outputs, server hosts
- **[APM](docs/apm/apm.md)** — Agent configuration

### Elastic Cloud
- **[Cloud](docs/cloud/cloud.md)** — Deployment password reset, organization members, network security (traffic filters)

### Examples
- **[End-to-End with Azure](docs/examples/end-to-end-azure.md)** — Deploy via Azure, reset password, configure full stack
- **[Standalone Elasticsearch](docs/examples/standalone-es.md)** — Self-hosted ES with security, ILM, and Kibana

## Development

```bash
make build          # Build the provider binary
make test           # Run tests with race detector + coverage
make test-short     # Run tests without race detector (faster)
make lint           # Run golangci-lint (matches CI)
make vet            # Run go vet
make fmt            # Auto-fix formatting (gofmt + gci)
make fmt-check      # Check formatting without modifying files
make ci             # Run all CI checks locally (lint + vet + test + build + tidy)
make ci-full        # Run all CI checks including vulncheck
make schema         # Generate the Pulumi schema
make gen-sdk        # Generate SDKs for all languages
make hooks          # Install git pre-commit hook
make unhooks        # Remove git pre-commit hook
```

### Project Structure

```
provider/
  cmd/pulumi-resource-elasticstack/  # Entry point
  pkg/
    provider/       # Provider config, registration
    clients/        # HTTP clients (ES, Kibana, Fleet, Cloud)
    elasticsearch/  # ES resources (index, template, security, ml, ...)
    kibana/         # Kibana resources (space, alerting, detection, ...)
    fleet/          # Fleet resources
    apm/            # APM resources
    cloud/          # Cloud resources (password, org member)
docs/               # Per-module documentation with examples
examples/           # Runnable Pulumi programs
sdk/                # Generated SDKs (nodejs, python, go, dotnet)
```

## Supported Languages

SDKs are generated for:

- TypeScript / JavaScript
- Python
- Go
- .NET (C#)

## License

Apache 2.0 — see [LICENSE](LICENSE).
