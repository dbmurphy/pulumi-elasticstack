# End-to-End Example: Azure + Elastic Cloud

This example shows a complete real-world scenario:

1. Deploy an Elasticsearch cluster through Azure marketplace
2. Reset the cluster password via the Cloud module
3. Configure the elasticstack provider with the new credentials
4. Set up a full observability stack (indices, ILM, security, Kibana, Fleet)
5. Invite team members with appropriate access levels

## Prerequisites

```bash
export EC_API_KEY="your-elastic-cloud-api-key"  # from https://cloud.elastic.co/account/keys
```

## Step 1: Deploy Cluster via Azure

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as azure from "@pulumi/azure-native";
import * as elasticstack from "@pulumi/elasticstack";

// Deploy Elastic Cloud through Azure marketplace
const cluster = new azure.elasticcloud.Monitor("elastic-cluster", {
    resourceGroupName: "my-rg",
    location: "eastus2",
    monitorProperties: {
        userInfo: {
            emailAddress: "admin@company.com",
            firstName: "Admin",
            lastName: "User",
        },
        planDetails: {
            planId: "ess-consumption-2024_Monthly",
            offerID: "ec-azure-pp",
            publisherID: "elastic",
        },
    },
    sku: {
        name: "ess-consumption-2024_Monthly",
    },
});
```

## Step 2: Reset Password via Cloud Module

Azure marketplace deployments don't return the `elastic` user password. Use `DeploymentPassword` to generate one.

```typescript
const password = new elasticstack.cloud.DeploymentPassword("cluster-pw", {
    deploymentId: cluster.monitorProperties.apply(p => p!.liftrResourceCategory!),
    refId: "main-elasticsearch",
});
```

## Step 3: Configure the Provider

```typescript
const esProvider = new elasticstack.Provider("es", {
    elasticsearch: {
        endpoints: [cluster.monitorProperties.apply(p =>
            `https://${p!.elasticProperties!.elasticsearchEndpoint}`)],
        username: "elastic",
        password: password.password,
    },
    kibana: {
        endpoints: [cluster.monitorProperties.apply(p =>
            `https://${p!.elasticProperties!.kibanaEndpoint}`)],
    },
});

// All subsequent resources use { provider: esProvider }
```

## Step 4: Elasticsearch Resources

```typescript
// ILM policy for log rotation
const logsIlm = new elasticstack.elasticsearch.IndexLifecycle("logs-ilm", {
    name: "logs-lifecycle",
    hot: JSON.stringify({
        actions: {
            rollover: { max_age: "7d", max_primary_shard_size: "50gb" },
        },
    }),
    warm: JSON.stringify({
        min_age: "7d",
        actions: { forcemerge: { max_num_segments: 1 } },
    }),
    delete: JSON.stringify({
        min_age: "90d",
        actions: { delete: {} },
    }),
}, { provider: esProvider });

// Component templates
const logsSettings = new elasticstack.elasticsearch.ComponentTemplate("logs-settings", {
    name: "logs-settings",
    template: JSON.stringify({
        settings: {
            number_of_shards: 2,
            number_of_replicas: 1,
            "index.lifecycle.name": "logs-lifecycle",
        },
    }),
}, { provider: esProvider });

const logsMappings = new elasticstack.elasticsearch.ComponentTemplate("logs-mappings", {
    name: "logs-mappings",
    template: JSON.stringify({
        mappings: {
            properties: {
                "@timestamp": { type: "date" },
                message: { type: "text" },
                level: { type: "keyword" },
                service: { type: "keyword" },
                trace_id: { type: "keyword" },
                host: { type: "keyword" },
            },
        },
    }),
}, { provider: esProvider });

// Index template
const logsTemplate = new elasticstack.elasticsearch.IndexTemplate("logs-tpl", {
    name: "app-logs",
    indexPatterns: ["app-logs-*"],
    composedOf: ["logs-settings", "logs-mappings"],
    priority: 200,
}, { provider: esProvider });

// Ingest pipeline
const logsPipeline = new elasticstack.elasticsearch.Pipeline("logs-pipeline", {
    name: "app-logs-pipeline",
    description: "Enrich application logs",
    processors: JSON.stringify([
        { set: { field: "_source.ingested_at", value: "{{_ingest.timestamp}}" } },
        { lowercase: { field: "level", ignore_failure: true } },
        { geoip: { field: "client_ip", target_field: "geo", ignore_missing: true } },
    ]),
}, { provider: esProvider });

// Security role for app team
const appRole = new elasticstack.elasticsearch.Role("app-role", {
    name: "app_developer",
    cluster: ["monitor"],
    indices: JSON.stringify([
        { names: ["app-logs-*"], privileges: ["read", "view_index_metadata"] },
        { names: ["app-metrics-*"], privileges: ["read"] },
    ]),
}, { provider: esProvider });

