# APM

Manage APM agent configurations.

## Resources

- `elasticstack.apm.AgentConfiguration`

## AgentConfiguration

Configure APM agent settings centrally. Agents fetch these settings on startup and periodically refresh them, so you can tune tracing without redeploying applications.

```typescript
import * as elasticstack from "@pulumi/elasticstack";

// Global defaults for all services
const globalConfig = new elasticstack.apm.AgentConfiguration("global", {
    serviceName: "",
    serviceEnvironment: "",
    settings: JSON.stringify({
        transaction_sample_rate: "0.5",
        capture_body: "off",
        log_level: "warning",
        span_frames_min_duration: "5ms",
    }),
});

// Service-specific: high-traffic API
const apiConfig = new elasticstack.apm.AgentConfiguration("api", {
    serviceName: "api-gateway",
    serviceEnvironment: "production",
    settings: JSON.stringify({
        transaction_sample_rate: "0.1",
        capture_body: "errors",
        transaction_max_spans: "500",
        stack_trace_limit: "50",
    }),
});

// Service-specific: payment service (full capture for compliance)
const paymentConfig = new elasticstack.apm.AgentConfiguration("payments", {
    serviceName: "payment-service",
    serviceEnvironment: "production",
    settings: JSON.stringify({
        transaction_sample_rate: "1.0",
        capture_body: "all",
        capture_headers: "true",
        log_level: "info",
    }),
});

// Development environment: full sampling
const devConfig = new elasticstack.apm.AgentConfiguration("dev", {
    serviceName: "",
    serviceEnvironment: "development",
    settings: JSON.stringify({
        transaction_sample_rate: "1.0",
        capture_body: "all",
        log_level: "debug",
    }),
});
```

### Key Inputs

| Input | Type | Description |
|-------|------|-------------|
| `serviceName` | string | Service name filter (empty = all services) |
| `serviceEnvironment` | string | Environment filter (empty = all environments) |
| `agentName` | string | Agent type filter (e.g., "nodejs", "python") |
| `settings` | string (JSON) | Key-value agent settings |

### Common Settings

| Setting | Description | Example Values |
|---------|-------------|----------------|
| `transaction_sample_rate` | Fraction of transactions to trace | `"0.1"` (10%), `"1.0"` (100%) |
| `capture_body` | When to capture request bodies | `"off"`, `"errors"`, `"all"` |
| `capture_headers` | Capture HTTP headers | `"true"`, `"false"` |
| `transaction_max_spans` | Max spans per transaction | `"500"` |
| `log_level` | Agent log level | `"debug"`, `"info"`, `"warning"`, `"error"` |
| `stack_trace_limit` | Max stack frames to collect | `"50"` |
| `span_frames_min_duration` | Min duration to collect stack frames | `"5ms"` |

### Precedence

Configurations are matched from most specific to least specific:

1. Service name + environment (e.g., `api-gateway` in `production`)
2. Service name only (e.g., `api-gateway` in any environment)
3. Environment only (e.g., all services in `production`)
4. Global (empty service name and environment)
