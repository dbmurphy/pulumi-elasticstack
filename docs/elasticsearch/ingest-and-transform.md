# Ingest & Transform

Manage Elasticsearch ingest pipelines and transforms.

## Resources

- `elasticstack.elasticsearch.Pipeline`
- `elasticstack.elasticsearch.Transform`

## Pipeline

Define pipelines that process documents before indexing.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const logsPipeline = new elasticstack.elasticsearch.Pipeline("logs", {
    name: "logs-pipeline",
    description: "Process application logs before indexing",
    processors: JSON.stringify([
        {
            date: {
                field: "timestamp",
                target_field: "@timestamp",
                formats: ["ISO8601", "yyyy-MM-dd HH:mm:ss"],
            },
        },
        {
            lowercase: { field: "level" },
        },
        {
            grok: {
                field: "message",
                patterns: ["%{IP:client_ip} %{WORD:method} %{URIPATHPARAM:path} %{NUMBER:status:int}"],
                ignore_failure: true,
            },
        },
        {
            set: {
                field: "_source.ingested_at",
                value: "{{_ingest.timestamp}}",
            },
        },
        {
            remove: {
                field: "temp_field",
                ignore_missing: true,
            },
        },
    ]),
    onFailure: JSON.stringify([
        {
            set: {
                field: "error.pipeline",
                value: "logs-pipeline",
            },
        },
        {
            set: {
                field: "error.message",
                value: "{{_ingest.on_failure_message}}",
            },
        },
    ]),
    version: 1,
});
```

### GeoIP Enrichment Pipeline

```typescript
const geoipPipeline = new elasticstack.elasticsearch.Pipeline("geoip", {
    name: "geoip-enrichment",
    description: "Add geographic info from IP addresses",
    processors: JSON.stringify([
        {
            geoip: {
                field: "client_ip",
                target_field: "geo",
            },
        },
        {
            user_agent: {
                field: "user_agent_string",
                target_field: "user_agent",
                ignore_missing: true,
            },
        },
    ]),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Pipeline name |
| `description` | string | Pipeline description |
| `processors` | string (JSON) | Array of processor definitions |
| `onFailure` | string (JSON) | Processors to run on failure |
| `version` | int | Pipeline version |
| `metadata` | string (JSON) | Pipeline metadata |

## Transform

Create transforms that pivot or aggregate data from source indices into destination indices.

```typescript
// Pivot transform: aggregate web logs into hourly stats per host
const webStats = new elasticstack.elasticsearch.Transform("web-stats", {
    name: "web-stats-hourly",
    source: JSON.stringify({
        index: ["weblogs-*"],
        query: {
            bool: {
                filter: [
                    { range: { "@timestamp": { gte: "now-1y" } } },
                ],
            },
        },
    }),
    destination: JSON.stringify({
        index: "web-stats-hourly",
    }),
    pivot: JSON.stringify({
        group_by: {
            host: { terms: { field: "host.keyword" } },
            hour: { date_histogram: { field: "@timestamp", calendar_interval: "1h" } },
        },
        aggregations: {
            total_requests: { value_count: { field: "_id" } },
            avg_response_time: { avg: { field: "response_time" } },
            max_response_time: { max: { field: "response_time" } },
            error_count: {
                filter: { range: { status: { gte: 500 } } },
            },
        },
    }),
    frequency: "1h",
    sync: JSON.stringify({
        time: {
            field: "@timestamp",
            delay: "60s",
        },
    }),
    enabled: true,
    description: "Hourly web traffic statistics per host",
});
```

### Latest Transform

```typescript
const latestStatus = new elasticstack.elasticsearch.Transform("latest-status", {
    name: "latest-device-status",
    source: JSON.stringify({ index: ["device-telemetry-*"] }),
    destination: JSON.stringify({ index: "device-latest-status" }),
    latest: JSON.stringify({
        unique_key: ["device_id"],
        sort: "@timestamp",
    }),
    frequency: "5m",
    sync: JSON.stringify({
        time: { field: "@timestamp", delay: "30s" },
    }),
    enabled: true,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Transform ID |
| `source` | string (JSON) | Source index and optional query |
| `destination` | string (JSON) | Destination index |
| `pivot` | string (JSON) | Pivot aggregation config (mutually exclusive with `latest`) |
| `latest` | string (JSON) | Latest document config (mutually exclusive with `pivot`) |
| `frequency` | string | Check interval (e.g., "1h", "5m") |
| `sync` | string (JSON) | Continuous sync configuration |
| `enabled` | bool | Start the transform immediately |
| `description` | string | Transform description |
