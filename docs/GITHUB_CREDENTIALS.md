# GitHub Credential Integration

## Components and calls

```text
systemd starts hermes-credential-provisioner on the host
  -> SigV4 POST /v1/credential-leases with a fixed profile
  -> provider registry selects the GitHub App issuer
  -> issuer creates an App JWT and requests one installation token
  -> provisioner validates target and expiry, then writes a tmpfs lease
  -> coding container mounts the lease directory read-only
  -> Git invokes git-credential-hermes get for an HTTPS remote
  -> helper returns the matching short-lived username/password to Git
```

The model does not call `git-credential-hermes` directly. Git invokes it after
the operator configures:

```sh
git config --global credential.helper hermes
```

The helper executable must be on `PATH` inside the coding container.

## GitHub App boundary

Create a dedicated GitHub App with the minimum repository permissions required
by the configured issuer. Install it only on explicitly selected repositories.
Use repository Rulesets or branch protection to deny direct writes to the
integration branch and other disallowed refs.

The App installation token limits repository and API permission scope; GitHub
ref protection limits which pushes are accepted. Both controls are required.

Do not install the App on a private repository unless its plan actually
enforces the configured protection rules.

## Activation order

1. Create and verify protected-ref rules.
2. Create the dedicated App and install it on selected repositories only.
3. Store the App private key out of band in the issuer's Secret.
4. Deploy the Credential Lease service in disabled mode, then enable the fixed
   profile with numeric IDs and repository coordinates.
5. Start the host provisioner and mount the tmpfs lease read-only into the
   coding container.
6. Test allowed working-branch push and denied integration-branch, other-ref,
   tag, expired-token, IMDS, and uncatalogued-egress paths.
