# Workflow

The `Workflow` is a namespaced resource that allows you to orchestrate complex secret management operations. It provides a way to define a sequence of jobs and steps to pull secrets from external providers, transform them, generate new secrets, and push them to various destinations.

* Defines a directed acyclic graph (DAG) of jobs that can depend on each other
* Each job contains steps that perform specific operations like pulling, pushing, or transforming secrets
* Supports data sharing between steps and jobs through templating
* Can be scheduled to run periodically

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: Workflow
metadata:
  name: example-workflow
  annotations:
    workflows.external-secrets.io/description: "Example workflow"
spec:
  version: v1alpha1
  name: example-workflow
  schedule:
    every: "1h"  # Run every hour
  jobs:
    generateSecret:
      standard:
        steps:
          - name: generateApiKey
            generator:
              kind: Password
              spec:
                passwordSpec:
                  length: 32
                  digits: 5
                  symbols: 5
```

## Workflow Architecture

A workflow consists of the following key components:

- **Jobs**: Individual units of work that can depend on other jobs
- **Steps**: Atomic operations within a job that perform specific actions
- **Variables**: Data that can be shared between steps and jobs

Workflows follow a state machine model with different phases:
- **Pending**: The workflow is waiting to be processed
- **Scheduled**: The workflow is scheduled to run at a future time
- **Running**: The workflow is currently executing
- **Succeeded**: The workflow completed successfully
- **Failed**: The workflow encountered an error and failed

## Job Types

### Standard Job

Executes a sequence of steps in order.

```yaml
jobs:
  fetchSecrets:
    standard:
      steps:
        - name: pullFromAWS
          pull:
            source:
              name: aws-store
              kind: SecretStore
            data:
              - remoteRef:
                  key: my-secret
                secretKey: MY_SECRET
```

### Loop Job

Executes steps in a loop over a range of values.

```yaml
jobs:
  processItems:
    loop:
      concurrency: 2  # Number of iterations to run in parallel (0 = unlimited)
      range: "{{ .global.jobs.createTargets.targetStores }}"  # Template string that resolves to a map/array
      steps:
        - name: print
          debug:
            message: "Processing: {{ .range.value }}"
```

### Conditional Job

Executes different branches of steps based on conditions.

```yaml
jobs:
  processBasedOnEnvironment:
    conditional:
      branches:
        - condition: "{{ eq .global.variables.environment \"production\" }}"
          steps:
            - name: productionSetup
              debug:
                message: "Setting up for production environment"
        - condition: "{{ eq .global.variables.environment \"staging\" }}"
          steps:
            - name: stagingSetup
              debug:
                message: "Setting up for staging environment"
        - condition: "{{ eq .global.variables.environment \"development\" }}"
          steps:
            - name: devSetup
              debug:
                message: "Setting up for development environment"
```

## Step Types

### Pull Step

Pulls secrets from external providers using a SecretStore.

```yaml
- name: pullSecret
  pull:
    source:
      name: aws-store
      kind: SecretStore
    data:
      - remoteRef:
          key: database-credentials
        secretKey: DB_PASSWORD
    dataFrom:
      - extract:
          key: api-config
```

### Push Step

Pushes secrets to external providers.

```yaml
- name: pushSecret
  push:
    secretSource: ".global.jobs.createSecret.createSecret"
    destination:
      storeRef:
        name: "aws-store"
        kind: SecretStore
    data:
      - match:
          secretKey: "api_key"
          remoteRef:
            remoteKey: "new-secret"
            property: "api_key"
```

### Transform Step

Transforms data using templates or mappings.

```yaml
- name: transformData
  transform:
    mappings:
      api_key: "{{ .global.jobs.generateSecret.generateApiKey.password }}"
      username: "app-{{ .global.jobs.generateId.generateUUID.uuid | substr 0 8 }}"
```

### Generator Step

Generates new secrets using ESO's built-in generators.

```yaml
- name: generateApiKey
  generator:
    kind: Password
    spec:
      passwordSpec:
        length: 32
        digits: 5
        symbols: 5
        symbolCharacters: "-_$@"
        noUpper: false
        allowRepeat: true
