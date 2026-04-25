// pkg/note/git_docker.go
//
// Git and Docker notes for use in Katalog templates.
//
// These notes work in two directions:
//
//  1. Input to git:/docker: blocks — constructing the values those blocks need.
//  2. Output from git:/docker: blocks — transforming .git.* and .docker.* context.
//
// Git context injected by the git: block:
//
//	.git.commit       — full commit SHA
//	.git.changed      — "true" / "false"
//	.git.path         — local clone path
//	.git.branch       — resolved branch name
//	.git.error        — non-empty on failure (when continueOnError: true)
//	.git.called       — "true" when the block ran
//
// Docker context injected by the docker: block:
//
//	.docker.image     — fully qualified image reference (registry/repo:tag)
//	.docker.digest    — image digest after push (sha256:abc...)
//	.docker.buildSucceeded  — "true" / "false"
//	.docker.error           — non-empty on failure
//	.docker.called          — "true" when the block ran
//
// Example — image with commit-pinned tag:
//
//	docker:
//	  image: "{{ .spec.registry }}/{{ repoName .spec.repo }}:{{ gitShortCommit .git.commit }}"
//
// Example — only deploy when git changed and build succeeded:
//
//	deployments:
//	  - name: "{{ .metadata.name }}"
//	    image: "{{ dockerWithDigest .docker.image .docker.digest }}"
//	    when:
//	      - field: git.changed
//	        equals: "true"
//	      - field: docker.buildSucceeded
//	        equals: "true"
package note

import (
	"fmt"
	"strings"
	"text/template"
)

func gitNotes() template.FuncMap {
	return template.FuncMap{
		// Commit manipulation
		"gitShortCommit": noteGitShortCommit,
		"gitIsCommit":    noteGitIsCommit,

		// URL and repo parsing
		"repoName":       noteRepoName,       // "github.com/org/payments" → "payments"
		"repoOrg":        noteRepoOrg,        // "github.com/org/payments" → "org"
		"repoHost":       noteRepoHost,       // "github.com/org/payments" → "github.com"
		"repoSSHToHTTPS": noteRepoSSHToHTTPS, // "git@github.com:org/repo.git" → "https://github.com/org/repo"
		"repoHTTPSToSSH": noteRepoHTTPSToSSH, // reverse

		// Branch / ref helpers
		"gitDefaultBranch": noteGitDefaultBranch, // returns value or "main" if empty
		"gitRefShort":      noteGitRefShort,      // "refs/heads/main" → "main"

		// State helpers
		"gitChanged": noteGitChanged, // "true"/"false" string → bool-like "true"/"false"
	}
}

func dockerNotes() template.FuncMap {
	return template.FuncMap{
		// Image reference parsing
		"dockerRegistry": noteDockerRegistry, // "ghcr.io/org/app:v1" → "ghcr.io"
		"dockerRepo":     noteDockerRepo,     // "ghcr.io/org/app:v1" → "org/app"
		"dockerTag":      noteDockerTag,      // "ghcr.io/org/app:v1" → "v1"
		"dockerNoTag":    noteDockerNoTag,    // "ghcr.io/org/app:v1" → "ghcr.io/org/app"
		"dockerName":     noteDockerName,     // "ghcr.io/org/app:v1" → "app"

		// Image reference construction
		"dockerWithTag":    noteDockerWithTag,    // replace or add tag
		"dockerWithDigest": noteDockerWithDigest, // append @sha256:... for immutable reference
		"dockerCommitTag":  noteDockerCommitTag,  // build commit-tagged image reference

		// State helpers
		"dockerBuildSucceeded": noteDockerBuildSucceeded, // "true"/"false" string → "true"/"false"
		"dockerHasDigest":      noteDockerHasDigest,      // digest is non-empty
	}
}

// ── Git notes ─────────────────────────────────────────────────────────────────

// noteGitShortCommit returns the first 7 characters of a commit SHA.
// Standard "short SHA" format used in image tags and annotations.
//
//	{{ gitShortCommit .git.commit }}
//	→ "a3f5c2b"
func noteGitShortCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) < 7 {
		return commit
	}
	return commit[:7]
}

// noteGitIsCommit returns true when the string looks like a git SHA.
// Validates that a commit field was populated (not empty, not "false").
//
//	{{ gitIsCommit .git.commit }}
func noteGitIsCommit(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 7 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// noteRepoName extracts the repository name from a Git URL or path.
// Works with HTTPS URLs, SSH URLs, and plain org/repo paths.
//
//	{{ repoName "https://github.com/myorg/payments" }}  → "payments"
//	{{ repoName "git@github.com:myorg/payments.git" }}  → "payments"
//	{{ repoName .spec.repo }}
func noteRepoName(repo string) string {
	repo = normalizeRepo(repo)
	parts := strings.Split(repo, "/")
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]
	return strings.TrimSuffix(name, ".git")
}

