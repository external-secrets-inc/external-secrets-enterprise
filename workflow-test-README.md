# Workflow Testing Script

This script deploys External Secrets Operator (ESO) workflows that don't depend on external stores (like AWS, Vault, or GCP), waits for their execution, and checks for any errors.

## Prerequisites

1. A running kind cluster
2. kubectl configured to use the kind cluster
3. External Secrets Operator (ESO) running locally on your computer

## Workflows Tested

The script tests the following workflows that use fake providers and don't require external secret stores:

1. `config/samples/workflows/jobs/switch.yaml` - Demonstrates switch job execution with a fake provider
2. `config/samples/workflows/steps/sensitive-outputs.yaml` - Demonstrates sensitive outputs handling with a fake provider

## Usage

1. Make sure your kind cluster is running
2. Make sure ESO is running locally on your computer
3. Run the script:

```bash
./deploy-workflows.sh
```

## What the Script Does

1. Deploys the workflows using kubectl
2. Waits for the WorkflowRun resources to complete (with a timeout of 5 minutes)
3. Checks the status of the WorkflowRun resources to see if there were any errors
4. Saves logs and details to a temporary directory for inspection

## Output

The script will output:
- The status of each workflow deployment
- The status of each WorkflowRun
- A summary of successful and failed workflows
- The location of logs and details for inspection

## Example Output

```
Starting workflow deployment and verification script...
Created temporary directory for logs: /tmp/tmp.XXXXXXXXXX
Deploying workflow: switch
Workflow deployed: switch
Waiting for WorkflowRun switch-job-production to complete...
Current status: Running
Current status: Running
WorkflowRun switch-job-production completed successfully
WorkflowRun details saved to /tmp/tmp.XXXXXXXXXX/switch-job-production-details.yaml
Waiting for WorkflowRun switch-job-staging to complete...
Current status: Running
WorkflowRun switch-job-staging completed successfully
WorkflowRun details saved to /tmp/tmp.XXXXXXXXXX/switch-job-staging-details.yaml
Workflow config/samples/workflows/jobs/switch.yaml completed successfully
----------------------------------------
Deploying workflow: sensitive-outputs
Workflow deployed: sensitive-outputs
Waiting for WorkflowRun sensitive-outputs-run to complete...
Current status: Running
WorkflowRun sensitive-outputs-run completed successfully
WorkflowRun details saved to /tmp/tmp.XXXXXXXXXX/sensitive-outputs-run-details.yaml
Workflow config/samples/workflows/steps/sensitive-outputs.yaml completed successfully
----------------------------------------
Workflow deployment and verification completed
Total workflows: 2
Successful: 2
Failed: 0
Logs and details are available in /tmp/tmp.XXXXXXXXXX
All workflows completed successfully!
```

## Troubleshooting

If any workflows fail, the script will:
1. Save the details of the failed WorkflowRun to the temporary directory
2. Print the location of the error logs
3. Exit with a non-zero status code

Check the logs in the temporary directory for details on why the workflow failed.