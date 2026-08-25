package types

// Git declares a Git-backed trigger for this lifecycle hook.
//
// When configured, Orkestra performs a lightweight Git checkout or update
// before evaluating the rest of the hook. The result is exposed to templates
// under `.git`:
//
//	.git.commit     — latest commit hash
//	.git.changed    — "true" if the commit changed since last reconcile
//	.git.path       — absolute path to the working directory
//
// This enables declarative, in-cluster CI/CD pipelines where Git acts as
// the source of build/test/deploy logic.
//
// Minimal v1 behaviour:
//   - On first reconcile: clone repo into a per-CRD working directory.
//   - On subsequent reconciles: fetch + fast-forward.
//   - If HEAD changed: mark `.git.changed = "true"`.
//   - No per-reconcile full clone.
//   - No credentials stored in CRD.
//   - No aggressive polling — relies on CRD-level resync.
//
// Future versions may add shallow clones, webhook triggers, caching,
// subdirectory diffing, and credential sources without breaking this contract.
type GitHookSpec struct {
	// Repo is the Git repository URL.
	Repo string `yaml:"repo" json:"repo"`

	// Branch is the branch to track. Default: "main".
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`

	// Path optionally scopes change detection to a subdirectory.
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// Reconcile controls whether this Git hook runs on every reconcile.
	//
	// When true:
	//   • Git fetch runs on every reconcile cycle.
	//   • `.git.changed` is updated accordingly.
	//
	// When false:
	//   • Git runs only on onCreate.
	//   • Useful for one-time initialization.
	Reconcile bool `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`

	// ContinueOnError controls behaviour when the Git operation fails.
	//
	// false (default) — Git failure returns an error and halts reconciliation.
	// true            — Git failure injects .git.error but reconciliation continues.
	//                   Subsequent when: conditions on git.changed will not fire.
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

	Or []Condition `yaml:"or,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
