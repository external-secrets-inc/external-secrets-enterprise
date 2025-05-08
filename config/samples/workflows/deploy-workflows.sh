#!/bin/bash

# Script to deploy workflows, wait for their execution, and check for errors
# This script only deploys workflows that don't depend on external stores like AWS, Vault, or GCP

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting workflow deployment and verification script...${NC}"

# Function to check if a command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Check if kubectl is installed
if ! command_exists kubectl; then
  echo -e "${RED}Error: kubectl is not installed. Please install kubectl and try again.${NC}"
  exit 1
fi

# Check if kind cluster is running
if ! kubectl cluster-info &>/dev/null; then
  echo -e "${RED}Error: Cannot connect to Kubernetes cluster. Please ensure your kind cluster is running.${NC}"
  exit 1
fi

# Create a temporary directory for logs
TEMP_DIR=$(mktemp -d)
echo -e "${GREEN}Created temporary directory for logs: ${TEMP_DIR}${NC}"

# Function to undeploy all existing workflowruns and workflows
undeploy_existing_workflows() {
  echo -e "${YELLOW}Undeploying all existing workflowruns and workflows...${NC}"

  # Get all workflowruns
  local workflowruns=$(kubectl get workflowruns -o name 2>/dev/null || echo "")

  if [[ -n "$workflowruns" ]]; then
    echo -e "${YELLOW}Found existing workflowruns. Deleting...${NC}"
    kubectl delete workflowruns --all > "${TEMP_DIR}/delete-workflowruns.log" 2>&1

    if [ $? -ne 0 ]; then
      echo -e "${RED}Failed to delete workflowruns${NC}"
      cat "${TEMP_DIR}/delete-workflowruns.log"
      return 1
    fi

    echo -e "${GREEN}All workflowruns deleted${NC}"

    # Wait for all workflowruns to be fully deleted
    echo -e "${YELLOW}Waiting for workflowruns to be fully deleted...${NC}"
    local timeout=60
    local start_time=$(date +%s)

    while true; do
      local remaining=$(kubectl get workflowruns --no-headers 2>/dev/null | wc -l | tr -d ' ')

      if [[ "$remaining" -eq 0 ]]; then
        echo -e "${GREEN}All workflowruns have been fully deleted${NC}"
        break
      fi

      echo -e "${YELLOW}Waiting for $remaining workflowruns to be deleted...${NC}"

      # Check if we've exceeded the timeout
      local current_time=$(date +%s)
      if (( current_time - start_time > timeout )); then
        echo -e "${RED}Timeout waiting for workflowruns to be deleted${NC}"
        return 1
      fi

      # Wait for 2 seconds before checking again
      sleep 2
    done
  else
    echo -e "${GREEN}No existing workflowruns found${NC}"
  fi

  # Get all workflows
  local workflows=$(kubectl get workflows -o name 2>/dev/null || echo "")

  if [[ -n "$workflows" ]]; then
    echo -e "${YELLOW}Found existing workflows. Deleting...${NC}"
    kubectl delete workflows --all > "${TEMP_DIR}/delete-workflows.log" 2>&1

    if [ $? -ne 0 ]; then
      echo -e "${RED}Failed to delete workflows${NC}"
      cat "${TEMP_DIR}/delete-workflows.log"
      return 1
    fi

    echo -e "${GREEN}All workflows deleted${NC}"

    # Wait for all workflows to be fully deleted
    echo -e "${YELLOW}Waiting for workflows to be fully deleted...${NC}"
    local timeout=60
    local start_time=$(date +%s)

    while true; do
      local remaining=$(kubectl get workflows --no-headers 2>/dev/null | wc -l | tr -d ' ')

      if [[ "$remaining" -eq 0 ]]; then
        echo -e "${GREEN}All workflows have been fully deleted${NC}"
        break
      fi

      echo -e "${YELLOW}Waiting for $remaining workflows to be deleted...${NC}"

      # Check if we've exceeded the timeout
      local current_time=$(date +%s)
      if (( current_time - start_time > timeout )); then
        echo -e "${RED}Timeout waiting for workflows to be deleted${NC}"
        return 1
      fi

      # Wait for 2 seconds before checking again
      sleep 2
    done
  else
    echo -e "${GREEN}No existing workflows found${NC}"
  fi

  echo -e "${GREEN}Environment is clean and ready for testing${NC}"
  return 0
}

