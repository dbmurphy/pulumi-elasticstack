# Lifecycle Management

Manage ILM policies, snapshot lifecycle policies, and snapshot repositories.

## Resources

- `elasticstack.elasticsearch.IndexLifecycle`
- `elasticstack.elasticsearch.SnapshotLifecycle`
- `elasticstack.elasticsearch.SnapshotRepository`

## IndexLifecycle

Define Index Lifecycle Management (ILM) policies for automatic index rollover, shrink, and deletion.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const logsIlm = new elasticstack.elasticsearch.IndexLifecycle("logs-ilm", {
    name: "logs-lifecycle",
    hot: JSON.stringify({
        min_age: "0ms",
        actions: {
            rollover: {
                max_age: "7d",
                max_primary_shard_size: "50gb",
            },
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
    cold: JSON.stringify({
        min_age: "30d",
        actions: {
            set_priority: { priority: 0 },
        },
    }),
    delete: JSON.stringify({
        min_age: "90d",
        actions: { delete: {} },
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | ILM policy name |
| `hot` | string (JSON) | Hot phase configuration |
| `warm` | string (JSON) | Warm phase configuration |
| `cold` | string (JSON) | Cold phase configuration |
| `frozen` | string (JSON) | Frozen phase configuration |
| `delete` | string (JSON) | Delete phase configuration |
| `metadata` | string (JSON) | Policy metadata |

## SnapshotLifecycle

Automate snapshot creation and retention.

```typescript
const slm = new elasticstack.elasticsearch.SnapshotLifecycle("nightly", {
    name: "nightly-snapshots",
    schedule: "0 30 1 * * ?",
    snapshotName: "<nightly-{now/d}>",
    repository: "my-s3-repo",
    indices: ["logs-*", "metrics-*"],
    expireAfter: "30d",
    maxCount: 30,
    minCount: 5,
    ignoreUnavailable: true,
    includeGlobalState: false,
    partial: true,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | SLM policy name |
| `schedule` | string | Cron schedule |
| `snapshotName` | string | Snapshot name pattern (supports date math) |
| `repository` | string | Snapshot repository name |
| `indices` | string[] | Index patterns to include |
| `expireAfter` | string | Auto-delete snapshots older than this |
| `maxCount` | int | Maximum snapshots to retain |
| `minCount` | int | Minimum snapshots to retain |

## SnapshotRepository

Configure where snapshots are stored.

```typescript
// S3 repository
const s3Repo = new elasticstack.elasticsearch.SnapshotRepository("s3-repo", {
    name: "my-s3-repo",
    type: "s3",
    settings: JSON.stringify({
        bucket: "my-es-snapshots",
        region: "us-east-1",
        base_path: "snapshots",
        compress: true,
    }),
    verify: true,
});

// Azure repository
const azureRepo = new elasticstack.elasticsearch.SnapshotRepository("azure-repo", {
    name: "my-azure-repo",
    type: "azure",
    settings: JSON.stringify({
        container: "es-snapshots",
        base_path: "snapshots",
        compress: true,
    }),
});

// Shared filesystem repository
const fsRepo = new elasticstack.elasticsearch.SnapshotRepository("fs-repo", {
    name: "my-fs-repo",
    type: "fs",
    settings: JSON.stringify({
        location: "/mnt/snapshots",
        compress: true,
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Repository name |
| `type` | string | Repository type (s3, azure, gcs, fs, etc.) |
| `settings` | string (JSON) | Repository-specific settings |
| `verify` | bool | Verify repository on create |
| `deletionProtection` | bool | Prevent accidental deletion |
