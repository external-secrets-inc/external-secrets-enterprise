#!/bin/bash
# Script to create a secret with the federation-sa service account token and required metadata

set -e

# Set up proxy to Kubernetes API
echo "Setting up Kubernetes API proxy..."
kubectl proxy --port=8001 &
K8S_PROXY_PID=$!
sleep 2  # Give proxy time to start

# Create the token
echo "Creating service account token..."
TOKEN=$(kubectl create token federation-sa -n federation-example)

# Get the CA certificate
echo "Getting CA certificate..."
CA_CERT=$(kubectl config view --raw -o jsonpath='{.clusters[?(@.name=="kind-external-secrets")].cluster.certificate-authority-data}')

# Get the JWKS
echo "Getting JWKS from Kubernetes API server..."
JWKS=$(curl -s http://localhost:8001/openid/v1/jwks)

# Create a JSON file with all required information
echo "Creating JSON payload..."
cat > federation-metadata.json <<EOF
{
  "ca.crt": "$CA_CERT",
  "jwks": $JWKS,
  "issuer": "https://kubernetes.default.svc.cluster.local",
  "oidc_discovery_url": "https://kubernetes.default.svc.cluster.local/.well-known/openid-configuration",
  "jwks_uri": "https://kubernetes.default.svc.cluster.local/openid/v1/jwks"
}
EOF

# Create the secret with token and metadata
echo "Creating secret with token and metadata..."
kubectl create secret generic federation-token -n federation-example \
  --from-literal=token=$TOKEN \
  --from-file=metadata=federation-metadata.json

# Clean up
rm federation-metadata.json
kill $K8S_PROXY_PID || true

echo "Secret 'federation-token' created in namespace 'federation-example' with token and metadata"
