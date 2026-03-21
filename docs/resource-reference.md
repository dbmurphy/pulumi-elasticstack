# Resource Reference

Quick-reference for all 57 resources and 1 function.

## Elasticsearch (26 resources + 1 function)

| Resource | Description | Docs |
|----------|-------------|------|
| `Index` | Manages an Elasticsearch index | [index-management](elasticsearch/index-management.md) |
| `IndexAlias` | Manages an index alias | [index-management](elasticsearch/index-management.md) |
| `DataStream` | Manages a data stream | [index-management](elasticsearch/index-management.md) |
| `DataStreamLifecycle` | Manages data stream lifecycle config | [index-management](elasticsearch/index-management.md) |
| `IndexTemplate` | Manages an index template | [templates](elasticsearch/templates.md) |
| `ComponentTemplate` | Manages a component template | [templates](elasticsearch/templates.md) |
| `IndexTemplateIlmAttachment` | Attaches ILM policy to an index template | [templates](elasticsearch/templates.md) |
| `IndexLifecycle` | Manages an ILM policy | [lifecycle](elasticsearch/lifecycle.md) |
| `SnapshotLifecycle` | Manages an SLM policy | [lifecycle](elasticsearch/lifecycle.md) |
| `SnapshotRepository` | Manages a snapshot repository | [lifecycle](elasticsearch/lifecycle.md) |
| `Pipeline` | Manages an ingest pipeline | [ingest-and-transform](elasticsearch/ingest-and-transform.md) |
| `Transform` | Manages a transform | [ingest-and-transform](elasticsearch/ingest-and-transform.md) |
| `User` | Manages a security user | [security](elasticsearch/security.md) |
| `SystemUser` | Manages built-in system user passwords | [security](elasticsearch/security.md) |
| `Role` | Manages a security role | [security](elasticsearch/security.md) |
| `RoleMapping` | Manages a security role mapping | [security](elasticsearch/security.md) |
| `ApiKey` | Creates an API key | [security](elasticsearch/security.md) |
| `AnomalyDetectionJob` | Manages an ML anomaly detection job | [ml](elasticsearch/ml.md) |
| `Datafeed` | Manages an ML datafeed | [ml](elasticsearch/ml.md) |
| `DatafeedState` | Manages datafeed running state | [ml](elasticsearch/ml.md) |
| `MlJobState` | Manages ML job running state | [ml](elasticsearch/ml.md) |
| `Watch` | Manages a Watcher watch | [advanced](elasticsearch/advanced.md) |
| `EnrichPolicy` | Manages an enrich policy | [advanced](elasticsearch/advanced.md) |
| `Script` | Manages a stored script | [advanced](elasticsearch/advanced.md) |
| `LogstashPipeline` | Manages a Logstash pipeline | [advanced](elasticsearch/advanced.md) |
| `Settings` | Manages cluster-level settings | [advanced](elasticsearch/advanced.md) |
| **Function** | | |
| `getInfo` | Get cluster info (version, name, UUID) | [getting-started](getting-started.md) |

## Kibana (21 resources)

| Resource | Description | Docs |
|----------|-------------|------|
| `Space` | Manages a Kibana space | [spaces-and-security](kibana/spaces-and-security.md) |
| `SecurityRole` | Manages a Kibana security role | [spaces-and-security](kibana/spaces-and-security.md) |
| `ActionConnector` | Manages an action connector | [alerting](kibana/alerting.md) |
| `Rule` | Manages an alerting rule | [alerting](kibana/alerting.md) |
| `MaintenanceWindow` | Manages a maintenance window | [alerting](kibana/alerting.md) |
| `DataView` | Manages a data view (index pattern) | [data-views](kibana/data-views.md) |
| `DefaultDataView` | Sets the default data view | [data-views](kibana/data-views.md) |
| `ImportSavedObjects` | Imports saved objects from NDJSON | [data-views](kibana/data-views.md) |
| `Slo` | Manages a Service Level Objective | [slo](kibana/slo.md) |
| `SecurityDetectionRule` | Manages a security detection rule | [detection](kibana/detection.md) |
| `SecurityEnableRule` | Enables/disables a prebuilt detection rule | [detection](kibana/detection.md) |
| `InstallPrebuiltRules` | Installs all prebuilt detection rules | [detection](kibana/detection.md) |
| `SecurityExceptionList` | Manages an exception list container | [detection](kibana/detection.md) |
| `SecurityExceptionItem` | Manages an exception list item | [detection](kibana/detection.md) |
| `SecurityList` | Manages a custom value list | [detection](kibana/detection.md) |
| `SecurityListDataStreams` | Associates data stream with a list | [detection](kibana/detection.md) |
| `SecurityListItem` | Manages values within a list | [detection](kibana/detection.md) |
| `Monitor` | Manages a synthetics monitor | [synthetics](kibana/synthetics.md) |
| `Parameter` | Manages a synthetics parameter | [synthetics](kibana/synthetics.md) |
| `SyntheticsPrivateLocation` | Manages a synthetics private location | [synthetics](kibana/synthetics.md) |
| `Dashboard` | Manages a Kibana dashboard | [dashboards](kibana/dashboards.md) |

## Fleet (5 resources)

| Resource | Description | Docs |
|----------|-------------|------|
| `AgentPolicy` | Manages a Fleet agent policy | [fleet](fleet/fleet.md) |
| `Integration` | Manages a Fleet integration package | [fleet](fleet/fleet.md) |
| `IntegrationPolicy` | Manages a Fleet integration policy | [fleet](fleet/fleet.md) |
| `Output` | Manages a Fleet output config | [fleet](fleet/fleet.md) |
| `ServerHost` | Manages a Fleet Server host | [fleet](fleet/fleet.md) |

## APM (1 resource)

| Resource | Description | Docs |
|----------|-------------|------|
| `AgentConfiguration` | Manages APM agent configuration | [apm](apm/apm.md) |

## Cloud (4 resources)

| Resource | Description | Docs |
|----------|-------------|------|
| `DeploymentPassword` | Resets elastic user password | [cloud](cloud/cloud.md) |
| `OrganizationMember` | Invites org member with role assignments | [cloud](cloud/cloud.md) |
| `TrafficFilter` | Manages network security rulesets (IP, Private Link, egress) | [cloud](cloud/cloud.md) |
| `TrafficFilterAssociation` | Attaches a traffic filter to a deployment | [cloud](cloud/cloud.md) |
