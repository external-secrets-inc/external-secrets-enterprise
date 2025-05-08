#!/bin/bash
# This script demonstrates how to trigger a workflow using the External Secrets API

# Configuration
API_SERVER="localhost:8080"
NAMESPACE="default"
TEMPLATE_NAME="rotate-database-credentials"
DATABASE_NAME="production-postgres"
NOTIFICATION_CHANNEL="#prod-alerts"

# Create a temporary JSON file for the request
cat > /tmp/workflow-request.json << EOF
{
  "templateName": "${TEMPLATE_NAME}",
  "arguments": {
    "databaseName": "${DATABASE_NAME}",
    "notificationChannel": "${NOTIFICATION_CHANNEL}"
  }
}
EOF

# Send the request to the API server
echo "Triggering workflow for database: ${DATABASE_NAME}"
curl -X POST \
  "http://${API_SERVER}/api/v1/namespaces/${NAMESPACE}/workflowruns" \
  -H "Content-Type: application/json" \
  -d @/tmp/workflow-request.json

# Clean up
rm /tmp/workflow-request.json

echo -e "\nCheck the status of the workflow with:"
echo "kubectl get workflowrun -n ${NAMESPACE}"