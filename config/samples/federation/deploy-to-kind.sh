#!/bin/bash

# This script deploys External Secrets Operator to a kind cluster

set -e

echo "Deleting existing kind cluster (if any)..."
kind delete cluster --name external-secrets || true

echo "Creating new kind cluster..."
kind create cluster --name external-secrets

echo "Building External Secrets Operator..."
export TAG=$(make docker.tag)
export IMAGE=$(make docker.imagename)
make docker.build

echo "Waiting for kind cluster to be ready..."
# Wait for the cluster to be ready by checking for nodes
while ! kubectl get nodes | grep -q "Ready"; do
  echo "Waiting for nodes to be ready..."
  sleep 5
done

# Wait for core components to be ready
echo "Waiting for core components to be ready..."
kubectl wait --for=condition=Ready --namespace=kube-system pods --all --timeout=120s

echo "Loading image into kind cluster..."
kind load docker-image $IMAGE:$TAG --name external-secrets

echo "Deploying External Secrets Operator..."
make helm.generate

helm upgrade --install external-secrets ./deploy/charts/external-secrets/ \
  --namespace external-secrets --create-namespace \
  --set image.repository=$IMAGE --set image.tag=$TAG \
  --set webhook.image.repository=$IMAGE --set webhook.image.tag=$TAG \
  --set certController.image.repository=$IMAGE --set certController.image.tag=$TAG \
  --set federation.service.enabled=true \
  --set federation.service.port=8000

echo "Waiting for External Secrets Operator to be ready..."
kubectl wait --for=condition=Available deployment/external-secrets -n external-secrets --timeout=60s

echo "Waiting for webhook service to be ready..."
if ! kubectl wait --for=condition=Available deployment/external-secrets-webhook -n external-secrets --timeout=60s; then
  echo "Webhook deployment not ready. Checking status..."
  kubectl describe deployment external-secrets-webhook -n external-secrets
  kubectl get pods -n external-secrets
  kubectl logs -l app.kubernetes.io/component=webhook -n external-secrets
fi

echo "Verifying webhook service..."
kubectl get service external-secrets-webhook -n external-secrets

echo "External Secrets Operator deployed successfully!"
echo ""
echo "IMPORTANT: Before running the setup script, make sure the webhook service is ready."
echo "You can check the status with: kubectl get pods -n external-secrets"
echo ""
echo "If the webhook service is ready, you can run ./setup.sh to deploy the federation example."
