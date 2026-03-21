import * as pulumi from "@pulumi/pulumi";
import * as elasticstack from "@pulumi/elasticstack";

// Elasticsearch index with lifecycle management
const myIndex = new elasticstack.elasticsearch.Index("my-index", {
    name: "my-application-logs",
    numberOfShards: 1,
    numberOfReplicas: 1,
    adoptOnCreate: true,
    deletionProtection: true,
});

// Index template for application logs
const logsTemplate = new elasticstack.elasticsearch.IndexTemplate("logs-template", {
    name: "my-logs-template",
    indexPatterns: ["my-logs-*"],
    template: JSON.stringify({
        settings: {
            number_of_shards: 1,
            number_of_replicas: 1,
        },
        mappings: {
            properties: {
                "@timestamp": { type: "date" },
                message: { type: "text" },
                level: { type: "keyword" },
            },
        },
    }),
    priority: 100,
});

// ILM policy for log rotation
const ilmPolicy = new elasticstack.elasticsearch.IndexLifecycle("logs-ilm", {
    name: "logs-lifecycle",
    hotPhase: JSON.stringify({
        actions: {
            rollover: { max_age: "7d", max_primary_shard_size: "50gb" },
        },
    }),
    deletePhase: JSON.stringify({
        min_age: "30d",
        actions: { delete: {} },
    }),
});

// Ingest pipeline
const pipeline = new elasticstack.elasticsearch.IngestPipeline("logs-pipeline", {
    name: "my-logs-pipeline",
    description: "Pipeline for processing application logs",
    processors: JSON.stringify([
        { set: { field: "_source.ingested_at", value: "{{_ingest.timestamp}}" } },
        { lowercase: { field: "level" } },
    ]),
});

// Security role for log readers
const readerRole = new elasticstack.elasticsearch.SecurityRole("log-reader", {
    name: "log_reader",
    indices: JSON.stringify([
        { names: ["my-logs-*"], privileges: ["read", "view_index_metadata"] },
    ]),
});

// Kibana space
const devSpace = new elasticstack.kibana.Space("dev-space", {
    spaceId: "development",
    name: "Development",
    description: "Development team workspace",
    color: "#00bfb3",
});

// Kibana data view
const logsDataView = new elasticstack.kibana.DataView("logs-view", {
    title: "my-logs-*",
    name: "Application Logs",
    timeFieldName: "@timestamp",
    spaceId: "development",
});

// Fleet agent policy
const agentPolicy = new elasticstack.fleet.AgentPolicy("monitoring-policy", {
    name: "monitoring-agents",
    namespace: "default",
    description: "Policy for monitoring agents",
    monitorLogs: true,
    monitorMetrics: true,
});

// Export resource IDs
export const indexName = myIndex.name;
export const templateName = logsTemplate.name;
export const spaceName = devSpace.name;
