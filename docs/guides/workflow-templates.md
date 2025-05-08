# Workflow Templates and External Triggering

This guide explains how to use workflow templates and external triggering capabilities in External Secrets Operator.

## Overview

Workflow Templates allow you to define reusable workflow patterns that can be instantiated with different parameters. External triggering enables programmatic creation of workflows through an API.

## WorkflowTemplate

A `WorkflowTemplate` defines a reusable workflow pattern with parameterization support.

```yaml
apiVersion: workflows.external-secrets.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: rotate-database-credentials
spec:
  version: v1
  name: "Database Credentials Rotation"
  parameters:
    - name: databaseName
      description: "Name of the database to rotate credentials for"
      required: true
    - name: secretStoreName
      description: "Name of the SecretStore to use"
      default: "vault-store"
  jobs:
    # Job definitions (same as in Workflow)
    fetch-current:
      standard:
        steps:
          - name: get-current-credentials
            # ...
```

### Template Parameters

Parameters allow you to customize the workflow when it's instantiated:

- `name`: Parameter name (required)
- `description`: Human-readable description
- `required`: Whether the parameter must be provided
- `default`: Default value if not provided

## WorkflowRun

A `WorkflowRun` instantiates a template with specific parameter values.

```yaml
apiVersion: workflows.external-secrets.io/v1alpha1
kind: WorkflowRun
metadata:
  name: rotate-production-db
spec:
  templateRef:
    name: rotate-database-credentials
  parameters:
    databaseName: "production-postgres"
    notificationChannel: "#prod-alerts"
```

### WorkflowRun Fields

- `templateRef`: Reference to the template to use
  - `name`: Template name
  - `namespace`: Template namespace (optional, defaults to WorkflowRun namespace)
- `parameters`: Map of parameter values
- `variables`: Additional variables to set in the workflow

## Template Syntax

Workflow templates use Go templating with some additional features to make them more concise and readable.

### Traditional Syntax

The traditional Go template syntax uses dot notation to access data:

```yaml
# Access global variables
{% raw %}{{ .global.variables.environment }}{% endraw %}

# Access job step outputs
{% raw %}{{ .global.jobs.generateSecret.generateApiKey.password }}{% endraw %}
```

### Shorthand $ Syntax

For improved readability, you can use the $ shorthand syntax:

```yaml
# Access global variables
{% raw %}{{ $environment }}{% endraw %}
# or directly in strings
"Environment: $environment"

# Access job step outputs
{% raw %}{{ $generateSecret.generateApiKey.password }}{% endraw %}
# or directly in strings
"Password: $generateSecret.generateApiKey.password"
```

### Examples

```yaml
# Transform step using $ syntax
- name: createAppConfig
  transform:
    mappings:
      api_key: "$generateSecret.generateApiKey.password"
      environment: "$environment"
      debug: "{% raw %}{{ eq $environment \"development\" }}{% endraw %}"

# Push step using $ syntax
- name: pushConfig
  push:
    secretSource: "$createSecret.createSecret"
    destination:
      storeRef:
        name: "example-store"
        kind: SecretStore
    data:
      - match:
          secretKey: "api_key"
          remoteRef:
            remoteKey: "$configKey"
            property: "api_key"
```

### Loop Job Context

In loop jobs, you can access the current iteration's key and value:

```yaml
- name: configureUser
  transform:
    mappings:
      username: "{% raw %}{{ .range.value.username }}{% endraw %}"
      # or with $ syntax for other references
      created_by: "$currentUser"
```

### Escaping $ Signs

If you need to use a literal $ sign (like for currency), you can escape it with double $$:

```yaml
# This will render as $100
price: "$$100"
```

## External API

The External Secrets Operator provides an HTTP API for triggering workflows programmatically.

### API Endpoints

- `POST /api/v1/namespaces/{namespace}/workflowruns`: Create a new WorkflowRun
- `GET /api/v1/namespaces/{namespace}/workflowruns`: List WorkflowRuns
- `GET /api/v1/namespaces/{namespace}/workflowruns/{name}`: Get a WorkflowRun

### Creating a WorkflowRun via API

```bash
curl -X POST \
  http://external-secrets-api:8080/api/v1/namespaces/default/workflowruns \
  -H 'Content-Type: application/json' \
  -d '{
    "templateName": "rotate-database-credentials",
    "parameters": {
      "databaseName": "production-postgres",
      "notificationChannel": "#prod-alerts"
    }
  }'
```

### API Response

```json
{
  "name": "rotate-database-credentials-abc123",
  "namespace": "default",
  "status": "created",
  "message": "WorkflowRun created successfully"
}
```

## Use Cases

Workflow templates and external triggering enable several powerful use cases:

1. **Scheduled Secret Rotation**: Use a CronJob to trigger secret rotation workflows on a schedule
2. **CI/CD Integration**: Trigger secret operations from your CI/CD pipeline
3. **Event-Driven Secret Management**: React to events by triggering appropriate workflows
4. **Self-Service Portal**: Build a UI that allows developers to trigger approved secret operations

## Configuration

To enable the workflow API server, set the following flags when starting the controller:

```
--enable-workflow-api=true
--workflow-api-port=:8080
```

## Security Considerations

- Implement proper RBAC for template usage and API access
- Consider using network policies to restrict access to the API server
- Use TLS for securing API communications