// noteRepoOrg extracts the organization/owner from a Git URL.
//
//	{{ repoOrg "https://github.com/myorg/payments" }}  → "myorg"
func noteRepoOrg(repo string) string {
	repo = normalizeRepo(repo)
	parts := strings.Split(repo, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-2]
}

// noteRepoHost extracts the hostname from a Git URL.
//
//	{{ repoHost "https://github.com/myorg/payments" }}  → "github.com"
func noteRepoHost(repo string) string {
	repo = normalizeRepo(repo)
	parts := strings.Split(repo, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// noteRepoSSHToHTTPS converts a git@ SSH URL to an HTTPS URL.
//
//	{{ repoSSHToHTTPS "git@github.com:myorg/repo.git" }}
//	→ "https://github.com/myorg/repo"
func noteRepoSSHToHTTPS(repo string) string {
	repo = strings.TrimSpace(repo)
	if !strings.HasPrefix(repo, "git@") {
		return repo // already HTTPS or plain path
	}
	// git@github.com:org/repo.git → github.com:org/repo.git
	repo = strings.TrimPrefix(repo, "git@")
	// github.com:org/repo.git → github.com/org/repo.git
	repo = strings.Replace(repo, ":", "/", 1)
	repo = strings.TrimSuffix(repo, ".git")
	return "https://" + repo
}

// noteRepoHTTPSToSSH converts an HTTPS URL to a git@ SSH URL.
//
//	{{ repoHTTPSToSSH "https://github.com/myorg/repo" }}
//	→ "git@github.com:myorg/repo.git"
func noteRepoHTTPSToSSH(repo string) string {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimPrefix(repo, "https://")
	repo = strings.TrimPrefix(repo, "http://")
	// Find the first slash separating host from path
	idx := strings.Index(repo, "/")
	if idx < 0 {
		return repo
	}
	host := repo[:idx]
	path := repo[idx+1:]
	path = strings.TrimSuffix(path, ".git")
	return fmt.Sprintf("git@%s:%s.git", host, path)
}

// noteGitDefaultBranch returns the branch value, defaulting to "main" when empty.
// Prevents empty branch declarations from causing clone failures.
//
//	{{ gitDefaultBranch .spec.branch }}
//	{{ gitDefaultBranch "" }}  → "main"
func noteGitDefaultBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "main"
	}
	return branch
}

// noteGitRefShort extracts the short ref name from a full git ref.
//
//	{{ gitRefShort "refs/heads/main" }}          → "main"
//	{{ gitRefShort "refs/tags/v1.2.3" }}         → "v1.2.3"
//	{{ gitRefShort "refs/pull/42/merge" }}        → "42"
//	{{ gitRefShort "main" }}                      → "main"  (passthrough)
func noteGitRefShort(ref string) string {
	ref = strings.TrimSpace(ref)
	prefixes := []string{
		"refs/heads/",
		"refs/tags/",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(ref, p) {
			return strings.TrimPrefix(ref, p)
		}
	}
	if strings.HasPrefix(ref, "refs/pull/") {
		parts := strings.Split(ref, "/")
		if len(parts) >= 3 {
			return parts[2] // PR number
		}
	}
	return ref
}

// noteGitChanged returns "true" when git.changed is "true", "false" otherwise.
// Normalizes the string to a consistent boolean-like value for when: conditions.
//
//	{{ gitChanged .git.changed }}
func noteGitChanged(changed string) string {
	if strings.TrimSpace(strings.ToLower(changed)) == "true" {
		return "true"
	}
	return "false"
}

// normalizeRepo strips URL scheme and trailing slashes for consistent parsing.
func normalizeRepo(repo string) string {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.TrimSuffix(repo, "/")
	for _, prefix := range []string{"https://", "http://", "git://"} {
		repo = strings.TrimPrefix(repo, prefix)
	}
	return repo
}

// ── Docker notes ──────────────────────────────────────────────────────────────

// noteDockerRegistry extracts the registry hostname from an image reference.
// Returns "" for images that use the implicit Docker Hub registry.
//
//	{{ dockerRegistry "ghcr.io/myorg/app:v1" }}  → "ghcr.io"
//	{{ dockerRegistry "myorg/app:v1" }}           → "" (Docker Hub)
func noteDockerRegistry(image string) string {
	image = strings.TrimSpace(image)
	name := strings.Split(image, ":")[0]
	parts := strings.SplitN(name, "/", 2)
	// A registry hostname contains a dot or colon, or is "localhost"
	if len(parts) > 1 && (strings.Contains(parts[0], ".") ||
		strings.Contains(parts[0], ":") || parts[0] == "localhost") {
		return parts[0]
	}
	return ""
}

// noteDockerRepo extracts the repository path (without registry and tag).
//
//	{{ dockerRepo "ghcr.io/myorg/app:v1" }}  → "myorg/app"
//	{{ dockerRepo "myorg/app:v1" }}           → "myorg/app"
func noteDockerRepo(image string) string {
	image = strings.TrimSpace(image)
	// Remove digest
	if idx := strings.Index(image, "@"); idx >= 0 {
		image = image[:idx]
	}
	// Remove tag
	name := image
	if idx := strings.LastIndex(image, ":"); idx >= 0 {
		// Only treat as tag separator if it's after the last slash
		lastSlash := strings.LastIndex(image, "/")
		if idx > lastSlash {
			name = image[:idx]
		}
	}
	// Remove registry
	registry := noteDockerRegistry(image)
	if registry != "" {
		name = strings.TrimPrefix(name, registry+"/")
	}
	return name
}