```

### JavaScript Step

Executes JavaScript code to process data.

```yaml
- name: processData
  javascript:
    script: |
      console.log("Processing data...");
      
      // Set a string value
      setString("name", "Alice");
      
      // Set a boolean value
      setBool("active", true);
      
      // Set a numeric value
      setNumber("score", 42);
      
      // Set a JSON object
      setJSON("config", {
        "environment": "production",
        "features": {
          "analytics": true
        }
      });
  outputs:
    - name: name
      type: string
    - name: active
      type: bool
    - name: score
      type: number
    - name: config
      type: map
      sensitive: false
```

## Step Outputs

Each step can define its expected outputs using the `outputs` field. This helps document what outputs a step provides and allows for proper type handling and sensitivity marking.

```yaml
- name: generateApiKey
  generator:
    kind: Password
    spec:
      passwordSpec:
        length: 32
        digits: 5
        symbols: 5
  outputs:
    - name: password
      type: string
      sensitive: true
```

The following output types are supported:
- `bool`: Boolean values (true/false)
- `number`: Numeric values (float64)
- `string`: Text values
- `time`: Time values (RFC3339 format)
- `map`: Complex data structures (JSON objects)

Setting `sensitive: true` will mask the output value in the workflow status with asterisks (`********`).

### Debug Step

Outputs debug messages, useful for troubleshooting.

```yaml
- name: debugInfo
  debug:
    message: "Current value: {{ .global.jobs.generateSecret.generateApiKey.password }}"
```

## Data Access and Templating

Workflows use Go templating to access data from previous steps and jobs. The template context includes:

- `.global`: Access to all workflow data
- `.range`: In loop jobs, contains the current iteration's key and value
- `.input`: Input data passed to the step

Example template: `{{ .global.jobs.generateSecret.generateApiKey.password }}`

## Scheduling

Workflows can be scheduled to run periodically using either a simple interval or a cron expression:

```yaml
spec:
  schedule:
    every: "1h"  # Run every hour
    # Or use a cron expression
    # cron: "0 0 * * *"  # Run at midnight every day
```

## Complete Example

This example workflow generates a secret and pushes it to multiple secret stores:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: Workflow
metadata:
  name: push-to-multiple-stores
  annotations:
    workflows.external-secrets.io/description: "Workflow that pushes generated secrets to multiple stores"
spec:
  version: v1alpha1
  name: push-to-multiple-stores
  jobs:
    generateSecret:
      standard:
        steps:
          - name: generateApiKey
            generator:
              kind: Password
              spec:
                passwordSpec:
                  length: 42
                  digits: 5
                  symbols: 5
                  symbolCharacters: "-_$@"
                  noUpper: false
                  allowRepeat: true
            outputs:
              - name: password
                type: string
                sensitive: true

    createSecret:
      dependsOn:
        - generateSecret
      standard:
        steps:
          - name: createSecret
            transform:
              mappings:
                api_key: "{{ .global.jobs.generateSecret.generateApiKey.password }}"
            outputs:
              - name: api_key
                type: string
                sensitive: true

          - name: createTargets
            javascript:
              script: |
                // Create an array to specify target stores
                setArray('targetStores', ['gcp-store', 'aws-store'])
                
                // Set environment variable for conditional job
                setString('environment', 'production')
            outputs:
              - name: targetStores
                type: map
              - name: environment
                type: string
                
    selectEnvironment:
      dependsOn:
        - createSecret
      conditional:
        branches:
          - condition: "{{ eq .global.jobs.createSecret.createTargets.environment \"production\" }}"
            steps:
              - name: productionConfig
                transform:
                  mappings:
                    retention_days: "90"
                    log_level: "error"
                    
          - condition: "{{ eq .global.jobs.createSecret.createTargets.environment \"staging\" }}"
            steps:
              - name: stagingConfig
                transform:
                  mappings:
                    retention_days: "30"
                    log_level: "info"
                    
          - condition: "{{ eq .global.jobs.createSecret.createTargets.environment \"development\" }}"
            steps:
              - name: devConfig
                transform:
                  mappings:
                    retention_days: "7"
                    log_level: "debug"

    pushToStores:
      dependsOn:
        - createSecret
        - selectEnvironment
      loop:
        concurrency: 0
        range: "{{ .global.jobs.createSecret.createTargets.targetStores }}"
        steps:
          - name: push
            push:
              secretSource: ".global.jobs.createSecret.createSecret"
              destination:
                storeRef:
                  name: "{{ .range.value }}"
                  kind: SecretStore
              data:
                - match:
                    secretKey: "api_key"
                    remoteRef:
                      remoteKey: "new-secret"
                      property: "api_key"