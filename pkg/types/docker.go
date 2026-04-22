package types

// DockerHookSpec declares a Docker build/push operation as part of a lifecycle hook.
//
// This hook allows Orkestra to build container images directly inside the
// reconcile loop, using the working directory produced by a Git hook or any
// other source. It behaves similarly to external calls and git hooks — it is
// a declarative precondition that runs before resource creation.
//
// Typical usage:
//
//	operatorBox:
//	  onReconcile:
//	    git:
//	      repo: "git@github.com:org/webapp.git"
//	      branch: "main"
//	      path: "services/webapp"
//	      reconcile: true
//
//	    docker:
//	      image: "registry.example.com/webapp:{{ .git.commit }}"
//	      workingDirectory: "{{ .git.path }}"
//	      push: true
//
//	    deployments:
//	      - name: "{{ .metadata.name }}"
//	        image: "{{ .docker.image }}"
//
// Minimal v1 behaviour:
//   - Executes `docker build` using workingDirectory as the build context.
//   - Uses Dockerfile in workingDirectory unless overridden.
//   - If push=true, executes `docker push` after a successful build.
//   - Exposes results to templates under `.docker.*`:
//     .docker.image
//     .docker.buildSucceeded
//     .docker.error
//   - No credentials stored in the CRD — authentication is external.
//   - No per-reconcile full rebuild unless reconcile=true.
//   - No caching or layer reuse guarantees (future versions may add this).
//
// This enables fully declarative CI/CD pipelines where Git → Test → Build → Push → Deploy
// are all expressed in YAML and executed inside the cluster.
type DockerHookSpec struct {
	// Image is the fully-qualified image reference to build.
	//
	// Examples:
	//   "registry.example.com/webapp:{{ .git.commit }}"
	//   "ghcr.io/org/service:{{ .status.version }}"
	//
	// This field supports template expressions and is required.
	Image string `yaml:"image" json:"image"`

	// WorkingDirectory is the directory used as the Docker build context.
	//
	// In most cases this is the same as the Git hook's .git.path:
	//   workingDirectory: "{{ .git.path }}"
	//
	// If omitted, Orkestra defaults to the CR's working directory (rarely useful).
	WorkingDirectory string `yaml:"workingDirectory,omitempty" json:"workingDirectory,omitempty"`

	// Dockerfile optionally overrides the Dockerfile path.
	//
	// Examples:
	//   "Dockerfile"
	//   "deploy/Dockerfile.prod"
	//
	// When empty, Orkestra uses "Dockerfile" in the working directory.
	Dockerfile string `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`

	// Push controls whether the built image should be pushed to the registry.
	//
	// When true:
	//   • Orkestra executes `docker push <image>` after a successful build.
	//   • Push failures are surfaced via .docker.error.
	//
	// When false:
	//   • The image is built locally only.
	Push bool `yaml:"push,omitempty" json:"push,omitempty"`

	// Builder selects the OCI build tool to use.
	//
	// Supported values: "docker" (default), "kaniko", "buildah", "podman".
	// When empty, Orkestra checks the OCI_BUILDER environment variable,
	// then falls back to "docker".
	//
	// kaniko — builds without a Docker socket; works in any Kubernetes pod.
	// buildah — rootless builds via buildah bud.
	// podman  — local builds via podman build.
	Builder string `yaml:"builder,omitempty" json:"builder,omitempty"`

	// Reconcile controls whether this Docker hook runs on every reconcile.
	//
	// When true:
	//   • Docker build/push runs on every reconcile cycle.
	//   • Useful for Git-driven pipelines where .git.changed triggers rebuilds.
	//
	// When false:
	//   • Docker build/push runs only during onCreate.
	//   • Useful for one-time initialization images.
	Reconcile bool `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`

	// ContinueOnError controls behaviour when the Docker build or push fails.
	//
	// false (default) — Docker failure returns an error and halts reconciliation.
	// true            — Docker failure injects .docker.error but reconciliation continues.
	ContinueOnError bool `yaml:"continueOnError,omitempty" json:"continueOnError,omitempty"`

	// When is an optional list of conditions that must all pass before
	// this field is written. If absent or empty, the field is always written.
	//
	// All conditions are AND-ed together.
	// To express OR logic, declare multiple StatusField entries for the same path.
	//
	// Conditions are evaluated against the full CR object map — the same
	// map available to template expressions. This means .status.phase,
	// .spec.image, .children.job.status.succeeded are all accessible.
	When []Condition `yaml:"when,omitempty"`

	AnyOf []Condition `yaml:"anyOf,omitempty"`
}
