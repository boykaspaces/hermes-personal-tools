#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

for path in PROJECT.md AGENTS.md .hermes/context-index.md .hermes/state.md \
  .hermes/checkpoints/README.md tasks/README.md tasks/current.md \
  docs/decisions/README.md; do
  test -f "$repo_root/$path" || { echo "missing context path: $path" >&2; exit 1; }
done

for path in \
  "$repo_root/services/authorizer/go.mod" \
  "$repo_root/services/mcp/go.mod" \
  "$repo_root/services/credentials/go.mod" \
  "$repo_root/clients/credential-agent/go.mod"; do
  grep -q '^module github.com/boykaspaces/hermes-personal-tools/' "$path"
done

if rg -n --glob '!scripts/validate.sh' '(github\.com/boyka/|210122338617|i-0f170ae7baf762606|2ugu1wgqaa|boyka5945@gmail\.com|boyka-[a-z-]+-policy)' "$repo_root"; then
  echo 'private deployment identifier detected' >&2
  exit 1
fi

if find "$repo_root" \( -name DEPLOYMENT_RECORD.md -o -name .DS_Store -o -name '*.pem' -o -name '*.key' -o -name .env \) -print -quit | grep -q .; then
  echo 'private, generated, or credential file detected' >&2
  exit 1
fi

bash -n "$repo_root/deploy/aws/validate-template.sh"
"$repo_root/deploy/aws/validate-template.sh"

ruby - "$repo_root" <<'RUBY'
require "pathname"
root = Pathname.new(ARGV.fetch(0))
errors = []
root.glob("**/*.md").sort.each do |file|
  file.read.scan(/\[[^\]]*\]\(([^)]+)\)/).flatten.each do |raw|
    target = raw.strip
    next if target.empty? || target.start_with?("http://", "https://", "mailto:", "#")
    target = target.split("#", 2).first
    next if target.empty?
    resolved = file.dirname.join(target).cleanpath
    errors << "#{file.relative_path_from(root)}: #{raw}" unless resolved.exist?
  end
end
abort(errors.join("\n")) unless errors.empty?
puts "markdown-relative-links-ok"
RUBY

active_task="$(awk -F': ' '/^Active Task:/ {print $2}' "$repo_root/tasks/current.md")"
state_task="$(awk -F': ' '/^Active Task:/ {print $2}' "$repo_root/.hermes/state.md")"
test "$active_task" = "$state_task" || {
  echo 'Task and State current pointers disagree' >&2
  exit 1
}

echo 'hermes-personal-tools repository validation passed'
