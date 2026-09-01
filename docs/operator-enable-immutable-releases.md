# One-time immutable release setup

Run this once as a repository administrator after the public repository exists
and before `v0.1.0` is published. It is an operator action, not a workflow
secret. The endpoint is intentionally absent from all GitHub Actions files.

```text
gh api --method PUT \
  -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  repos/kimjooyoon/gooo-semantic-regression-bisector/immutable-releases
```

Verify the setting before release:

```text
gh api \
  -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  repos/kimjooyoon/gooo-semantic-regression-bisector/immutable-releases
```

The release workflow uses only the Actions-provided `github.token`. It creates
an annotated tag object, creates a draft release, uploads assets, publishes the
release, and then verifies the public release and Git tag APIs.
