# AWS Deployment

This directory deploys the Personal Tools API, bearer Authorizer, optional
Credential Lease service, purpose-specific Secrets, least-privilege Lambda
roles, logs, and a private versioned artifact bucket.

## Files

| Path | Purpose |
|---|---|
| `artifacts-cloudformation.yaml` | Private versioned S3 bucket for immutable build archives |
| `cloudformation.yaml` | API Gateway, Lambda, Secret, log, role, condition, and output definitions |
| `policies/` | Operator policy templates requiring explicit substitution and review |
| `validate-template.sh` | Local syntax and security-invariant validation; optional AWS validation |

## Build and validate

From the repository root:

```sh
make check
make build
./deploy/aws/validate-template.sh
```

Use `./deploy/aws/validate-template.sh --aws "$AWS_REGION"` only with an AWS
identity authorized to call `cloudformation:ValidateTemplate`.

## Artifact bucket

```sh
aws cloudformation deploy \
  --stack-name personal-tools-artifacts \
  --template-file deploy/aws/artifacts-cloudformation.yaml \
  --region "$AWS_REGION"
```

Read `ArtifactBucketName` from the stack output. Upload each ZIP under a unique
release version or content digest and record its S3 version ID and SHA-256 in
the private operator repository.

## Main stack

Keep deployment values in a private parameter file outside this repository.
At minimum provide immutable artifact keys, optional object version IDs, and
the fixed deployment targets required by enabled providers.

```sh
aws cloudformation deploy \
  --stack-name personal-tools \
  --template-file deploy/aws/cloudformation.yaml \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides \
    ProjectTag="$PROJECT_TAG" \
    LambdaCodeS3Bucket="$ARTIFACT_BUCKET" \
    MCPCodeS3Key="$MCP_KEY" \
    AuthorizerCodeS3Key="$AUTHORIZER_KEY" \
  --region "$AWS_REGION"
```

Provider modes default to disabled or mock. Enabling a production provider
requires all parameters asserted by the template Rules section.

## Secrets

The stack creates empty or generated Secrets. Populate Google OAuth credentials
and a GitHub App private key out of band through a narrow credential-operator
identity. Never pass those values through CloudFormation parameters or command
history.

## Operator policies

Follow [`policies/README.md`](./policies/README.md). Policy examples are not
automatically attached and must be rendered for the target account and Region,
reviewed, applied temporarily where possible, and recorded privately.
