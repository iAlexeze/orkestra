# 28 — Git Notes

## In Development

Parse and transform Git repository URLs, commit SHAs, branch refs, and the `.git.*` context injected by the `git:` block. These notes work in two directions: constructing values the `git:` block needs, and transforming the context it injects after running.

## Git context

When `git:` runs, it injects these fields into the template context:

| Field | Value |
|-------|-------|
| `.git.commit` | full commit SHA |
| `.git.branch` | resolved branch name |
| `.git.path` | local clone path |
| `.git.changed` | `"true"` / `"false"` |
| `.git.error` | non-empty when `continueOnError: true` and clone failed |
| `.git.called` | `"true"` when the block ran |

## Reference

### `gitShortCommit`

Return the first 7 characters of a commit SHA — the standard short format used in image tags and annotations.

Keywords: git, commit, sha, short, string, tag

```yaml
# Tag an image with the commit SHA
docker:
  image: "ghcr.io/myorg/app:{{ gitShortCommit .git.commit }}"

# Annotate the deployed resource
metadata:
  annotations:
    myorg.io/commit: "{{ gitShortCommit .git.commit }}"
```

---

### `gitIsCommit`

Return `true` when the string looks like a git SHA (7+ hex characters). Use to verify the `git:` block ran and populated `.git.commit` before using it.

Keywords: git, commit, sha, valid, boolean, check

```yaml
when:
  - field: "{{ gitIsCommit .git.commit }}"
    equals: "true"
```

---

### `repoName`

Extract the repository name (last path segment) from a Git URL or path. Accepts HTTPS URLs, SSH URLs, and plain `org/repo` paths. Strips `.git` suffix automatically.

Keywords: git, repo, name, url, parse, string

```yaml
# "https://github.com/myorg/payments" → "payments"
# "git@github.com:myorg/payments.git" → "payments"
name: "{{ repoName .spec.repo }}-operator"
```

---

### `repoOrg`

Extract the organization or owner from a Git URL.

Keywords: git, repo, org, owner, url, parse, string

```yaml
# "https://github.com/myorg/payments" → "myorg"
metadata:
  labels:
    myorg.io/owner: "{{ repoOrg .spec.repo }}"
```

---

### `repoHost`

Extract the hostname from a Git URL.

Keywords: git, repo, host, hostname, url, parse, string

```yaml
# "https://github.com/myorg/payments" → "github.com"
metadata:
  annotations:
    myorg.io/git-host: "{{ repoHost .spec.repo }}"
```

---

### `repoSSHToHTTPS` / `repoHTTPSToSSH`

Convert between SSH (`git@`) and HTTPS (`https://`) Git URL formats.

Keywords: git, repo, url, ssh, https, convert, string

```yaml
# SSH → HTTPS (for display or HTTP clone)
# "git@github.com:myorg/repo.git" → "https://github.com/myorg/repo"
value: "{{ repoSSHToHTTPS .spec.repo }}"

# HTTPS → SSH (for clone with deploy key)
# "https://github.com/myorg/repo" → "git@github.com:myorg/repo.git"
value: "{{ repoHTTPSToSSH .spec.repo }}"
```

---

### `gitDefaultBranch`

Return the branch value, or `"main"` when empty. Prevents empty branch fields from causing clone failures in the `git:` block.

Keywords: git, branch, default, main, fallback, string

```yaml
git:
  repo: "{{ .spec.repo }}"
  branch: "{{ gitDefaultBranch .spec.branch }}"
```

---

### `gitRefShort`

Extract the short name from a full git ref. Handles branches, tags, and PR refs.

Keywords: git, ref, short, branch, tag, parse, string

```yaml
# "refs/heads/main"     → "main"
# "refs/tags/v1.2.3"   → "v1.2.3"
# "refs/pull/42/merge"  → "42"
metadata:
  annotations:
    myorg.io/branch: "{{ gitRefShort .git.branch }}"
```

---

### `gitChanged`

Normalize `.git.changed` to `"true"` or `"false"`. Use in `when:` conditions to gate resources on whether the repository had changes since the last clone.

Keywords: git, changed, boolean, condition, when, string

```yaml
# Only rebuild when git reports changes
when:
  - field: "{{ gitChanged .git.changed }}"
    equals: "true"
```

---

## Quick reference

| Note | Accepts | Returns | Notes |
|------|---------|---------|-------|
| `gitShortCommit` | `commit string` | `string` | first 7 chars |
| `gitIsCommit` | `s string` | `bool` | 7+ hex chars |
| `repoName` | `repo string` | `string` | last path segment, no `.git` |
| `repoOrg` | `repo string` | `string` | second-to-last segment |
| `repoHost` | `repo string` | `string` | hostname only |
| `repoSSHToHTTPS` | `repo string` | `string` | git@ → https:// |
| `repoHTTPSToSSH` | `repo string` | `string` | https:// → git@ |
| `gitDefaultBranch` | `branch string` | `string` | `""` → `"main"` |
| `gitRefShort` | `ref string` | `string` | strips `refs/heads/`, `refs/tags/` |
| `gitChanged` | `changed string` | `string` | normalized `"true"` / `"false"` |
