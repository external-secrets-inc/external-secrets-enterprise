#!/bin/bash
# Script to connect to federation server with Kubernetes service account token

set -e  # Exit immediately if a command exits with a non-zero status
set -x  # Print commands and their arguments as they are executed

# Step 1: Set up proxies to both services
echo "Setting up Kubernetes API proxy..."
kubectl proxy --port=8001 &
K8S_PROXY_PID=$!
sleep 2  # Give proxy time to start

echo "Setting up federation server proxy..."
kubectl port-forward -n external-secrets svc/external-secrets-federation-server 8000:8000 &
FEDERATION_PROXY_PID=$!
sleep 2  # Give proxy time to start

# Step 2: Get the Kubernetes service account token
echo "Getting service account token..."
TOKEN=$(kubectl create token federation-sa -n federation-example)

# Step 3: Get the CA certificate from Kubernetes config
echo "Getting CA certificate..."
CA_CERT=$(kubectl config view --raw -o jsonpath='{.clusters[?(@.name=="kind-external-secrets")].cluster.certificate-authority-data}')

# Step 4: Get the JWKS directly from Kubernetes API
echo "Getting JWKS from Kubernetes API server..."
JWKS=$(curl -s http://localhost:8001/openid/v1/jwks)
echo "JWKS obtained:"
echo "$JWKS" | jq .

# Step 5: Create a comprehensive JSON payload
echo "Creating JSON payload..."
JSON_PAYLOAD=$(jq -n \
  --arg cert "$CA_CERT" \
  '{
    "ca.crt": $cert
  }')

# Save payload to file for inspection
echo "$JSON_PAYLOAD" > payload.json
echo "Payload saved to payload.json"

# Step 6: Make the request to the federation server via proxy
echo "Sending request to federation server..."
curl -X POST \
  http://localhost:8000/generators/federation-example/Fake/federation-generator \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "@payload.json"

# Step 7: Clean up
rm ./payload.json || true
echo "Cleaning up proxies..."
kill $K8S_PROXY_PID || true
kill $FEDERATION_PROXY_PID || true
echo "Script execution completed"
