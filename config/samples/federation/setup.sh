#!/bin/bash

# This script sets up the Federation Generator example

set -e

echo "Note: Make sure you have installed the CRDs using 'make crds.install' from the root of the repository"
echo "      and deployed External Secrets Operator with your changes."
echo ""

echo "Creating federation server..."
kubectl apply -f federation.yaml

echo "Creating authorization..."
kubectl apply -f authorization.yaml

echo "Creating password generator..."
kubectl apply -f password-generator.yaml

echo "Creating service account token secret..."
kubectl apply -f token-secret.yaml

echo "Creating federation generator..."
kubectl apply -f federation-generator.yaml

echo "Creating external secret..."
kubectl apply -f external-secret.yaml

echo "Waiting for external secret to be ready..."
kubectl wait --for=condition=Ready externalsecret/federation-test --timeout=60s || echo "Warning: Timed out waiting for external secret to be ready"

echo "Verifying secret was created..."
kubectl get secret federation-test -o jsonpath='{.data}' | jq -r 'map_values(@base64d)' || echo "Warning: Secret not found or could not be decoded"

echo "Setup complete!"