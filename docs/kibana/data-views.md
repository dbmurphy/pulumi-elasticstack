# Data Views

Manage Kibana data views (index patterns), default data views, and saved object imports.

## Resources

- `elasticstack.kibana.DataView`
- `elasticstack.kibana.DefaultDataView`
- `elasticstack.kibana.ImportSavedObjects`

## DataView

Create data views that define how Kibana accesses Elasticsearch data.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const logsView = new elasticstack.kibana.DataView("logs", {
    title: "logs-*",
    name: "Application Logs",
    timeFieldName: "@timestamp",
    spaceId: "default",
});

// With field formatting and runtime fields
const metricsView = new elasticstack.kibana.DataView("metrics", {
    title: "metrics-*",
    name: "Infrastructure Metrics",
    timeFieldName: "@timestamp",
    sourceFilters: JSON.stringify(["temp_*", "debug_*"]),
    fieldFormats: JSON.stringify({
        "bytes": { id: "bytes", params: { pattern: "0,0.[000]b" } },
        "response_time": { id: "duration", params: { inputFormat: "milliseconds" } },
    }),
    runtimeFieldMap: JSON.stringify({
        "day_of_week": {
            type: "keyword",
            script: { source: "emit(doc['@timestamp'].value.dayOfWeekEnum.getDisplayName(TextStyle.FULL, Locale.ROOT))" },
        },
    }),
    spaceId: "production",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `title` | string | Index pattern (e.g., "logs-*") |
| `name` | string | Display name |
| `timeFieldName` | string | Default time field |
| `sourceFilters` | string (JSON) | Fields to exclude from _source |
| `fieldFormats` | string (JSON) | Custom field display formats |
| `fieldAttrs` | string (JSON) | Field attributes (labels, counts) |
| `runtimeFieldMap` | string (JSON) | Runtime field definitions |
| `allowNoIndex` | bool | Allow pattern to match no indices |
| `spaceId` | string | Kibana space |

## DefaultDataView

Set which data view is the default for a Kibana space.

```typescript
const defaultView = new elasticstack.kibana.DefaultDataView("default", {
    dataViewId: logsView.id,
    force: true,
    spaceId: "default",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `dataViewId` | string | ID of the data view to set as default |
| `force` | bool | Override the current default |
| `spaceId` | string | Kibana space |

## ImportSavedObjects

Import dashboards, visualizations, and other saved objects from NDJSON.

```typescript
import * as fs from "fs";

const dashboards = new elasticstack.kibana.ImportSavedObjects("dashboards", {
    fileContents: fs.readFileSync("./exports/dashboards.ndjson", "utf-8"),
    overwrite: true,
    spaceId: "production",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `fileContents` | string | NDJSON content of saved objects |
| `overwrite` | bool | Overwrite existing objects |
| `spaceId` | string | Target Kibana space |
