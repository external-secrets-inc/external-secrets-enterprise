# Okta Federation Server Commands

## Overview

This guide walks you through setting up Okta OAuth2 authentication for the External Secrets Federation Server, from creating an Okta application to testing the integration.

## Step 1: Set Up Okta Application

### 1.1 Create an Okta Account (if needed)

If you don't have an Okta account:
1. Go to https://developer.okta.com/signup/
2. Sign up for a free developer account
3. You'll receive an Okta domain like `https://dev-12345.okta.com` or `https://trial-1038013.okta.com`

### 1.2 Create an OAuth2 Application

1. Log in to your Okta Admin Console (e.g., `https://trial-1038013-admin.okta.com`)
2. Navigate to **Applications** → **Applications** in the left sidebar
3. Click **Create App Integration**
4. Select **API Services** (this is for machine-to-machine authentication)
5. Click **Next**
6. Configure the application:
   - **App integration name**: `External Secrets Federation` (or any descriptive name)
7. Click **Save**
8. You'll be redirected to the application's settings page
9. **Copy the Client ID** - you'll need this (e.g., `0oawl3l22qkmQK274697`)

### 1.3 Generate RSA Key Pair

**Option 1: Use the helper script (Recommended)**

```bash
cd scripts
chmod +x generate-okta-keys.sh
./generate-okta-keys.sh
```

This will generate `private_key.pem` and `public_key.pem` and display instructions.

**Option 2: Manual generation**

```bash
# Generate private key
openssl genrsa -out private_key.pem 2048

# Extract public key
openssl rsa -in private_key.pem -pubout -out public_key.pem

# Verify the key
openssl rsa -in private_key.pem -check -noout
```

### 1.4 Register Public Key in Okta

1. In your Okta application settings, scroll down to **CLIENT CREDENTIALS**
2. Under **Client authentication**, select **Public key / Private key**
3. Click **Add key** (or **Manage keys** if keys already exist)
4. Select **Add** → **PEM** (or **JWK** if you have the JWK format)
5. Paste your **public key** content:
   ```
   -----BEGIN PUBLIC KEY-----
   MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA8p7P8vn3qk...
   -----END PUBLIC KEY-----
   ```
6. Click **Save**
7. **Note the Key ID (kid)** - Okta will generate this automatically

**Alternative: Register as JWK**

If using JWK format, your public key should look like:
```json
{
  "kty": "RSA",
  "e": "AQAB",
  "use": "sig",
  "kid": "my-key-id",
  "alg": "RS256",
  "n": "xGOr-H7A-PWiUJb3..."
}
```

### 1.5 Grant API Scopes (Optional)

If you plan to access Okta Management APIs (e.g., for `CheckIdentityExists` functionality):

1. In the application settings, go to **Okta API Scopes** tab
2. Click **Grant** for the scopes you need:
   - `okta.apps.read` - Read application information
   - `okta.users.read` - Read user information (if needed)
3. Click **Save**

**Note**: For basic federation authentication (just getting an access token to authenticate to your federation server), you may not need any Okta API scopes. The federation server validates the token using JWKS, not Okta Management APIs.

### 1.6 Note Your Configuration

Save these values - you'll need them:

```bash
OKTA_DOMAIN="https://trial-1038013.okta.com"
OKTA_CLIENT_ID="0oawl3l22qkmQK274697"
OKTA_AUTH_SERVER_ID=""  # Empty for org authorization server, or "default", or custom ID
PRIVATE_KEY_PATH="./private_key.pem"
```

**Determining the Issuer URL:**

- **Org Authorization Server** (no scopes or Okta Management API scopes):
  - Issuer: `https://trial-1038013.okta.com`
  - Token endpoint: `https://trial-1038013.okta.com/oauth2/v1/token`
  - JWKS endpoint: `https://trial-1038013.okta.com/oauth2/v1/keys`