# Function to generate a formatted table for workflow status
# Parameters:
#   $1 - check_failed: If set to "check", will check for failed workflows and return status (default: "check")
generate_workflow_status_table() {
  local check_failed=${1:-"check"}
  
  if [[ "$check_failed" == "check" ]]; then
    echo -e "${YELLOW}Generating workflow status table and checking for failures...${NC}"
  else
    echo -e "${YELLOW}Generating workflow status table...${NC}"
  fi
  
  # Save all workflows data to a temporary file
  kubectl get workflows -o json > "${TEMP_DIR}/workflows-data.json" 2>&1
  
  if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to get workflow data${NC}"
    cat "${TEMP_DIR}/workflows-data.json"
    return 1
  fi
  
  # Print table header
  printf "┌─────────────────────────────────┬────────────┬───────────────┬────────────────────┬────────────────────┐\n"
  printf "│ %-33s │ %-10s │ %-13s │ %-18s │ %-18s │\n" "WORKFLOW NAME" "STATUS" "WORKFLOWRUNS" "CREATED" "LAST UPDATED"
  printf "├─────────────────────────────────┼────────────┼───────────────┼────────────────────┼────────────────────┤\n"
  
  # Get all workflow names
  local workflow_names=($(kubectl get workflows -o jsonpath='{.items[*].metadata.name}' 2>/dev/null))
  
  # Process each workflow
  for workflow in "${workflow_names[@]}"; do
    # Get workflow details
    local phase=$(kubectl get workflow "$workflow" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
    local created=$(kubectl get workflow "$workflow" -o jsonpath='{.metadata.creationTimestamp}' 2>/dev/null | cut -d'T' -f1,2 | sed 's/T/ /' | cut -d'.' -f1)
    local updated=$(kubectl get workflow "$workflow" -o jsonpath='{.status.lastUpdated}' 2>/dev/null | cut -d'T' -f1,2 | sed 's/T/ /' | cut -d'.' -f1)
    
    # Get associated workflowruns
    local workflowruns=$(kubectl get workflowruns --selector=workflows.external-secrets.io/workflow-name="$workflow" --no-headers 2>/dev/null | wc -l | tr -d ' ')
    
    # Set color based on status
    local status_color="${NC}"
    if [[ "$phase" == "Succeeded" ]]; then
      status_color="${GREEN}"
    elif [[ "$phase" == "Failed" ]]; then
      status_color="${RED}"
    elif [[ "$phase" == "Running" ]]; then
      status_color="${YELLOW}"
    fi
    
    # Print row with colored status
    printf "│ %-33s │ ${status_color}%-10s${NC} │ %-13s │ %-18s │ %-18s │\n" "$workflow" "$phase" "$workflowruns" "$created" "$updated"
  done
  
  # Print table footer
  printf "└─────────────────────────────────┴────────────┴───────────────┴────────────────────┴────────────────────┘\n"
  
  # Only check for failed workflows if requested
  if [[ "$check_failed" == "check" ]]; then
    # Check if any workflows are not in Succeeded state
    local failed_workflows=$(kubectl get workflows -o jsonpath='{.items[?(@.status.phase!="Succeeded")].metadata.name}' 2>/dev/null)
    
    if [[ -n "$failed_workflows" ]]; then
      echo -e "${RED}The following workflows are not in Succeeded state: ${failed_workflows}${NC}"
      
      # Get details for each failed workflow
      for workflow in $failed_workflows; do
        echo -e "${YELLOW}Details for workflow ${workflow}:${NC}"
        kubectl get workflow "$workflow" -o yaml > "${TEMP_DIR}/${workflow}-details.yaml"
        echo -e "${RED}Details saved to ${TEMP_DIR}/${workflow}-details.yaml${NC}"
      done
      
      return 1
    fi
    
    echo -e "${GREEN}All workflows are in Succeeded state${NC}"
  fi
  
  return 0
}

# Function to check workflow status
check_workflow_status() {
  echo -e "${YELLOW}Checking status of all workflows...${NC}"
  
  # Generate and display the workflow status table
  if ! generate_workflow_status_table; then
    return 1
  fi
  
  return 0
}

# Function to deploy a workflow and wait for its completion
deploy_and_check_workflow() {
  local workflow_file=$1
  local workflow_name=$(basename "$workflow_file" .yaml)

  echo -e "${YELLOW}Deploying workflow: ${workflow_name}${NC}"

  # Check if the file exists
  if [ ! -f "$workflow_file" ]; then
    echo -e "${RED}Workflow file not found: ${workflow_file}${NC}"
    return 1
  fi

  # Count the number of WorkflowRun definitions in the file
  local workflowrun_count=$(grep -c "kind: WorkflowRun" "$workflow_file")
  
  if [ "$workflowrun_count" -eq 0 ]; then
    echo -e "${YELLOW}No WorkflowRun definitions found in ${workflow_file}, skipping...${NC}"
    return 0
  fi
  
  echo -e "${GREEN}Found ${workflowrun_count} WorkflowRun definition(s) in ${workflow_file}${NC}"

  # Apply the workflow YAML
  kubectl apply -f "$workflow_file" > "${TEMP_DIR}/${workflow_name}-apply.log" 2>&1

  if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to deploy workflow: ${workflow_name}${NC}"
    cat "${TEMP_DIR}/${workflow_name}-apply.log"
    return 1
  fi

  echo -e "${GREEN}Workflow deployed: ${workflow_name}${NC}"

  # Extract WorkflowRun names from the YAML file
  # This improved version handles YAML structure better by looking for "name:" after "kind: WorkflowRun" and "metadata:"
  local workflow_runs=$(awk '/kind: WorkflowRun/{flag=1} flag&&/metadata:/{flag=2} flag==2&&/name:/{print $2; flag=0}' "$workflow_file" | tr -d '"')

  # Check if we found any workflow runs
  if [ -z "$workflow_runs" ]; then
    echo -e "${YELLOW}Could not extract WorkflowRun names from ${workflow_file}, checking for created WorkflowRuns...${NC}"
    # Try to find WorkflowRuns created in the last minute that might be related to this workflow
    # This works on both macOS and Linux
    local one_minute_ago=""
    if [[ "$OSTYPE" == "darwin"* ]]; then
      # macOS date command
      one_minute_ago=$(date -u -v-1M +"%Y-%m-%dT%H:%M:%SZ")
    else
      # Linux date command
      one_minute_ago=$(date -u -d "1 minute ago" +"%Y-%m-%dT%H:%M:%SZ")
    fi
    local recent_runs=$(kubectl get workflowruns --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[?(@.metadata.creationTimestamp>="'$one_minute_ago'")].metadata.name}')
    if [ -n "$recent_runs" ]; then
      echo -e "${GREEN}Found recently created WorkflowRuns: ${recent_runs}${NC}"
      workflow_runs=$recent_runs
    else
      echo -e "${YELLOW}No recently created WorkflowRuns found, continuing...${NC}"
      return 0
    fi
  fi

  local run_count=0
  local failed_count=0

  for run in $workflow_runs; do
    run_count=$((run_count + 1))
    echo -e "${YELLOW}Waiting for WorkflowRun ${run} to complete (${run_count}/${workflowrun_count})...${NC}"

    # Wait for the WorkflowRun to complete (timeout after 5 minutes)
    local timeout=300
    local start_time=$(date +%s)
    local status=""

    while true; do
      # Get the current status of the WorkflowRun
      status=$(kubectl get workflowrun "$run" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")

      # Check if the WorkflowRun has completed
      if [[ "$status" == "Succeeded" ]]; then
        echo -e "${GREEN}WorkflowRun ${run} completed successfully${NC}"
        break
      elif [[ "$status" == "Failed" ]]; then
        echo -e "${RED}WorkflowRun ${run} failed${NC}"
        kubectl get workflowrun "$run" -o yaml > "${TEMP_DIR}/${run}-status.yaml"
        echo -e "${RED}Details saved to ${TEMP_DIR}/${run}-status.yaml${NC}"
        failed_count=$((failed_count + 1))
        break
      elif [[ "$status" == "NotFound" ]]; then
        echo -e "${YELLOW}Waiting for WorkflowRun ${run} to be created...${NC}"
      else
        echo -e "${YELLOW}Current status: ${status}${NC}"
      fi

      # Check if we've exceeded the timeout
      local current_time=$(date +%s)
      if (( current_time - start_time > timeout )); then
        echo -e "${RED}Timeout waiting for WorkflowRun ${run} to complete${NC}"
        kubectl get workflowrun "$run" -o yaml > "${TEMP_DIR}/${run}-timeout.yaml" 2>/dev/null || echo "WorkflowRun not found" > "${TEMP_DIR}/${run}-timeout.log"
        echo -e "${RED}Details saved to ${TEMP_DIR}/${run}-timeout.yaml${NC}"
        failed_count=$((failed_count + 1))
        break
      fi

      # Wait for 5 seconds before checking again
      sleep 5
    done

    if [[ "$status" == "Succeeded" ]]; then
      # Get the details of the completed WorkflowRun
      kubectl get workflowrun "$run" -o yaml > "${TEMP_DIR}/${run}-details.yaml"
      echo -e "${GREEN}WorkflowRun details saved to ${TEMP_DIR}/${run}-details.yaml${NC}"

      # Check for any errors in the WorkflowRun
      local error_count=$(kubectl get workflowrun "$run" -o jsonpath='{.status.conditions[?(@.type=="Error")].status}' 2>/dev/null | grep -c "True" || echo "0")

      if [[ "$error_count" -gt 0 ]]; then
        echo -e "${RED}Errors found in WorkflowRun ${run}${NC}"
        kubectl get workflowrun "$run" -o jsonpath='{.status.conditions[?(@.type=="Error")]}' > "${TEMP_DIR}/${run}-errors.json"
        echo -e "${RED}Error details saved to ${TEMP_DIR}/${run}-errors.json${NC}"
        failed_count=$((failed_count + 1))
      fi
    fi
  done

  if [[ "$failed_count" -gt 0 ]]; then
    echo -e "${RED}${failed_count} out of ${run_count} WorkflowRuns failed for ${workflow_name}${NC}"
    return 1
  else
    echo -e "${GREEN}All ${run_count} WorkflowRuns completed successfully for ${workflow_name}${NC}"
    return 0
  fi
}

# Dynamically discover workflow files that don't depend on external stores
# This makes the script more maintainable as new workflow files are added
echo -e "${YELLOW}Discovering workflow files...${NC}"
WORKFLOWS=()

# Check if jobs directory exists
if [ -d "./jobs" ]; then
  # Add files from the jobs directory
  for file in ./jobs/*.yaml; do
    # Skip if no files match the pattern
    [ -e "$file" ] || continue
    
    # Skip files that might depend on external stores (add patterns as needed)
    if ! grep -q "aws\|vault\|gcp\|azure" "$file"; then
      WORKFLOWS+=("$file")
      echo -e "${GREEN}Added workflow: $file${NC}"
    else
      echo -e "${YELLOW}Skipping workflow that may depend on external stores: $file${NC}"
    fi
  done
else
  echo -e "${YELLOW}Jobs directory not found, skipping...${NC}"
fi

# Check if steps directory exists
if [ -d "./steps" ]; then
  # Add files from the steps directory
  for file in ./steps/*.yaml; do
    # Skip if no files match the pattern
    [ -e "$file" ] || continue
    
    # Skip files that might depend on external stores (add patterns as needed)
    if ! grep -q "aws\|vault\|gcp\|azure" "$file"; then
      WORKFLOWS+=("$file")
      echo -e "${GREEN}Added workflow: $file${NC}"
    else
      echo -e "${YELLOW}Skipping workflow that may depend on external stores: $file${NC}"
    fi
  done
else
  echo -e "${YELLOW}Steps directory not found, skipping...${NC}"
fi

# Fallback to hardcoded list if no workflows were found
if [ ${#WORKFLOWS[@]} -eq 0 ]; then
  echo -e "${YELLOW}No workflows discovered, using default list${NC}"
  DEFAULT_WORKFLOWS=(
    "./jobs/loop.yaml"
    "./jobs/standard.yaml"
    "./jobs/switch.yaml"
    "./steps/debug.yaml"
    "./steps/generator.yaml"
    "./steps/javascript.yaml"
    "./steps/pull.yaml"
    "./steps/push.yaml"
    "./steps/transform.yaml"
    "./steps/sensitive-outputs.yaml"
  )
  
  # Check each file in the default list
  for file in "${DEFAULT_WORKFLOWS[@]}"; do
    if [ -f "$file" ]; then
      WORKFLOWS+=("$file")
      echo -e "${GREEN}Added default workflow: $file${NC}"
    else
      echo -e "${YELLOW}Default workflow file not found, skipping: $file${NC}"
    fi
  done
  
  if [ ${#WORKFLOWS[@]} -eq 0 ]; then
    echo -e "${RED}No workflow files found in default list. Please check the paths.${NC}"
    exit 1
  fi
fi

# Undeploy any existing workflowruns and workflows
if ! undeploy_existing_workflows; then
  echo -e "${RED}Failed to clean up existing workflows. Exiting.${NC}"
  exit 1
fi

# Deploy and check each workflow
FAILED=0
for workflow in "${WORKFLOWS[@]}"; do
  if deploy_and_check_workflow "$workflow"; then
    echo -e "${GREEN}Workflow $workflow completed successfully${NC}"
  else
    echo -e "${RED}Workflow $workflow failed${NC}"
    FAILED=$((FAILED + 1))
  fi
  echo "----------------------------------------"
done

# Check the status of all workflows
echo -e "${YELLOW}Checking overall workflow status...${NC}"
if check_workflow_status; then
  echo -e "${GREEN}All workflows are in Succeeded state${NC}"
  WORKFLOWS_STATUS=0
else
  echo -e "${RED}Some workflows are not in Succeeded state${NC}"
  WORKFLOWS_STATUS=1
fi

# Collect information about failed workflows
FAILED_WORKFLOWS=()
if [ "$FAILED" -gt 0 ]; then
  for workflow in "${WORKFLOWS[@]}"; do
    workflow_name=$(basename "$workflow" .yaml)
    if [ -f "${TEMP_DIR}/${workflow_name}-apply.log" ] && grep -q "Error\|Failed" "${TEMP_DIR}/${workflow_name}-apply.log"; then
      FAILED_WORKFLOWS+=("$workflow_name (deployment error)")
    fi
    
    # Check for failed workflowruns associated with this workflow
    for run_file in "${TEMP_DIR}"/*-status.yaml "${TEMP_DIR}"/*-timeout.yaml; do
      if [ -f "$run_file" ]; then
        run_name=$(basename "$run_file" | sed 's/-status.yaml\|-timeout.yaml//')
        if kubectl get workflowrun "$run_name" -o jsonpath='{.metadata.labels.workflows\.external-secrets\.io/workflow-name}' 2>/dev/null | grep -q "$workflow_name"; then
          FAILED_WORKFLOWS+=("$workflow_name (run: $run_name)")
        fi
      fi
    done
  done
fi

# Print summary with statistics
echo -e "\n${GREEN}=== WORKFLOW DEPLOYMENT SUMMARY ===${NC}"
printf "┌───────────────────────────┬─────────┐\n"
printf "│ %-25s │ %-7s │\n" "METRIC" "VALUE"
printf "├───────────────────────────┼─────────┤\n"
printf "│ %-25s │ %-7d │\n" "Total workflows" "${#WORKFLOWS[@]}"
printf "│ %-25s │ %-7d │\n" "Successful deployments" "$((${#WORKFLOWS[@]} - FAILED))"
printf "│ %-25s │ %-7d │\n" "Failed deployments" "$FAILED"
printf "│ %-25s │ %-7s │\n" "Workflows status check" "$([ $WORKFLOWS_STATUS -eq 0 ] && echo "${GREEN}PASSED${NC}" || echo "${RED}FAILED${NC}")"
printf "└───────────────────────────┴─────────┘\n"
echo -e "Logs and details are available in ${TEMP_DIR}"

# Print details of failed workflows if any
if [ "${#FAILED_WORKFLOWS[@]}" -gt 0 ]; then
  echo -e "\n${RED}=== FAILED WORKFLOWS DETAILS ===${NC}"
  printf "┌─────────────────────────────────────────────────────────────────┐\n"
  printf "│ %-65s │\n" "FAILED WORKFLOW"
  printf "├─────────────────────────────────────────────────────────────────┤\n"
  for failed in "${FAILED_WORKFLOWS[@]}"; do
    printf "│ %-65s │\n" "$failed"
  done
  printf "└─────────────────────────────────────────────────────────────────┘\n"
  echo -e "${YELLOW}Check the logs in ${TEMP_DIR} for more details${NC}"
fi

# Display the final status table again for a complete overview
echo -e "\n${GREEN}=== FINAL WORKFLOW STATUS TABLE ===${NC}"
generate_workflow_status_table "no-check"

if [[ "$FAILED" -gt 0 || "$WORKFLOWS_STATUS" -ne 0 ]]; then
  echo -e "${RED}Some workflows failed or are not in Succeeded state. Please check the logs for details.${NC}"
  exit 1
else
  echo -e "${GREEN}All workflows completed successfully and are in Succeeded state!${NC}"
  exit 0
fi
