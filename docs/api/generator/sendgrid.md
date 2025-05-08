The SendGrid generator automates the creation and rotation of SendGrid API tokens within the External Secrets Operator (ESO) framework. By integrating directly with the SendGrid API, it generates new API keys with specified scopes and securely injects them into Kubernetes secrets. The generator also handles the cleanup of old API keys to maintain security best practices.

## Output Keys and Values

| Key    | Description           |
| ------ | --------------------- |
| apiKey | the generated API Key |

## Authentication

Use `spec.auth.secretRef.apiKeySecretRef` to point to an initial SendGrid API Key, which **must have the necessary scopes to manage additional API keys**. Refer to the [SendGrid API Documentation](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/api-keys) for more details and guides on how to create a new API key.

## Parameters

| Key           | Default | Description                                                                                                                                                                                                        |
| ------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| scopes        | `[]`      | A list of available scopes for the new API Key. You can see all available scopes on [SendGrid API Documentation](https://www.twilio.com/docs/sendgrid/api-reference/how-to-use-the-sendgrid-v3-api/authorization). |
| dataResidency | `global`  | `global` or `eu`. Used for when you are using a regional subuser.                                                                                                                                                  |

!!! warning "Default scopes"
    If the scopes are not specified, the generator will automatically create an API key with full access.

## Example Manifest

```yaml
{% include 'generator-sendgrid.yaml' %}
```

Example `ExternalSecret` that references the SendGrid generator:

```yaml
{% include 'generator-sendgrid-example.yaml' %}
```
