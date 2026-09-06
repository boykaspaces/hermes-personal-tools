# Operator Policy Templates

| Template | Intended use |
|---|---|
| `deployer-policy.json.tmpl` | Temporary reviewed CloudFormation and artifact deployment access |
| `credential-operator-policy.json.tmpl` | Narrow out-of-band Secret write access without Secret readback |

Required substitutions:

- `${AWS_ACCOUNT_ID}` — 12-digit target AWS account ID.
- `${AWS_REGION}` — target AWS Region.

Render to a temporary path outside the repository, validate with `jq`, inspect
the final resources, then apply to the intended operator identity. Do not
commit rendered policies because they contain production identifiers.

These templates describe an upper bound for this deployment shape. Review AWS
service authorization behavior and narrow resources further when the target
stack name differs from the documented default.
