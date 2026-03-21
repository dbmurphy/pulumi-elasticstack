# Machine Learning

Manage Elasticsearch ML anomaly detection jobs, datafeeds, and their running states.

## Resources

- `elasticstack.elasticsearch.AnomalyDetectionJob`
- `elasticstack.elasticsearch.Datafeed`
- `elasticstack.elasticsearch.DatafeedState`
- `elasticstack.elasticsearch.MlJobState`

## AnomalyDetectionJob

Create anomaly detection jobs to find unusual patterns in time-series data.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const latencyJob = new elasticstack.elasticsearch.AnomalyDetectionJob("latency", {
    jobId: "api-latency-anomalies",
    description: "Detect unusual API response time patterns",
    analysisConfig: JSON.stringify({
        bucket_span: "15m",
        detectors: [
            {
                function: "high_mean",
                field_name: "response_time",
                partition_field_name: "endpoint",
                detector_description: "High mean response time per endpoint",
            },
            {
                function: "high_count",
                partition_field_name: "status_code",
                detector_description: "Unusual request count per status code",
            },
        ],
        influencers: ["endpoint", "status_code", "host"],
    }),
    dataDescription: JSON.stringify({
        time_field: "@timestamp",
        time_format: "epoch_ms",
    }),
    analysisLimits: JSON.stringify({
        model_memory_limit: "256mb",
    }),
    modelSnapshotRetentionDays: 7,
    dailyModelSnapshotRetentionAfterDays: 1,
    resultsIndexName: "custom-anomalies",
    groups: ["api", "performance"],
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `jobId` | string | Unique job identifier |
| `analysisConfig` | string (JSON) | Detectors, bucket span, influencers |
| `dataDescription` | string (JSON) | Time field and format |
| `analysisLimits` | string (JSON) | Memory limits |
| `modelSnapshotRetentionDays` | int | Days to retain model snapshots |
| `resultsIndexName` | string | Custom results index name |
| `groups` | string[] | Job groups for organization |
| `description` | string | Job description |

## Datafeed

Connect a data source to an anomaly detection job.

```typescript
const latencyFeed = new elasticstack.elasticsearch.Datafeed("latency-feed", {
    datafeedId: "datafeed-api-latency",
    jobId: latencyJob.jobId,
    indices: ["weblogs-*"],
    query: JSON.stringify({
        bool: {
            filter: [
                { term: { "type": "api_request" } },
            ],
        },
    }),
    frequency: "5m",
    queryDelay: "60s",
    scrollSize: 1000,
});
```

### With Runtime Fields

```typescript
const enrichedFeed = new elasticstack.elasticsearch.Datafeed("enriched-feed", {
    datafeedId: "datafeed-enriched",
    jobId: "my-job-id",
    indices: ["events-*"],
    runtimeMappings: JSON.stringify({
        hour_of_day: {
            type: "long",
            script: "emit(doc['@timestamp'].value.getHour())",
        },
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `datafeedId` | string | Unique datafeed identifier |
| `jobId` | string | Associated ML job ID |
| `indices` | string[] | Source index patterns |
| `query` | string (JSON) | Filter query |
| `frequency` | string | Polling interval |
| `queryDelay` | string | Delay before querying |
| `scrollSize` | int | Documents per scroll page |
| `runtimeMappings` | string (JSON) | Runtime field definitions |

## DatafeedState

Control whether a datafeed is started or stopped.

```typescript
const feedState = new elasticstack.elasticsearch.DatafeedState("feed-running", {
    datafeedId: "datafeed-api-latency",
    started: true,
});
```

## MlJobState

Control whether an ML job is opened or closed.

```typescript
const jobState = new elasticstack.elasticsearch.MlJobState("job-open", {
    jobId: "api-latency-anomalies",
    opened: true,
});
```

## Complete ML Pipeline

```typescript
// 1. Create the job
const job = new elasticstack.elasticsearch.AnomalyDetectionJob("fraud", {
    jobId: "fraud-detection",
    analysisConfig: JSON.stringify({
        bucket_span: "5m",
        detectors: [
            { function: "high_sum", field_name: "amount", by_field_name: "user_id" },
        ],
        influencers: ["user_id", "merchant_category"],
    }),
    dataDescription: JSON.stringify({ time_field: "@timestamp" }),
});

// 2. Create the datafeed
const feed = new elasticstack.elasticsearch.Datafeed("fraud-feed", {
    datafeedId: "datafeed-fraud",
    jobId: job.jobId,
    indices: ["transactions-*"],
});

// 3. Open the job
const jobOpen = new elasticstack.elasticsearch.MlJobState("fraud-open", {
    jobId: job.jobId,
    opened: true,
});

// 4. Start the datafeed
const feedStart = new elasticstack.elasticsearch.DatafeedState("fraud-feed-start", {
    datafeedId: feed.datafeedId,
    started: true,
});
```
