# Dashboards

Manage Kibana dashboards programmatically.

## Resources

- `elasticstack.kibana.Dashboard`

## Dashboard

Create and manage Kibana dashboards. The dashboard body is the raw JSON representation matching the Kibana saved object format.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const overview = new elasticstack.kibana.Dashboard("overview", {
    body: JSON.stringify({
        attributes: {
            title: "Application Overview",
            description: "High-level application health metrics",
            panelsJSON: JSON.stringify([
                {
                    version: "8.14.0",
                    type: "lens",
                    gridData: { x: 0, y: 0, w: 24, h: 15, i: "panel-1" },
                    panelIndex: "panel-1",
                    embeddableConfig: {
                        attributes: {
                            title: "Request Rate",
                            visualizationType: "lnsXY",
                        },
                    },
                },
                {
                    version: "8.14.0",
                    type: "lens",
                    gridData: { x: 24, y: 0, w: 24, h: 15, i: "panel-2" },
                    panelIndex: "panel-2",
                    embeddableConfig: {
                        attributes: {
                            title: "Error Rate",
                            visualizationType: "lnsMetric",
                        },
                    },
                },
            ]),
            optionsJSON: JSON.stringify({
                useMargins: true,
                syncColors: true,
                syncCursor: true,
                syncTooltips: true,
                hidePanelTitles: false,
            }),
            timeRestore: true,
            timeTo: "now",
            timeFrom: "now-24h",
            refreshInterval: {
                pause: false,
                value: 30000,
            },
            kibanaSavedObjectMeta: {
                searchSourceJSON: JSON.stringify({
                    query: { query: "", language: "kuery" },
                    filter: [],
                }),
            },
        },
    }),
    spaceId: "production",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `body` | string (JSON) | Full dashboard saved object body |
| `spaceId` | string | Kibana space |
| `adoptOnCreate` | bool | Adopt existing dashboard |

### Tips

- Export an existing dashboard from Kibana as NDJSON, then use its structure as a template for the `body` field.
- For managing many dashboards, consider using `ImportSavedObjects` with NDJSON files instead — see [Data Views](data-views.md).
- The dashboard API is experimental and may change between Kibana versions.