- **Default Authorization Server** (custom scopes):
  - Issuer: `https://trial-1038013.okta.com/oauth2/default`
  - Token endpoint: `https://trial-1038013.okta.com/oauth2/default/v1/token`
  - JWKS endpoint: `https://trial-1038013.okta.com/oauth2/default/v1/keys`

- **Custom Authorization Server**:
  - Issuer: `https://trial-1038013.okta.com/oauth2/{customId}`
  - Token endpoint: `https://trial-1038013.okta.com/oauth2/{customId}/v1/token`

## Step 2: Deploy Kubernetes Resources

### 2.1 Create OktaFederation Resource

```yaml
apiVersion: federation.external-secrets.io/v1alpha1
kind: OktaFederation
metadata:
  name: okta-prod
spec:
  domain: https://trial-1038013.okta.com
  authorizationServerID: ""  # Empty for org auth server, or "default", or your custom ID
```

Apply it:
```bash
kubectl apply -f okta-federation.yaml
```

### 2.2 Create Authorization Resource

This maps your Okta client to allowed ClusterSecretStores:

```yaml
apiVersion: federation.external-secrets.io/v1alpha1
kind: Authorization
metadata:
  name: okta-client-authorization
spec:
  federationRef:
    kind: OktaFederation
    name: okta-prod
  
  subject:
    oidc:
      # IMPORTANT: Issuer must match your Okta authorization server
      # For org auth server: https://trial-1038013.okta.com
      # For default auth server: https://trial-1038013.okta.com/oauth2/default
      issuer: "https://trial-1038013.okta.com"
      # Subject must match your Okta client ID
      subject: "0oawl3l22qkmQK274697"
  
  # Which ClusterSecretStores this client can access
  allowedClusterSecretStores:
    - "vault-backend"
  
  allowedGenerators: []
  allowedGeneratorStates: []
```

Apply it:
```bash
kubectl apply -f okta-authorization.yaml
```

## Step 3: Test Getting an Okta Access Token

### 3.1 Using the Helper Script (Recommended)

```bash
cd scripts
chmod +x get-okta-token.sh

# Set required environment variables
export OKTA_CLIENT_ID="0oawl3l22qkmQK274697"
export OKTA_DOMAIN="https://trial-1038013.okta.com"
export OKTA_PRIVATE_KEY_PATH="../private_key.pem"  # Path to your private key
export OKTA_AUTH_SERVER_ID=""  # Empty for org server
export OKTA_SCOPES=""  # Empty for federation auth (most common)

# Get the token
./get-okta-token.sh
```

The script will:
- Generate a signed JWT client assertion
- Exchange it for an Okta access token
- Display the token and expiry time
- Provide an export command for easy use

To capture just the token for scripts:
```bash
ACCESS_TOKEN=$(./get-okta-token.sh 2>/dev/null | tail -n1)
export ACCESS_TOKEN
```

### 3.2 Manual Token Request (Advanced)

If you prefer to understand the process or need to customize it:

```bash
#!/bin/bash
CLIENT_ID="0oawl3l22qkmQK274697"
PRIVATE_KEY_FILE="./private_key.pem"
OKTA_DOMAIN="https://trial-1038013.okta.com"
TOKEN_ENDPOINT="$OKTA_DOMAIN/oauth2/v1/token"

# Generate JWT for client assertion
CURRENT_TIME=$(date +%s)
EXPIRY_TIME=$((CURRENT_TIME + 300))
JTI=$(cat /dev/urandom | LC_ALL=C tr -dc 'a-f0-9' | fold -w 32 | head -n 1)

# Create JWT components
HEADER='{"alg":"RS256","typ":"JWT"}'
PAYLOAD="{\"iss\":\"$CLIENT_ID\",\"sub\":\"$CLIENT_ID\",\"aud\":\"$TOKEN_ENDPOINT\",\"exp\":$EXPIRY_TIME,\"iat\":$CURRENT_TIME,\"jti\":\"$JTI\"}"

# Base64URL encode
HEADER_B64=$(echo -n "$HEADER" | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')
PAYLOAD_B64=$(echo -n "$PAYLOAD" | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')

# Sign with private key
SIGNATURE=$(echo -n "${HEADER_B64}.${PAYLOAD_B64}" | openssl dgst -sha256 -sign "$PRIVATE_KEY_FILE" -binary | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')

# Construct final JWT
CLIENT_ASSERTION="${HEADER_B64}.${PAYLOAD_B64}.${SIGNATURE}"

# Request access token
curl -X POST "$TOKEN_ENDPOINT" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=$CLIENT_ID" \
  -d "client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer" \
  -d "client_assertion=$CLIENT_ASSERTION" \
  | jq -r '.access_token'
```

