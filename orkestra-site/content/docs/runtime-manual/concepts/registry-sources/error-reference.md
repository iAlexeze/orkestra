---
title: "Error Reference"
weight: 132
---

# Error reference

## Structure validation errors

These errors occur when a pulled pattern is missing required files or
contains empty files. They fire during `ork validate` and at startup.

```
registry pattern "<url>"@<version> failed structure validation:
  missing: <filename>
  empty:   <filename>
```

**Resolution:** Add the missing file or populate the empty one in the
upstream pattern. If you do not control the pattern, raise an issue with
the pattern maintainer.

## URL parse errors

```
registry "<url>": building request: invalid URL
```

Common causes: `oci://` scheme prefix in the URL (remove it), invalid
characters in the URL, whitespace in the URL field.

## Version not found

```
registry "<url>"@<version>: pull failed: reference not found
```

The version tag or branch does not exist in the registry. Check the
available versions with `oras discover` (OCI) or by browsing the repository
(Git).

## Auth failures

```
fetching "katalog.yaml" from GitHub at ref "main": authentication required (401)
  — check that auth credentials are set and have not expired
```

```
fetching "katalog.yaml": access denied (403)
  — check that the token has sufficient permissions
```

**Resolution:** Verify the environment variable named in `fromEnv` is set
and contains a valid, non-expired token with read access to the repository
or registry.

## Kind mismatch

```
registry "<url>"@<version>: useKomposer is false but katalog.yaml contains
kind "Komposer" — set useKomposer: true to load the upstream Komposer,
or check the pattern structure
```

**Resolution:** Either set `useKomposer: true` to load the Komposer, or
check whether the pattern has a separate `katalog.yaml` containing a Katalog
as expected.

## ORAS not found

```
registry "<url>"@<version>: OCI pull: exec: "oras": executable file not found in $PATH
```

**Resolution:** Install `oras`: `brew install oras` or follow the
[ORAS installation guide](https://oras.land/docs/installation).
