# Hermes Credential Agent

This module contains the reusable host provisioner and protocol adapters that
consume Personal Tools credential leases.

## Components

| Binary/package | Responsibility |
|---|---|
| `hermes-credential-provisioner` | Uses the EC2 instance role to SigV4-sign a fixed profile request, refreshes before expiry, and atomically writes a `0600` lease |
| `git-credential-hermes` | Implements Git's `get/store/erase` helper protocol and returns only a valid lease matching HTTPS host and repository path |
| `internal/lease` | Validates the generic schema, expiry, and exact target boundaries |
| `internal/provisioner` | IAM request signing, redacted refresh loop, and atomic storage |
| `internal/gitcredential` | Git adapter parsing and matching; `store` and `erase` never persist input |

The deployment runs the provisioner outside the coding container and stores
leases under the user's systemd runtime directory (`/run/user/<uid>`), which is
tmpfs-backed. Only that lease directory is mounted read-only into the coding
container. The container does not receive the instance role, renewable lease
authority, GitHub App private key, Personal Tools bearer token, or Podman
socket.

The short-lived Git token is necessarily readable by code executing with the
same privilege inside the selected terminal container. The controls are short
expiry, one-repository scope, target matching, redacted logs, restricted
egress, and GitHub-enforced ref rules—not a claim of token invisibility.

The Makefile builds Linux amd64 binaries into
`dist/personal-tools-credential-agent.zip` for the current `t3` EC2 host.
