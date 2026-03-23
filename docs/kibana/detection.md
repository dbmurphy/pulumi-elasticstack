# Security Detection

Manage Kibana security detection rules, exceptions, and value lists.

## Resources

- `elasticstack.kibana.SecurityDetectionRule`
- `elasticstack.kibana.SecurityEnableRule`
- `elasticstack.kibana.InstallPrebuiltRules`
- `elasticstack.kibana.SecurityExceptionList`
- `elasticstack.kibana.SecurityExceptionItem`
- `elasticstack.kibana.SecurityList`
- `elasticstack.kibana.SecurityListDataStreams`
- `elasticstack.kibana.SecurityListItem`

## SecurityDetectionRule

Create custom detection rules for threat detection.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

// Brute force detection
const bruteForce = new elasticstack.kibana.SecurityDetectionRule("brute-force", {
    name: "Brute Force Login Attempts",
    description: "Detects multiple failed login attempts from the same source",
    riskScore: 73,
    severity: "high",
    ruleType: "query",
    query: "event.action: \"authentication_failure\" AND event.outcome: \"failure\"",
    language: "kuery",
    indexPatterns: ["logs-*", "auditbeat-*"],
    interval: "5m",
    fromTime: "now-10m",
    enabled: true,
    tags: ["brute-force", "authentication"],
    spaceId: "security-ops",
});

// Suspicious process execution
const suspiciousProc = new elasticstack.kibana.SecurityDetectionRule("suspicious-proc", {
    name: "Suspicious PowerShell Execution",
    description: "Detects encoded PowerShell commands often used in attacks",
    riskScore: 85,
    severity: "critical",
    ruleType: "query",
    query: "process.name: \"powershell.exe\" AND process.args: (\"-enc\" OR \"-encodedcommand\")",
    language: "kuery",
    indexPatterns: ["winlogbeat-*", "logs-endpoint*"],
    interval: "1m",
    enabled: true,
    tags: ["windows", "execution", "powershell"],
    spaceId: "security-ops",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | Rule name |
| `description` | string | Rule description |
| `riskScore` | int | Risk score (0-100) |
| `severity` | string | `"low"`, `"medium"`, `"high"`, or `"critical"` |
| `ruleType` | string | `"query"`, `"threshold"`, `"eql"`, `"machine_learning"` |
| `query` | string | Detection query |
| `language` | string | `"kuery"` or `"lucene"` |
| `indexPatterns` | string[] | Index patterns to search |
| `interval` | string | Evaluation interval |
| `fromTime` | string | Lookback period (e.g., "now-10m") |
| `enabled` | bool | Whether the rule is active |
| `tags` | string[] | MITRE ATT&CK tags, custom tags |

## SecurityEnableRule

Enable or disable a prebuilt detection rule by its rule ID.

```typescript
// Install prebuilt rules first
const prebuilt = new elasticstack.kibana.InstallPrebuiltRules("prebuilt", {
    spaceId: "security-ops",
});

// Enable specific prebuilt rules
const sshBrute = new elasticstack.kibana.SecurityEnableRule("ssh-brute", {
    ruleId: "ssh_brute_force_attempt",
    enabled: true,
    spaceId: "security-ops",
});

const privEsc = new elasticstack.kibana.SecurityEnableRule("priv-esc", {
    ruleId: "privilege_escalation_via_sudo",
    enabled: true,
    spaceId: "security-ops",
});
```

## InstallPrebuiltRules

Install all Elastic prebuilt detection rules in a space.

```typescript
const prebuilt = new elasticstack.kibana.InstallPrebuiltRules("install", {
    spaceId: "security-ops",
});
```

## SecurityExceptionList

Create exception list containers for grouping exception items.

```typescript
const falsePositives = new elasticstack.kibana.SecurityExceptionList("false-positives", {
    name: "Known False Positives",
    description: "Exception list for known benign activities",
    listType: "detection",
    tags: ["false-positive", "tuning"],
    spaceId: "security-ops",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `name` | string | List name |
| `description` | string | List description |
| `listType` | string | `"detection"` or `"endpoint"` |
| `tags` | string[] | Tags |
| `spaceId` | string | Kibana space |

## SecurityExceptionItem

Add specific exceptions to an exception list.

```typescript
const scannerException = new elasticstack.kibana.SecurityExceptionItem("scanner", {
    listId: falsePositives.id,
    entryValue: "vulnerability-scanner.internal.company.com",
    entryField: "source.address",
    entryType: "match",
    osTypes: [],
    comments: "Known vulnerability scanner - exclude from brute force alerts",
    spaceId: "security-ops",
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `listId` | string | Parent exception list ID |
| `entryValue` | string | Value to match |
| `entryField` | string | Field to match against |
| `entryType` | string | `"match"`, `"match_any"`, `"list"`, `"exists"` |
| `osTypes` | string[] | OS filters: `"windows"`, `"linux"`, `"macos"` |
| `comments` | string | Explanation for the exception |

## SecurityList

Manage custom value lists for threat indicators and allowlists.

```typescript
const maliciousIps = new elasticstack.kibana.SecurityList("bad-ips", {
    name: "Known Malicious IPs",
    description: "Threat intelligence IP blocklist",
    type: "ip",
    values: ["192.168.1.100", "10.0.0.50", "203.0.113.42"],
    spaceId: "security-ops",
});

const trustedDomains = new elasticstack.kibana.SecurityList("trusted-domains", {
    name: "Trusted Domains",
    description: "Known-good domains to exclude from alerts",
    type: "keyword",
    values: ["company.com", "trusted-partner.com", "cdn.company.com"],
    spaceId: "security-ops",
});
```

## SecurityListItem

Add individual items to a value list.

```typescript
const newIp = new elasticstack.kibana.SecurityListItem("new-threat", {
    listId: maliciousIps.id,
    value: "198.51.100.23",
    spaceId: "security-ops",
});
```

## SecurityListDataStreams

Associate data streams with a security list.

```typescript
const dsAssoc = new elasticstack.kibana.SecurityListDataStreams("threat-ds", {
    listId: maliciousIps.id,
    dataStream: "logs-threat_intel.*",
    spaceId: "security-ops",
});
```