// Service account user
const appUser = new elasticstack.elasticsearch.User("app-user", {
    username: "app_service",
    password: pulumi.secret("app-service-password"),
    roles: ["app_developer"],
    fullName: "Application Service Account",
}, { provider: esProvider });
```

## Step 5: Kibana Resources

```typescript
// Kibana space for the team
const appSpace = new elasticstack.kibana.Space("app-space", {
    spaceId: "application",
    name: "Application Team",
    description: "Application monitoring and dashboards",
    color: "#00bfb3",
}, { provider: esProvider });

// Data view for logs
const logsView = new elasticstack.kibana.DataView("logs-view", {
    title: "app-logs-*",
    name: "Application Logs",
    timeFieldName: "@timestamp",
    spaceId: "application",
}, { provider: esProvider });

// Slack connector for alerts
const slackConnector = new elasticstack.kibana.ActionConnector("slack", {
    name: "oncall-slack",
    connectorTypeId: ".slack_api",
    config: JSON.stringify({
        allowedChannels: [{ id: "C01234567", name: "alerts" }],
    }),
    secrets: JSON.stringify({ token: "xoxb-slack-token" }),
    spaceId: "application",
}, { provider: esProvider });

// High error rate alert
const errorAlert = new elasticstack.kibana.Rule("error-alert", {
    name: "High Error Rate",
    consumer: "alerts",
    ruleTypeId: ".index-threshold",
    schedule: JSON.stringify({ interval: "5m" }),
    params: JSON.stringify({
        index: ["app-logs-*"],
        timeField: "@timestamp",
        aggType: "count",
        groupBy: "top",
        termField: "service",
        termSize: 10,
        timeWindowSize: 5,
        timeWindowUnit: "m",
        thresholdComparator: ">",
        threshold: [100],
    }),
    actions: JSON.stringify([{
        group: "threshold met",
        id: slackConnector.id,
        params: { text: "High error rate on {{context.group}}: {{context.value}} errors" },
    }]),
    enabled: true,
    spaceId: "application",
}, { provider: esProvider });

// SLO for API availability
const apiSlo = new elasticstack.kibana.Slo("api-slo", {
    name: "API Availability > 99.9%",
    indicator: JSON.stringify({
        type: "sli.kql.custom",
        params: {
            index: "app-logs-*",
            good: "status_code >= 200 AND status_code < 500",
            total: "*",
            filter: "service: \"api-gateway\"",
            timestampField: "@timestamp",
        },
    }),
    timeWindow: JSON.stringify({ duration: "30d", type: "rolling" }),
    budgetingMethod: "occurrences",
    objective: JSON.stringify({ target: 0.999 }),
    spaceId: "application",
}, { provider: esProvider });
```

## Step 6: Fleet Setup

```typescript
const agentPolicy = new elasticstack.fleet.AgentPolicy("app-agents", {
    name: "application-servers",
    namespace: "production",
    monitorLogs: true,
    monitorMetrics: true,
}, { provider: esProvider });

const systemIntegration = new elasticstack.fleet.Integration("system", {
    name: "system",
    version: "1.54.0",
}, { provider: esProvider });

const systemPolicy = new elasticstack.fleet.IntegrationPolicy("system-pol", {
    name: "system-monitoring",
    agentPolicyId: agentPolicy.id,
    integrationName: "system",
    integrationVersion: "1.54.0",
}, { provider: esProvider });
```

## Step 7: Invite Team Members

```typescript
const orgId = "my-org-id";

// Org admin with billing access
const orgAdmin = new elasticstack.cloud.OrganizationMember("org-admin", {
    organizationId: orgId,
    email: "cto@company.com",
    organizationOwner: true,
    billingAdmin: true,
    deploymentRoleAll: "admin",
});

// Developer: admin on staging/dev, editor on production
const developer = new elasticstack.cloud.OrganizationMember("developer", {
    organizationId: orgId,
    email: "developer@company.com",
    deploymentRoles: [
        { deploymentId: "prod-id", role: "editor" },
        { deploymentId: "staging-id", role: "admin" },
    ],
});

// Viewer: read-only across all deployments
const viewer = new elasticstack.cloud.OrganizationMember("viewer", {
    organizationId: orgId,
    email: "analyst@company.com",
    deploymentRoleAll: "viewer",
    expiresIn: "720h",
});
```

## Exports

```typescript
export const esEndpoint = cluster.monitorProperties.apply(p =>
    p!.elasticProperties!.elasticsearchEndpoint);
export const kibanaEndpoint = cluster.monitorProperties.apply(p =>
    p!.elasticProperties!.kibanaEndpoint);
export const elasticPassword = password.password;
export const spaceName = appSpace.name;
```
