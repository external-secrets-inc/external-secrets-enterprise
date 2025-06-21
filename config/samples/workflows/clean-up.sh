#!/bin/bash
# Script to delete everything related to workflows from the cluster

# Default Constants
DELETION_TIMEOUT=60
DELETION_POLL_INTERVAL=2

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting workflow deletion script...${NC}"

# Check if kubectl is installed
if ! command -v kubectl >/dev/null 2>&1; then
  echo -e "${RED}Error: kubectl is not installed. Please install kubectl and try again.${NC}"
  exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &>/dev/null; then
  echo -e "${RED}Error: Cannot connect to Kubernetes cluster. Please ensure your cluster is running.${NC}"
  exit 1
fi

# Function to delete Kubernetes resources of a specific type
delete_k8s_resources() {
  local resource_type=$1
  local resource_plural=$2

  echo -e "${YELLOW}Checking for existing ${resource_plural}...${NC}"
  local resources
  resources=$(kubectl get "${resource_type}" -o name 2>/dev/null || echo "")

  if [[ -n "$resources" ]]; then
    echo -e "${YELLOW}Found existing ${resource_plural}. Deleting...${NC}"
    if ! kubectl delete "${resource_type}" --all --timeout="${DELETION_TIMEOUT}s" 2>&1; then
      echo -e "${RED}Failed to delete ${resource_plural}${NC}"
      return 1
    fi
    echo -e "${GREEN}All ${resource_plural} deleted${NC}"

    wait_for_resource_deletion "$resource_type" "$resource_plural"
    return $?
  else
    echo -e "${GREEN}No existing ${resource_plural} found${NC}"
    return 0
  fi
}

# Function to wait for complete deletion of Kubernetes resources
wait_for_resource_deletion() {
  local resource_type=$1
  local resource_plural=$2

  echo -e "${YELLOW}Waiting for ${resource_plural} to be fully deleted...${NC}"
  local start_time
  start_time=$(date +%s)

  while true; do
    local remaining
    remaining=$(kubectl get "${resource_type}" --no-headers 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$remaining" -eq 0 ]]; then
      echo -e "${GREEN}All ${resource_plural} have been fully deleted${NC}"
      return 0
    fi

    echo -e "${YELLOW}Waiting for $remaining ${resource_plural} to be deleted...${NC}"

    local current_time
    current_time=$(date +%s)
    if (( current_time - start_time > DELETION_TIMEOUT )); then
      echo -e "${RED}Timeout waiting for ${resource_plural} to be deleted${NC}"
      return 1
    fi

    sleep $DELETION_POLL_INTERVAL
  done
}

# Delete all workflow-related resources
echo -e "${YELLOW}Deleting all workflow-related resources from the cluster...${NC}"

# Delete workflowruns first (they depend on workflows)
if ! delete_k8s_resources "workflowruns" "workflowruns"; then
  echo -e "${RED}Failed to delete all workflowruns${NC}"
  exit 1
fi

# Then delete workflows (they may depend on workflowtemplates)
if ! delete_k8s_resources "workflows" "workflows"; then
  echo -e "${RED}Failed to delete all workflows${NC}"
  exit 1
fi

# Finally delete workflowtemplates
if ! delete_k8s_resources "workflowtemplates" "workflowtemplates"; then
  echo -e "${RED}Failed to delete all workflowtemplates${NC}"
  exit 1
fi

echo -e "${GREEN}All workflow-related resources have been successfully deleted from the cluster!${NC}"
exit 0
