# Standalone Elasticsearch Example

This example shows using the provider with a self-hosted Elasticsearch cluster (no Azure/Cloud dependency).

## Configuration

Set environment variables for your cluster:

```bash
export ELASTICSEARCH_ENDPOINTS='["https://localhost:9200"]'
export ELASTICSEARCH_USERNAME="elastic"
export ELASTICSEARCH_PASSWORD="changeme"
export KIBANA_ENDPOINT='["https://localhost:5601"]'
```

Or configure via `Pulumi.dev.yaml`:

```yaml
config:
  elasticstack:elasticsearch:
    endpoints:
      - https://localhost:9200
    username: elastic
    password:
      secure: v1:encrypted
    insecure: true
  elasticstack:kibana:
    endpoints:
      - https://localhost:5601
```

## Program

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as elasticstack from "@pulumi/elasticstack";

// ─── Verify Connectivity ───────────────────────────────────────────

const info = elasticstack.elasticsearch.getInfo({});
export const clusterName = info.then(i => i.clusterName);
export const esVersion = info.then(i => i.versionNumber);

// ─── ILM Policy ────────────────────────────────────────────────────

const ilm = new elasticstack.elasticsearch.IndexLifecycle("logs-ilm", {
    name: "logs-lifecycle",
    hot: JSON.stringify({
        actions: {
            rollover: { max_age: "7d", max_primary_shard_size: "50gb" },
            set_priority: { priority: 100 },
        },
    }),
    warm: JSON.stringify({
        min_age: "7d",
        actions: {
            shrink: { number_of_shards: 1 },
            forcemerge: { max_num_segments: 1 },
            set_priority: { priority: 50 },
        },
    }),
    delete: JSON.stringify({
        min_age: "30d",
        actions: { delete: {} },
    }),
});

// ─── Index Template ────────────────────────────────────────────────

const template = new elasticstack.elasticsearch.IndexTemplate("logs-tpl", {
    name: "application-logs",
    indexPatterns: ["app-logs-*"],
    template: JSON.stringify({
        settings: {
            number_of_shards: 1,
            number_of_replicas: 1,
            "index.lifecycle.name": "logs-lifecycle",
            "index.lifecycle.rollover_alias": "app-logs",
        },
        mappings: {
            properties: {
                "@timestamp": { type: "date" },
                message: { type: "text" },
                level: { type: "keyword" },
                service: { type: "keyword" },
                host: { type: "keyword" },
                trace_id: { type: "keyword" },
                duration_ms: { type: "float" },
            },
        },
        aliases: {
            "app-logs": { is_write_index: true },
        },
    }),
    priority: 200,
});

// ─── Initial Index ─────────────────────────────────────────────────

const index = new elasticstack.elasticsearch.Index("logs-initial", {
    name: "app-logs-000001",
    aliases: JSON.stringify({
        "app-logs": { is_write_index: true },
    }),
    adoptOnCreate: true,
});

// ─── Ingest Pipeline ───────────────────────────────────────────────

const pipeline = new elasticstack.elasticsearch.Pipeline("logs-pipeline", {
    name: "app-logs-pipeline",
    description: "Enrich and normalize application logs",
    processors: JSON.stringify([
        {
            set: {
                field: "_source.ingested_at",
                value: "{{_ingest.timestamp}}",
            },
        },
        {
            lowercase: {
                field: "level",
                ignore_failure: true,
            },
        },
        {
            grok: {
                field: "message",
                patterns: [
                    "%{IP:client_ip} %{WORD:method} %{URIPATHPARAM:path} %{NUMBER:status:int} %{NUMBER:bytes:long}",
                ],
                ignore_failure: true,
            },
        },
    ]),
});

// ─── Security ──────────────────────────────────────────────────────

const readerRole = new elasticstack.elasticsearch.Role("reader", {
    name: "log_reader",
    cluster: ["monitor"],
    indices: JSON.stringify([
        {
            names: ["app-logs-*"],
            privileges: ["read", "view_index_metadata"],
        },
    ]),
});

const writerRole = new elasticstack.elasticsearch.Role("writer", {
    name: "log_writer",
    cluster: [],
    indices: JSON.stringify([
        {
            names: ["app-logs-*"],
            privileges: ["write", "create_index"],
        },
    ]),
});

const readerUser = new elasticstack.elasticsearch.User("reader-user", {
    username: "log_reader",
    password: pulumi.secret("reader-password"),
    roles: ["log_reader", "kibana_user"],
    fullName: "Log Reader",
    enabled: true,
});

const writerUser = new elasticstack.elasticsearch.User("writer-user", {
    username: "log_writer",
    password: pulumi.secret("writer-password"),
    roles: ["log_writer"],
    fullName: "Log Writer Service",
    enabled: true,
});

const apiKey = new elasticstack.elasticsearch.ApiKey("service-key", {
    name: "log-shipper-key",
    roleDescriptors: JSON.stringify({
        log_writer: {
            index: [
                { names: ["app-logs-*"], privileges: ["write", "create_index"] },
            ],
        },
    }),
    expiration: "90d",
});

// ─── Kibana ────────────────────────────────────────────────────────

const space = new elasticstack.kibana.Space("app-space", {
    spaceId: "application",
    name: "Application",
    description: "Application team workspace",
    color: "#0077cc",
});

const dataView = new elasticstack.kibana.DataView("logs-view", {
    title: "app-logs-*",
    name: "Application Logs",
    timeFieldName: "@timestamp",
    spaceId: space.spaceId,
});

const defaultView = new elasticstack.kibana.DefaultDataView("default", {
    dataViewId: dataView.id,
    force: true,
    spaceId: space.spaceId,
});

// ─── Exports ───────────────────────────────────────────────────────

export const indexName = index.name;
export const pipelineName = pipeline.name;
export const spaceName = space.name;
export const apiKeyEncoded = apiKey.encoded;
```