## Step 4: Call Federation Server

### Fetch a Secret from ClusterSecretStore

```bash
curl -X POST http://localhost:8080/secretstore/vault-backend/secrets/my-secret \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json"
```

**Parameters:**
- `vault-backend`: Name of the ClusterSecretStore (must be in `allowedClusterSecretStores`)
- `my-secret`: The secret key to fetch from the backend

**Expected Response (200 OK):**
```json
"base64-encoded-secret-value"
```

**Error Responses:**
- **401 Unauthorized**: Token invalid or expired
- **404 Not Found**: ClusterSecretStore not in allowed list, or secret not found
- **400 Bad Request**: Invalid request or secret store error

## Step 5: Verify AuthorizedIdentity Created

After a successful call, check that an AuthorizedIdentity was created:

```bash
kubectl get authorizedidentities
```

You should see an identity named like `okta-0oawl3l22qkmqk274697` with:
- `spec.subject.oidc.issuer`: Your Okta issuer
- `spec.subject.oidc.subject`: Your client ID
- `spec.issuedCredentials`: List of secrets/generators accessed

Example:
```bash
kubectl get authorizedidentity okta-0oawl3l22qkmqk274697 -o yaml
```

## Troubleshooting

### Okta Application Setup Issues

#### Error: "Invalid client credentials"
- **Cause**: Client ID is incorrect or the application doesn't exist
- **Solution**: Double-check the Client ID in your Okta application settings

#### Error: "Invalid JWT signature"
- **Cause**: The private key doesn't match the public key registered in Okta
- **Solution**: 
  1. Regenerate the key pair
  2. Re-register the public key in Okta
  3. Ensure you're using the matching private key

#### Error: "Key with kid 'xxx' not found"
- **Cause**: The JWT's `kid` (Key ID) doesn't match any registered public key
- **Solution**: 
  1. Check which `kid` Okta assigned to your public key
  2. Either use the Okta-generated `kid` or don't specify a `kid` in your JWT header

#### Error: "consent_required" or "You are not allowed any of the requested scopes"
- **Cause**: Requesting scopes that haven't been granted in Okta
- **Solution**: 
  1. For federation authentication (most common), don't request any scopes (omit the `scope` parameter)
  2. If you need Okta API access, grant the required scopes in Okta API Scopes tab
  3. Use `--okta-scopes=""` to explicitly request no scopes

### Token Generation Issues

#### JWT Creation Fails
**Verify your private key format:**
```bash
openssl rsa -in private_key.pem -check -noout
```

**Test JWT generation:**
```bash
# Create a simple test JWT
HEADER='{"alg":"RS256","typ":"JWT"}'
PAYLOAD='{"test":"value"}'
HEADER_B64=$(echo -n "$HEADER" | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')
PAYLOAD_B64=$(echo -n "$PAYLOAD" | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')
SIGNATURE=$(echo -n "${HEADER_B64}.${PAYLOAD_B64}" | openssl dgst -sha256 -sign private_key.pem -binary | openssl base64 -e -A | tr '+/' '-_' | tr -d '=')
echo "${HEADER_B64}.${PAYLOAD_B64}.${SIGNATURE}"
```

