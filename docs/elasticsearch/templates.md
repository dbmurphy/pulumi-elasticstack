# Templates

Manage Elasticsearch index templates, component templates, and ILM policy attachments.

## Resources

- `elasticstack.elasticsearch.IndexTemplate`
- `elasticstack.elasticsearch.ComponentTemplate`
- `elasticstack.elasticsearch.IndexTemplateIlmAttachment`

## IndexTemplate

Define index templates that automatically apply settings, mappings, and aliases to new indices.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

const logsTemplate = new elasticstack.elasticsearch.IndexTemplate("logs", {
    name: "logs-template",
    indexPatterns: ["logs-*"],
    composedOf: ["logs-settings", "logs-mappings"],
    template: JSON.stringify({
        settings: {
            number_of_shards: 2,
            number_of_replicas: 1,
            "index.refresh_interval": "10s",
        },
        mappings: {
            properties: {
                "@timestamp": { type: "date" },
                message: { type: "text" },
                level: { type: "keyword" },
            },
        },
        aliases: {
            "all-logs": {},
        },
    }),
    priority: 200,
    version: 1,
    meta: JSON.stringify({ managed_by: "pulumi" }),
});
```

### With Data Stream

```typescript
const metricsTemplate = new elasticstack.elasticsearch.IndexTemplate("metrics", {
    name: "metrics-template",
    indexPatterns: ["metrics-*"],
    dataStream: JSON.stringify({}),
    template: JSON.stringify({
        settings: { number_of_replicas: 1 },
    }),
    priority: 100,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Template name |
| `indexPatterns` | string[] | Index name patterns to match |
| `composedOf` | string[] | Component template names to compose |
| `dataStream` | string (JSON) | Data stream config (enables data stream mode) |
| `template` | string (JSON) | Settings, mappings, and aliases |
| `priority` | int | Template priority (higher wins) |
| `version` | int | Template version |
| `meta` | string (JSON) | User metadata |

## ComponentTemplate

Reusable building blocks for index templates.

```typescript
const settingsComponent = new elasticstack.elasticsearch.ComponentTemplate("log-settings", {
    name: "logs-settings",
    template: JSON.stringify({
        settings: {
            number_of_shards: 1,
            number_of_replicas: 1,
            "index.refresh_interval": "5s",
        },
    }),
});

const mappingsComponent = new elasticstack.elasticsearch.ComponentTemplate("log-mappings", {
    name: "logs-mappings",
    template: JSON.stringify({
        mappings: {
            properties: {
                "@timestamp": { type: "date" },
                message: { type: "text" },
                host: { type: "keyword" },
            },
        },
    }),
});

// Compose into an index template
const template = new elasticstack.elasticsearch.IndexTemplate("composed", {
    name: "logs-composed",
    indexPatterns: ["logs-*"],
    composedOf: [settingsComponent.name, mappingsComponent.name],
    priority: 100,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Component template name |
| `template` | string (JSON) | Settings, mappings, and/or aliases |
| `version` | int | Template version |
| `meta` | string (JSON) | User metadata |

## IndexTemplateIlmAttachment

Attach an ILM policy to an existing index template.

```typescript
const attachment = new elasticstack.elasticsearch.IndexTemplateIlmAttachment("logs-ilm", {
    indexTemplateName: "logs-template",
    policyName: "logs-lifecycle",
    dataStream: true,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `indexTemplateName` | string | Name of the index template |
| `policyName` | string | Name of the ILM policy to attach |
| `dataStream` | bool | Whether the template targets data streams |
