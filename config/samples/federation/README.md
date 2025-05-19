# Federation Generator

This directory contains examples and scripts for using the Federation Generator with External Secrets Operator.

## Overview

The Federation Generator allows pods to request credentials from a federation server. It acts as a client to the federation server's generator endpoint, enabling the external-secrets CLI to leverage the federation server's generator capabilities without any modifications to the CLI itself.

## Prerequisites

- A Kubernetes cluster with External Secrets Operator installed
- A federation server running in the cluster

## Files in this Directory

- `README.md` - This file
- `authorization.yaml` - Example Authorization resource
- `federation.yaml` - Example KubernetesFederation resource
- `password-generator.yaml` - Example Password Generator resource
- `federation-generator.yaml` - Example Federation Generator resource
- `external-secret.yaml` - Example ExternalSecret using the Federation Generator
- `token-secret.yaml` - Example Secret containing the auth token
- `setup.sh` - Script to set up the example resources

## How It Works

1. The Federation Generator connects to the federation server
2. It authenticates using the provided token
3. It requests credentials from the specified generator (e.g., Password Generator)
4. The federation server verifies the authorization
5. The federation server generates the credentials using the specified generator
6. The Federation Generator returns the credentials to the ExternalSecret controller
7. The ExternalSecret controller creates a Secret with the credentials

## Installation

Before using the Federation Generator, you need to build and deploy External Secrets Operator with your changes to install the Federation generator CRD.

1. Build and deploy External Secrets Operator with your changes:
   ```bash
   # From the root of the external-secrets-enterprise repository
   make docker.build
   make docker.push
   # Deploy to your cluster
   ```

2. Install the CRDs:
   ```bash
   # From the root of the external-secrets-enterprise repository
   make crds.install
   ```

## Usage

1. Create a federation server:
   ```bash
   kubectl apply -f federation.yaml
   ```

2. Create an authorization:
   ```bash
   kubectl apply -f authorization.yaml
   ```

3. Create a password generator:
   ```bash
   kubectl apply -f password-generator.yaml
   ```

4. Create a service account token secret:
   ```bash
   kubectl apply -f token-secret.yaml
   ```

5. Create a federation generator:
   ```bash
   kubectl apply -f federation-generator.yaml
   ```

6. Create an external secret:
   ```bash
   kubectl apply -f external-secret.yaml
   ```

7. Verify the secret was created:
   ```bash
   kubectl get secret federation-test -o jsonpath='{.data}' | jq -r 'map_values(@base64d)'
   ```

Alternatively, you can run the setup script after installing the CRDs:
```bash
./setup.sh
```

## Troubleshooting

If you encounter issues, check the logs of the External Secrets Operator:
```bash
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets
```

Check the status of the External Secret:
```bash
kubectl describe externalsecret federation-test