// noteDockerTag extracts the tag from an image reference.
// Returns "latest" when no tag is present.
//
//	{{ dockerTag "ghcr.io/myorg/app:v1.2.3" }}  → "v1.2.3"
//	{{ dockerTag "ghcr.io/myorg/app" }}          → "latest"
func noteDockerTag(image string) string {
	image = strings.TrimSpace(image)
	// Digest reference — no tag
	if strings.Contains(image, "@") {
		return ""
	}
	lastSlash := strings.LastIndex(image, "/")
	nameAndTag := image[lastSlash+1:]
	if idx := strings.LastIndex(nameAndTag, ":"); idx >= 0 {
		return nameAndTag[idx+1:]
	}
	return "latest"
}

// noteDockerNoTag returns the image reference without the tag or digest.
// Useful for constructing a new reference with a different tag.
//
//	{{ dockerNoTag "ghcr.io/myorg/app:v1" }}  → "ghcr.io/myorg/app"
func noteDockerNoTag(image string) string {
	image = strings.TrimSpace(image)
	// Remove digest
	if idx := strings.Index(image, "@"); idx >= 0 {
		image = image[:idx]
	}
	// Remove tag — only after the last slash
	lastSlash := strings.LastIndex(image, "/")
	if idx := strings.LastIndex(image[lastSlash+1:], ":"); idx >= 0 {
		return image[:lastSlash+1+idx]
	}
	return image
}

// noteDockerName extracts just the image name (last path segment, no tag).
//
//	{{ dockerName "ghcr.io/myorg/app:v1" }}  → "app"
func noteDockerName(image string) string {
	repo := noteDockerRepo(image)
	parts := strings.Split(repo, "/")
	return parts[len(parts)-1]
}

// noteDockerWithTag returns the image reference with the tag replaced or added.
//
//	{{ dockerWithTag "ghcr.io/myorg/app" "v1.2.3" }}           → "ghcr.io/myorg/app:v1.2.3"
//	{{ dockerWithTag "ghcr.io/myorg/app:latest" "v1.2.3" }}    → "ghcr.io/myorg/app:v1.2.3"
//	{{ dockerWithTag .spec.image (gitShortCommit .git.commit) }} → image tagged with commit SHA
func noteDockerWithTag(image, tag string) string {
	if tag == "" {
		return image
	}
	return noteDockerNoTag(image) + ":" + tag
}

// noteDockerWithDigest appends a digest to an image reference for an immutable pin.
// Use after a successful push to lock the deployed image to a specific layer digest.
//
//	{{ dockerWithDigest .docker.image .docker.digest }}
//	→ "ghcr.io/myorg/app:v1@sha256:abc123..."
//
// When digest is empty (build not yet complete), returns the image unchanged.
func noteDockerWithDigest(image, digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return image
	}
	// Remove any existing digest
	if idx := strings.Index(image, "@"); idx >= 0 {
		image = image[:idx]
	}
	return image + "@" + digest
}

// noteDockerCommitTag builds a commit-tagged image reference.
// Combines a base image reference (without tag) with the short commit SHA.
// The canonical way to build a mutable-but-traceable image tag.
//
//	{{ dockerCommitTag .spec.registry .spec.repo .git.commit }}
//	→ "registry.io/myorg/app:a3f5c2b"
//
//	# Or using other notes:
//	{{ dockerWithTag (dockerNoTag .spec.image) (gitShortCommit .git.commit) }}
func noteDockerCommitTag(registry, repo, commit string) string {
	short := noteGitShortCommit(commit)
	if short == "" {
		short = "unknown"
	}
	registry = strings.TrimSuffix(strings.TrimSpace(registry), "/")
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if registry == "" {
		return fmt.Sprintf("%s:%s", repo, short)
	}
	return fmt.Sprintf("%s/%s:%s", registry, repo, short)
}

// noteDockerBuildSucceeded returns "true" when docker.buildSucceeded is "true".
//
//	{{ dockerBuildSucceeded .docker.buildSucceeded }}
func noteDockerBuildSucceeded(s string) string {
	if strings.TrimSpace(strings.ToLower(s)) == "true" {
		return "true"
	}
	return "false"
}

// noteDockerHasDigest returns true when a digest is present and non-empty.
// Use to gate deployment on a verified, pushed image.
//
//	when:
//	  - field: "{{ dockerHasDigest .docker.digest }}"
//	    equals: "true"
func noteDockerHasDigest(digest string) bool {
	digest = strings.TrimSpace(digest)
	return digest != "" && strings.HasPrefix(digest, "sha256:")
}
