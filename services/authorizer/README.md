# Token Authorizer Service

This service has one responsibility: authenticate the Hermes client token and
return a trusted API Gateway authorizer context.

It can read only the Hermes client-token Secret. It does not contain MCP,
Google Workspace, provider or tool-execution code.

Input header:

```text
X-Hermes-Gateway-Token: Bearer <token>
```

Successful context:

```json
{"caller_id":"hermes"}
```

The token grants access to this personal MCP service as a whole. The
Authorizer does not issue application scopes or decide which MCP tools run.
