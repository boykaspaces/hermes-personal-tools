# Architecture

## Runtime components

```text
Hermes Gateway
  -> API Gateway custom authorizer -> Authorizer Lambda -> client-token Secret
  -> API Gateway /mcp              -> MCP Lambda        -> selected providers

Hermes host IAM role
  -> API Gateway AWS_IAM lease route
  -> Credential Lease Lambda
  -> provider registry -> GitHub App issuer -> private-key Secret
  -> short-lived lease file in host tmpfs
  -> read-only mount in coding container
  -> git-credential-hermes, invoked by Git
```

The MCP service, bearer Authorizer, and Credential Lease service are separate
Go modules and Lambda roles. The host/client module is not a Lambda and never
contains a provider private key.

## Extension boundaries

- MCP features: config + provider interface/adapters + tool module + explicit
  production catalog entry.
- Credential providers: provider-specific configuration and issuer registered
  behind the generic profile registry.
- Credential consumers: protocol adapter over the generic lease validation and
  lookup core.

One Lambda handler can route multiple server-defined credential profiles.
Adding a provider does not require a new HTTP handler, and callers cannot
supply provider names, raw targets, permissions, or lease lifetime.
