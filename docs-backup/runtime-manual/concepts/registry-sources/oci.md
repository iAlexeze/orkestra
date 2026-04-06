# The `oci` field

Controls the pull mechanism. `false` by default.

```yaml
oci: false   # default — pull via Git or raw HTTP
oci: true    # pull as an OCI artifact using ORAS
```

**`oci: false` (default)**

Orkestra determines the pull method from the URL:

- GitHub URLs → raw file HTTP, one request per required file (fast, no clone)
- GitLab URLs → raw file HTTP, one request per required file
- Any other URL → `git clone` at the specified ref

You do not need to declare `oci: false` — it is the default. The field exists
to make the intent explicit when you want to be clear in a Komposer that a
source is Git-based.

**`oci: true`**

Orkestra pulls the artifact using the ORAS protocol and extracts the files.
The artifact reference is constructed as `url:version`.

```yaml
# oci: true — OCI pull
# Artifact ref constructed as: ghcr.io/myorg/postgres:v14
- url: ghcr.io/myorg/postgres@v14
  oci: true
```

!!! note "ORAS dependency"
    OCI pulls currently shell out to the `oras` CLI. Ensure `oras` is
    installed and available on `PATH` when using `oci: true`.

```bash
# Install oras
brew install oras

# Verify
oras version
```

Native ORAS Go library integration is planned for a future release, which
will remove this dependency.

!!! tip "When to use each"
    Use `oci: false` (default) when your registry is a GitHub or GitLab
    repository — no extra tooling required, and individual file fetching
    is faster than pulling a full OCI artifact.

    Use `oci: true` when your patterns are published as OCI artifacts
    (the standard for the public OrkestraRegistry) or when you want the
    guarantees of OCI distribution: immutable tags, content-addressable
    digests, and registry access controls.