#### Token Endpoint Returns 400/401
**Check token request:**
```bash
# Enable verbose curl output to see the full request/response
curl -v -X POST "$OKTA_DOMAIN/oauth2/v1/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=$CLIENT_ID" \
  -d "client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer" \
  -d "client_assertion=$CLIENT_ASSERTION"
```

**Decode your client assertion JWT:**
```bash
# Copy your CLIENT_ASSERTION and decode at https://jwt.io
echo "Paste this at jwt.io: $CLIENT_ASSERTION"
```

Verify:
- `iss` and `sub` both equal your Client ID
- `aud` matches your token endpoint URL exactly
- `exp` is in the future (Unix timestamp)
- `iat` is the current time (Unix timestamp)

### Federation Server Issues

#### "no authorization configured for issuer"
- **Cause**: No Authorization resource matches the token's issuer
- **Solution**: 
  1. Check the `iss` claim in your token (decode at jwt.io)
  2. Ensure your Authorization resource's `subject.oidc.issuer` matches exactly
  3. For org auth server, use just the domain: `https://trial-1038013.okta.com`
  4. For default auth server, append `/oauth2/default`

#### "failed to get JWKS"
- **Cause**: Federation server can't reach Okta to fetch public keys
- **Solution**: 
  1. Check network connectivity from the cluster to Okta
  2. Verify the OktaFederation resource has the correct domain
  3. Check if the correct JWKS endpoint is being called (logs will show the URL)

#### "token validation failed"
- **Cause**: Token signature verification failed
- **Solution**:
  1. Ensure the public key is correctly registered in Okta
  2. Verify you're using the correct private key
  3. Check that the token hasn't expired
  4. Confirm the issuer in the token matches the Authorization Server URL

#### "Not Found" when accessing secrets
- **Cause**: ClusterSecretStore not in the Authorization's `allowedClusterSecretStores`
- **Solution**: Add the ClusterSecretStore name to the list:
  ```yaml
  allowedClusterSecretStores:
    - "vault-backend"  # Add your store here
  ```

### Verify Okta Token Claims

Decode the JWT access token at https://jwt.io to verify:
- `iss`: Matches your Authorization's `subject.oidc.issuer`
- `sub`: Matches your Authorization's `subject.oidc.subject` (your Client ID)
- `exp`: Token not expired (Unix timestamp in the future)
- `iat`: Token issue time
- `cid`: Client ID (Okta-specific claim)

### Check Federation Server Logs

```bash
kubectl logs -n external-secrets-system deployment/external-secrets -c federation-server
```

### Verify Okta JWKS Endpoint

Test that the JWKS endpoint is accessible:
```bash
# For org authorization server
curl https://trial-1038013.okta.com/oauth2/v1/keys | jq

# For default authorization server  
curl https://trial-1038013.okta.com/oauth2/default/v1/keys | jq
```

You should see a list of public keys with `kid`, `n` (modulus), and `e` (exponent) values.

## Quick Reference

### Key Files
- `private_key.pem` - Your RSA private key (keep secure!)
- `public_key.pem` - Your RSA public key (register in Okta)

### Environment Variables
```bash
export OKTA_DOMAIN="https://trial-1038013.okta.com"
export OKTA_CLIENT_ID="0oawl3l22qkmQK274697"
export OKTA_AUTH_SERVER_ID=""  # Empty for org server
```

### Important URLs
- **Okta Admin Console**: `https://{your-domain}-admin.okta.com`
- **Token Endpoint (org)**: `https://{your-domain}/oauth2/v1/token`
- **JWKS Endpoint (org)**: `https://{your-domain}/oauth2/v1/keys`
- **Token Endpoint (default)**: `https://{your-domain}/oauth2/default/v1/token`
- **JWKS Endpoint (default)**: `https://{your-domain}/oauth2/default/v1/keys`
