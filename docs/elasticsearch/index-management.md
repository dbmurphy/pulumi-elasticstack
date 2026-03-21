# Index Management

Manage Elasticsearch indices, aliases, data streams, and data stream lifecycles.

## Resources

- `elasticstack.elasticsearch.Index`
- `elasticstack.elasticsearch.IndexAlias`
- `elasticstack.elasticsearch.DataStream`
- `elasticstack.elasticsearch.DataStreamLifecycle`

## Index

Create and manage an Elasticsearch index with mappings, settings, and aliases.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const logs = new elasticstack.elasticsearch.Index("logs", {
    name: "application-logs-000001",
    numberOfShards: 3,
    numberOfReplicas: 1,
    mappings: JSON.stringify({
        properties: {
            "@timestamp": { type: "date" },
            message: { type: "text" },
            level: { type: "keyword" },
            service: { type: "keyword" },
            trace_id: { type: "keyword" },
        },
    }),
    settings: JSON.stringify({
        "index.refresh_interval": "5s",
        "index.max_result_window": 50000,
    }),
    aliases: JSON.stringify({
        "application-logs": { is_write_index: true },
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Index name (required) |
| `numberOfShards` | int | Number of primary shards |
| `numberOfReplicas` | int | Number of replica shards |
| `mappings` | string (JSON) | Field mappings |
| `settings` | string (JSON) | Index settings |
| `settingsRaw` | string (JSON) | Raw settings (overrides individual settings) |
| `aliases` | string (JSON) | Index aliases |
| `deletionProtection` | bool | Prevent accidental deletion |
| `adoptOnCreate` | bool | Adopt existing index instead of failing |

## IndexAlias

Manage an alias that points to one or more indices.

```typescript
const alias = new elasticstack.elasticsearch.IndexAlias("logs-alias", {
    name: "current-logs",
    indices: ["application-logs-000001", "application-logs-000002"],
    routing: "1",
    isWriteIndex: true,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Alias name |
| `indices` | string[] | Target index names |
| `filter` | string (JSON) | Optional filter query |
| `routing` | string | Default routing value |
| `isWriteIndex` | bool | Mark as write index |

## DataStream

Manage a data stream for time-series data.

```typescript
// First create an index template with data_stream enabled
const template = new elasticstack.elasticsearch.IndexTemplate("metrics-tpl", {
    name: "metrics-template",
    indexPatterns: ["metrics-*"],
    dataStream: JSON.stringify({}),
    template: JSON.stringify({
        settings: { number_of_replicas: 1 },
        mappings: {
            properties: {
                "@timestamp": { type: "date" },
                host: { type: "keyword" },
                cpu: { type: "float" },
            },
        },
    }),
});

// Then create the data stream
const ds = new elasticstack.elasticsearch.DataStream("metrics", {
    name: "metrics-app",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Data stream name |
| `adoptOnCreate` | bool | Adopt existing data stream |
| `deletionProtection` | bool | Prevent accidental deletion |

## DataStreamLifecycle

Configure retention and downsampling for a data stream.

```typescript
const lifecycle = new elasticstack.elasticsearch.DataStreamLifecycle("metrics-lifecycle", {
    name: "metrics-app",
    dataRetention: "30d",
    enabled: true,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Data stream name |
| `dataRetention` | string | Retention period (e.g., "30d") |
| `downsampling` | string (JSON) | Downsampling configuration |
| `enabled` | bool | Enable/disable lifecycle |
