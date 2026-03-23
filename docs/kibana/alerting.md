# Alerting

Manage Kibana action connectors, alerting rules, and maintenance windows.

## Resources

- `elasticstack.kibana.ActionConnector`
- `elasticstack.kibana.Rule`
- `elasticstack.kibana.MaintenanceWindow`

## ActionConnector

Create connectors that alerting rules use to send notifications.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

// Slack connector
const slack = new elasticstack.kibana.ActionConnector("slack", {
    name: "oncall-slack",
    connectorTypeId: ".slack_api",
    config: JSON.stringify({
        allowedChannels: [{ id: "C01234567", name: "alerts" }],
    }),
    secrets: JSON.stringify({
        token: "xoxb-slack-bot-token",
    }),
    spaceId: "default",
});

// Email connector
const email = new elasticstack.kibana.ActionConnector("email", {
    name: "alert-email",
    connectorTypeId: ".email",
    config: JSON.stringify({
        from: "alerts@company.com",
        host: "smtp.company.com",
        port: 587,
        secure: true,
    }),
    secrets: JSON.stringify({
        user: "alerts@company.com",
        password: "smtp-password",
    }),
});

// PagerDuty connector
const pagerduty = new elasticstack.kibana.ActionConnector("pagerduty", {
    name: "oncall-pagerduty",
    connectorTypeId: ".pagerduty",
    config: JSON.stringify({
        apiUrl: "https://events.pagerduty.com/v2/enqueue",
    }),
    secrets: JSON.stringify({
        routingKey: "your-routing-key",
    }),
});

// Webhook connector
const webhook = new elasticstack.kibana.ActionConnector("webhook", {
    name: "custom-webhook",
    connectorTypeId: ".webhook",
    config: JSON.stringify({
        method: "post",
        url: "https://api.company.com/alerts",
        headers: { "Content-Type": "application/json" },
    }),
    secrets: JSON.stringify({
        user: "webhook-user",
        password: "webhook-password",
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Connector display name |
| `connectorTypeId` | string | Type: `.slack_api`, `.email`, `.pagerduty`, `.webhook`, `.jira`, etc. |
| `config` | string (JSON) | Connector configuration |
| `secrets` | string (JSON) | Sensitive credentials |
| `spaceId` | string | Kibana space (default: "default") |

## Rule

Create rules that evaluate conditions and trigger actions.

```typescript
// Index threshold rule
const highErrorRate = new elasticstack.kibana.Rule("error-rate", {
    name: "High Error Rate",
    consumer: "alerts",
    ruleTypeId: ".index-threshold",
    schedule: JSON.stringify({ interval: "5m" }),
    params: JSON.stringify({
        index: ["logs-*"],
        timeField: "@timestamp",
        aggType: "count",
        groupBy: "top",
        termField: "service.name",
        termSize: 10,
        timeWindowSize: 5,
        timeWindowUnit: "m",
        thresholdComparator: ">",
        threshold: [100],
    }),
    actions: JSON.stringify([
        {
            group: "threshold met",
            id: "slack-connector-id",
            params: {
                channels: ["alerts"],
                text: "High error rate: {{context.group}} has {{context.value}} errors",
            },
        },
    ]),
    enabled: true,
    tags: ["production", "errors"],
    notifyWhen: "onThrottleInterval",
    throttle: "15m",
    spaceId: "default",
});

// Elasticsearch query rule
const diskSpace = new elasticstack.kibana.Rule("disk-space", {
    name: "Low Disk Space",
    consumer: "alerts",
    ruleTypeId: ".es-query",
    schedule: JSON.stringify({ interval: "10m" }),
    params: JSON.stringify({
        searchType: "esQuery",
        index: ["metricbeat-*"],
        timeField: "@timestamp",
        esQuery: JSON.stringify({
            query: {
                bool: {
                    filter: [
                        { range: { "system.filesystem.used.pct": { gte: 0.9 } } },
                    ],
                },
            },
        }),
        timeWindowSize: 10,
        timeWindowUnit: "m",
        threshold: [0],
        thresholdComparator: ">",
    }),
    actions: JSON.stringify([
        {
            group: "query matched",
            id: "pagerduty-connector-id",
            params: {
                severity: "critical",
                summary: "Low disk space detected on {{context.hits}} hosts",
            },
        },
    ]),
    enabled: true,
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Rule name |
| `consumer` | string | Feature consuming the rule (e.g., "alerts", "siem") |
| `ruleTypeId` | string | Rule type: `.index-threshold`, `.es-query`, `siem.queryRule`, etc. |
| `schedule` | string (JSON) | Evaluation interval |
| `params` | string (JSON) | Rule-type-specific parameters |
| `actions` | string (JSON) | Actions to trigger |
| `enabled` | bool | Whether the rule is active |
| `tags` | string[] | Tags for organization |
| `notifyWhen` | string | Notification frequency |
| `throttle` | string | Minimum time between notifications |
| `spaceId` | string | Kibana space |

## MaintenanceWindow

Suppress alerting notifications during planned maintenance.

```typescript
const weeklyMaintenance = new elasticstack.kibana.MaintenanceWindow("weekly", {
    title: "Weekly Deployment Window",
    enabled: true,
    schedule: JSON.stringify({
        dtstart: "2024-01-07T02:00:00Z",
        duration: "2h",
        rrule: {
            freq: "WEEKLY",
            byday: ["SU"],
            count: 52,
        },
    }),
    spaceId: "default",
});

// One-time maintenance window
const migration = new elasticstack.kibana.MaintenanceWindow("migration", {
    title: "Database Migration - Q1",
    enabled: true,
    schedule: JSON.stringify({
        dtstart: "2024-03-15T22:00:00Z",
        duration: "4h",
    }),
    scopedQuery: JSON.stringify({
        kql: "tags: database OR tags: migration",
    }),
    spaceId: "default",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `title` | string | Maintenance window name |
| `enabled` | bool | Whether the window is active |
| `schedule` | string (JSON) | Schedule with dtstart, duration, and optional rrule |
| `scopedQuery` | string (JSON) | KQL filter to scope which alerts are suppressed |
| `spaceId` | string | Kibana space |
