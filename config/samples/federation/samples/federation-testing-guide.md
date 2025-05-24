# Federation Server Testing Guide

## Quick Setup

1. **Enable Federation in ESO**
```bash
helm upgrade external-secrets external-secrets/external-secrets \
  --set federation.service.enabled=true \
  --set federation.listen.port=8000
```

2. **Deploy Sample Resources**
```bash
kubectl apply -f samples/
```

3. **Get Service Account Token**
```bash
SA_TOKEN=$(kubectl create token test-federation-sa -n test-federation --duration=1h)
CA_CERT=$(kubectl get configmap kube-root-ca.crt -n kube-system -o jsonpath='{.data.ca\.crt}' | base64 -w 0)
```

4. **Port Forward Federation Server**
```bash
kubectl port-forward -n external-secrets svc/external-secrets-federation-server 8000:8000
```

## Test Endpoints

**Get Secret:**
```bash
curl -X POST http://localhost:8000/secretstore/test-cluster-store/secrets/test-secret \
  -H "Authorization: Bearer $SA_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"ca.crt\": \"$CA_CERT\"}"
```

**Generate Password:**
```bash
curl -X POST http://localhost:8000/generators/test-federation/PasswordGenerator/test-password-gen \
  -H "Authorization: Bearer $SA_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"ca.crt\": \"$CA_CERT\"}"
```

## Sample Files

- `kubernetes-federation.yaml` - Federation provider config
- `authorization.yaml` - Access permissions
- `cluster-secret-store.yaml` - Test secret store
- `test-setup.yaml` - Test namespace and service account