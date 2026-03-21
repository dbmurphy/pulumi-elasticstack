# SLOs (Service Level Objectives)

Manage Kibana SLOs for tracking service reliability.

## Resources

- `elasticstack.kibana.Slo`

## Slo

Define SLOs with indicators, time windows, and budgeting methods.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

// APM latency SLO
const apiLatency = new elasticstack.kibana.Slo("api-latency", {
    name: "API Latency P99 < 500ms",
    description: "99th percentile API latency stays under 500ms",
    indicator: JSON.stringify({
        type: "sli.apm.transactionDuration",
        params: {
            service: "api-gateway",
            environment: "production",
            transactionType: "request",
            transactionName: "*",
            threshold: 500,
            index: "metrics-apm*",
        },
    }),
    timeWindow: JSON.stringify({
        duration: "30d",
        type: "rolling",
    }),
    budgetingMethod: "occurrences",
    objective: JSON.stringify({
        target: 0.99,
    }),
    tags: ["api", "latency", "production"],
    spaceId: "production",
});

// Custom KQL SLO
const availability = new elasticstack.kibana.Slo("availability", {
    name: "Service Availability > 99.9%",
    description: "Service returns successful responses 99.9% of the time",
    indicator: JSON.stringify({
        type: "sli.kql.custom",
        params: {
            index: "logs-*",
            good: "status_code >= 200 AND status_code < 500",
            total: "*",
            filter: "service.name: \"api-gateway\"",
            timestampField: "@timestamp",
        },
    }),
    timeWindow: JSON.stringify({
        duration: "30d",
        type: "rolling",
    }),
    budgetingMethod: "occurrences",
    objective: JSON.stringify({
        target: 0.999,
    }),
    tags: ["availability", "production"],
    spaceId: "production",
});

// Metric-based SLO with time slicing
const throughput = new elasticstack.kibana.Slo("throughput", {
    name: "API Throughput > 1000 rps",
    description: "API processes at least 1000 requests per second",
    indicator: JSON.stringify({
        type: "sli.metric.custom",
        params: {
            index: "metrics-*",
            good: {
                metrics: [
                    { name: "A", aggregation: "sum", field: "http.request.count" },
                ],
                equation: "A",
            },
            total: {
                metrics: [
                    { name: "A", aggregation: "sum", field: "http.request.count" },
                ],
                equation: "A",
            },
            timestampField: "@timestamp",
        },
    }),
    timeWindow: JSON.stringify({
        duration: "7d",
        type: "rolling",
    }),
    budgetingMethod: "timeslices",
    objective: JSON.stringify({
        target: 0.95,
        timesliceTarget: 0.9,
        timesliceWindow: "5m",
    }),
    settings: JSON.stringify({
        syncDelay: "5m",
        frequency: "1m",
    }),
    tags: ["throughput"],
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | SLO name |
| `description` | string | SLO description |
| `indicator` | string (JSON) | SLI type and parameters |
| `timeWindow` | string (JSON) | Rolling or calendar window |
| `budgetingMethod` | string | `"occurrences"` or `"timeslices"` |
| `objective` | string (JSON) | Target percentage and optional timeslice settings |
| `settings` | string (JSON) | Sync delay and evaluation frequency |
| `tags` | string[] | Tags for filtering |
| `groupBy` | string | Field to group SLO instances by |
| `spaceId` | string | Kibana space |

### Indicator Types

| Type | Description |
|------|-------------|
| `sli.kql.custom` | Custom KQL-based good/total events |
| `sli.metric.custom` | Custom metric aggregation |
| `sli.apm.transactionDuration` | APM latency-based |
| `sli.apm.transactionErrorRate` | APM error rate-based |

### Budgeting Methods

| Method | Description |
|--------|-------------|
| `occurrences` | Ratio of good events to total events |
| `timeslices` | Percentage of time windows meeting the target |
