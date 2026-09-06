#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
template="$script_dir/cloudformation.yaml"
artifacts_template="$script_dir/artifacts-cloudformation.yaml"

ruby -e 'require "yaml"; ARGV.each { |path| YAML.parse_file(path) }' \
  "$template" "$artifacts_template"

ruby -rjson -e '
  ARGV.each do |path|
    body = File.read(path)
      .gsub("${AWS_ACCOUNT_ID}", "123456789012")
      .gsub("${AWS_REGION}", "us-east-1")
    JSON.parse(body)
  end
' "$script_dir"/policies/*.json.tmpl

rg -q 'RouteKey: POST /v1/credential-leases' "$template"
rg -q 'AuthorizationType: AWS_IAM' "$template"
rg -q 'GITHUB_PRIVATE_KEY_SECRET_ARN: !Ref GitHubAppPrivateKeySecret' "$template"
rg -q 'Resource: !Ref GitHubAppPrivateKeySecret' "$template"
rg -q 'ProjectTag:' "$template"
rg -q 'Value: !Ref ProjectTag' "$template"
rg -q '"requestId".*"routeKey".*"status".*"responseLength".*"integrationError"' "$template"

if rg -n 'requestBody|responseBody|dataTraceEnabled|authorizationValue' "$template"; then
  echo 'API logging must not include sensitive request, response, or authorization values' >&2
  exit 1
fi

if [[ "${1:-}" == "--aws" ]]; then
  region="${2:?usage: validate-template.sh --aws AWS_REGION}"
  aws cloudformation validate-template --template-body "file://$template" --region "$region" >/dev/null
  aws cloudformation validate-template --template-body "file://$artifacts_template" --region "$region" >/dev/null
fi

echo 'personal-tools-template-validation-ok'
