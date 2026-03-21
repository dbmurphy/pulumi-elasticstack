# Advanced Features

Manage Watcher watches, enrich policies, stored scripts, Logstash pipelines, and cluster settings.

## Resources

- `elasticstack.elasticsearch.Watch`
- `elasticstack.elasticsearch.EnrichPolicy`
- `elasticstack.elasticsearch.Script`
- `elasticstack.elasticsearch.LogstashPipeline`
- `elasticstack.elasticsearch.Settings`

## Watch

Create Watcher watches for alerting on data changes.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const errorWatch = new elasticstack.elasticsearch.Watch("error-alert", {
    watchId: "high-error-rate",
    active: true,
    trigger: JSON.stringify({
        schedule: { interval: "5m" },
    }),
    input: JSON.stringify({
        search: {
            request: {
                indices: ["logs-*"],
                body: {
                    query: {
                        bool: {
                            filter: [
                                { range: { "@timestamp": { gte: "now-5m" } } },
                                { term: { level: "error" } },
                            ],
                        },
                    },
                },
            },
        },
    }),
    condition: JSON.stringify({
        compare: { "ctx.payload.hits.total.value": { gt: 100 } },
    }),
    actions: JSON.stringify({
        send_email: {
            email: {
                to: ["oncall@company.com"],
                subject: "High error rate detected",
                body: {
                    text: "{{ctx.payload.hits.total.value}} errors in the last 5 minutes",
                },
            },
        },
        log_alert: {
            logging: {
                text: "High error rate: {{ctx.payload.hits.total.value}} errors",
            },
        },
    }),
    throttlePeriod: "15m",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `watchId` | string | Unique watch identifier |
| `active` | bool | Whether the watch is active |
| `trigger` | string (JSON) | Schedule trigger |
| `input` | string (JSON) | Data input (search, http, chain) |
| `condition` | string (JSON) | Condition to evaluate |
| `actions` | string (JSON) | Actions to execute |
| `transform` | string (JSON) | Optional data transformation |
| `throttlePeriod` | string | Minimum time between executions |

## EnrichPolicy

Create enrich policies for enriching documents during ingest.

```typescript
const geoEnrich = new elasticstack.elasticsearch.EnrichPolicy("geo-enrich", {
    name: "ip-geo-lookup",
    policyType: "match",
    indices: ["ip-database"],
    matchField: "ip_range",
    enrichFields: ["country", "city", "latitude", "longitude"],
    executeOnCreate: true,
});

// Use with an ingest pipeline
const enrichPipeline = new elasticstack.elasticsearch.Pipeline("enrich-pipeline", {
    name: "ip-enrichment",
    processors: JSON.stringify([
        {
            enrich: {
                policy_name: "ip-geo-lookup",
                field: "client_ip",
                target_field: "geo",
            },
        },
    ]),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Policy name |
| `policyType` | string | Policy type: "match", "range", or "geo_match" |
| `indices` | string[] | Source indices for enrichment data |
| `matchField` | string | Field to match against |
| `enrichFields` | string[] | Fields to add from the source |
| `executeOnCreate` | bool | Execute the policy after creation |
| `executeOnUpdate` | bool | Re-execute on updates |
| `executionTimeout` | string | ES timeout for execution (e.g., "5m") |

> **Long-running execution:** When `executeOnCreate` or `executeOnUpdate` is true, Elasticsearch builds the enrich index synchronously. For large source datasets this can take many minutes. The provider's context-deadline-based retry (default 10 minutes) ensures the operation has time to complete. See [Retry and Timeout Behavior](../getting-started.md#retry-and-timeout-behavior) for details.

## Script

Manage stored scripts for use in queries, aggregations, and pipelines.

```typescript
const scoreScript = new elasticstack.elasticsearch.Script("custom-score", {
    scriptId: "calculate_relevance",
    lang: "painless",
    source: `
        double score = _score;
        if (doc['boost'].size() > 0) {
            score *= doc['boost'].value;
        }
        if (doc['published_date'].size() > 0) {
            long ageInDays = (System.currentTimeMillis() - doc['published_date'].value.toInstant().toEpochMilli()) / 86400000L;
            score *= Math.max(0.5, 1.0 - (ageInDays / 365.0));
        }
        return score;
    `,
    context: "score",
});

// Ingest script
const ingestScript = new elasticstack.elasticsearch.Script("tag-script", {
    scriptId: "add_environment_tag",
    lang: "painless",
    source: `
        if (ctx.hostname != null && ctx.hostname.startsWith('prod-')) {
            ctx.environment = 'production';
        } else {
            ctx.environment = 'development';
        }
    `,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `scriptId` | string | Script identifier |
| `lang` | string | Script language (usually "painless") |
| `source` | string | Script source code |
| `context` | string | Script context (score, ingest, update, etc.) |

## LogstashPipeline

Manage centrally-managed Logstash pipelines stored in Elasticsearch.

```typescript
const pipeline = new elasticstack.elasticsearch.LogstashPipeline("syslog", {
    pipelineId: "syslog-processing",
    pipeline: `
input {
    beats {
        port => 5044
        tags => ["syslog"]
    }
}

filter {
    if "syslog" in [tags] {
        grok {
            match => { "message" => "%{SYSLOGTIMESTAMP:syslog_timestamp} %{SYSLOGHOST:syslog_hostname} %{DATA:syslog_program}(?:\\[%{POSINT:syslog_pid}\\])?: %{GREEDYDATA:syslog_message}" }
        }
        date {
            match => [ "syslog_timestamp", "MMM  d HH:mm:ss", "MMM dd HH:mm:ss" ]
        }
    }
}

output {
    elasticsearch {
        hosts => ["https://localhost:9200"]
        index => "syslog-%{+YYYY.MM.dd}"
    }
}
    `,
    description: "Process syslog messages from Beats",
    pipelineWorkers: 4,
    pipelineBatchSize: 500,
    queueType: "persisted",
    queueMaxBytes: "1gb",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `pipelineId` | string | Pipeline identifier |
| `pipeline` | string | Logstash pipeline configuration |
| `description` | string | Pipeline description |
| `pipelineWorkers` | int | Number of worker threads |
| `pipelineBatchSize` | int | Batch size per worker |
| `queueType` | string | Queue type: "memory" or "persisted" |
| `queueMaxBytes` | string | Max queue size |

## Settings

Manage persistent and transient cluster-level settings.

```typescript
const settings = new elasticstack.elasticsearch.Settings("cluster", {
    persistent: JSON.stringify({
        "cluster.routing.allocation.disk.watermark.low": "85%",
        "cluster.routing.allocation.disk.watermark.high": "90%",
        "cluster.routing.allocation.disk.watermark.flood_stage": "95%",
        "action.auto_create_index": "logs-*,metrics-*,.watches",
        "xpack.security.audit.enabled": true,
    }),
    transient: JSON.stringify({
        "cluster.routing.allocation.enable": "all",
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `persistent` | string (JSON) | Settings that survive cluster restart |
| `transient` | string (JSON) | Settings that reset on cluster restart |
