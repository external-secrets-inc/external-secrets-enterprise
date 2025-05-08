#!/bin/bash
# Test script for the Workflow API

# Configuration
API_SERVER="localhost:8080"
NAMESPACE="default"
TEMPLATE_NAME="rotate-database-credentials"

# Create a WorkflowTemplate
echo "Creating WorkflowTemplate..."
kubectl apply -f tests/workflowtemplate_test.yaml

# Wait for the template to be created
sleep 2

# Test API health check
echo "Testing API health check..."
curl -s http://${API_SERVER}/healthz

# Test creating a WorkflowRun via API
echo -e "\n\nTesting WorkflowRun creation via API..."
curl -s -X POST \
  "http://${API_SERVER}/api/v1/namespaces/${NAMESPACE}/workflowruns" \
  -H "Content-Type: application/json" \
  -d '{
    "templateName": "'"${TEMPLATE_NAME}"'",
    "parameters": {
      "databaseName": "test-db",
      "notificationChannel": "#test-alerts"
    }
  }'

# List WorkflowRuns
echo -e "\n\nListing WorkflowRuns..."
curl -s "http://${API_SERVER}/api/v1/namespaces/${NAMESPACE}/workflowruns"

# Get the WorkflowRun name
WORKFLOWRUN_NAME=$(kubectl get workflowrun -n ${NAMESPACE} -l workflows.external-secrets.io/api=true -o jsonpath='{.items[0].metadata.name}')

if [ -n "$WORKFLOWRUN_NAME" ]; then
  # Get a specific WorkflowRun
  echo -e "\n\nGetting WorkflowRun ${WORKFLOWRUN_NAME}..."
  curl -s "http://${API_SERVER}/api/v1/namespaces/${NAMESPACE}/workflowruns/${WORKFLOWRUN_NAME}"
  
  # Check the status of the WorkflowRun
  echo -e "\n\nChecking WorkflowRun status..."
  kubectl get workflowrun ${WORKFLOWRUN_NAME} -n ${NAMESPACE} -o yaml
else
  echo -e "\n\nNo WorkflowRun found"
fi

echo -e "\n\nTest completed"