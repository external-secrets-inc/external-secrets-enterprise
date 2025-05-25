# Federation Generator Example

This directory contains examples of how to use the Federation generator with a local federation server.

## Prerequisites

- A Kubernetes cluster with External Secrets Operator installed
- The federation server enabled in the External Secrets Operator

## Setup

1. Deploy the External Secrets Operator with the federation server enabled:

```bash
./deploy-to-kind.sh
```

2. Set up the federation example:

```bash
# Create the namespace and service account
kubectl create namespace federation-example
kubectl create serviceaccount federation-sa -n federation-example

# Create the Fake generator
kubectl apply -f source-generator/01-namespace.yaml
kubectl apply -f source-generator/02-rbac.yaml
kubectl apply -f source-generator/03-federation.yaml
kubectl apply -f source-generator/04-authorization.yaml
kubectl apply -f source-generator/05-fake-generator.yaml
```

## Testing the Federation Server

You can test that the federation server is working correctly by running:

```bash
./test.sh
```

This script will:
1. Set up proxies to the Kubernetes API and federation server
2. Get a service account token for the federation-sa service account
3. Get the CA certificate from the Kubernetes config
4. Get the JWKS from the Kubernetes API server
5. Create a JSON payload with the necessary information
6. Send a request to the federation server to get the secret from the Fake generator

## Using the Federation Generator

The Federation generator allows you to access generators in the federation server from within your cluster.

1. Create a token secret for the federation-sa service account:

```bash
./create-federation-token.sh
```

2. Apply the Federation generator configuration:

```bash
kubectl apply -f federation-generator.yaml
```

3. Apply the ExternalSecret that uses the Federation generator:

```bash
kubectl apply -f external-secret.yaml
```

4. Test the Federation generator:

```bash
./test-federation-generator.sh
```

## Files

- `deploy-to-kind.sh`: Script to deploy External Secrets Operator with federation server enabled
- `test.sh`: Script to test the federation server directly
- `create-federation-token.sh`: Script to create a token secret for the federation-sa service account
- `federation-generator.yaml`: Federation generator configuration
- `external-secret.yaml`: ExternalSecret that uses the Federation generator
- `test-federation-generator.sh`: Script to test the Federation generator
- `source-secret/`: Example files for accessing secrets through the federation server
- `source-generator/`: Example files for accessing generators through the federation server
