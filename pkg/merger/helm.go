// pkg/merger/helm.go
package merger

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	helmvals "helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

// loadHelmSource renders a Helm chart and extracts Katalog CRD definitions.
// The chart must contain at least one template with kind: Katalog.
// pkg/merger/helm.go

func (m *Merger) loadHelmSource(src orktypes.HelmSource) ([]orktypes.CRDEntry, error) {
	logger.Info().
		Str("repo", src.Repo).
		Str("chart", src.Chart).
		Str("version", src.Version).
		Msg("merger: loading helm source")

	chartPath, cleanup, err := resolveChartPath(src)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return renderAndExtract(src, chartPath)
}

// resolveChartPath returns the local path to the chart directory or .tgz,
// downloading or cloning if necessary.
// cleanup is non-nil when a temp directory was created and must be removed.
func resolveChartPath(src orktypes.HelmSource) (chartPath string, cleanup func(), err error) {
	switch {
	case isGitURL(src.Repo):
		// ── Git source ────────────────────────────────────────────────────
		// Clone the repo to a temp dir, then use the chart subdirectory.
		return resolveGitChart(src)

	case isLocalPath(src.Repo):
		// ── Local source ──────────────────────────────────────────────────
		// repo is a directory. chart is a subdirectory within it.
		// If chart is empty, use repo directly.
		if src.Chart != "" {
			return filepath.Join(src.Repo, src.Chart), nil, nil
		}
		return src.Repo, nil, nil

	default:
		// ── Remote Helm repo ──────────────────────────────────────────────
		return resolveRemoteChart(src)
	}
}

// isLocalPath returns true for file system paths.
// Handles absolute paths, relative paths, and ~ home directory.
func isLocalPath(s string) bool {
	return strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "~") ||
		// no scheme at all — treat as relative path
		(!strings.Contains(s, "://") && !strings.HasPrefix(s, "http"))
}

// isGitURL returns true for git repository URLs.
func isGitURL(s string) bool {
	return strings.HasSuffix(s, ".git") ||
		strings.HasPrefix(s, "git://") ||
		strings.HasPrefix(s, "git@") ||
		// GitHub/GitLab URLs that point to a repo (not a file)
		(strings.Contains(s, "github.com") || strings.Contains(s, "gitlab.com")) &&
			!strings.Contains(s, "/raw/")
}

// resolveLocalChart validates and returns the local chart path.
func resolveLocalChart(src orktypes.HelmSource) (string, func(), error) {
	chartPath := src.Repo
	if src.Chart != "" {
		chartPath = filepath.Join(src.Repo, src.Chart)
	}

	// Resolve ~ to home directory
	if strings.HasPrefix(chartPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", nil, fmt.Errorf("resolving home dir: %w", err)
		}
		chartPath = filepath.Join(home, chartPath[1:])
	}

	if _, err := os.Stat(chartPath); err != nil {
		return "", nil, fmt.Errorf("local chart path %q not found: %w", chartPath, err)
	}

	return chartPath, nil, nil
}

// resolveGitChart clones the git repository to a temp directory
// and returns the path to the chart within it.
func resolveGitChart(src orktypes.HelmSource) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "orkestra-git-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir for git clone: %w", err)
	}

	cleanup := func() { os.RemoveAll(tmpDir) }

	// Build the git clone command
	// ref can be a branch, tag, or commit
	ref := src.Version // version doubles as git ref for git sources
	if ref == "" {
		ref = "HEAD"
	}

	logger.Info().
		Str("repo", src.Repo).
		Str("ref", ref).
		Msg("merger: cloning git repository")

	// Shallow clone at the specific ref
	cmd := exec.Command("git", "clone",
		"--depth", "1",
		"--branch", ref,
		src.Repo,
		tmpDir,
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		// Branch clone failed — try as commit hash with full clone
		cmd = exec.Command("git", "clone", src.Repo, tmpDir)
		if err2 := cmd.Run(); err2 != nil {
			cleanup()
			return "", nil, fmt.Errorf("cloning %q: %w", src.Repo, err)
		}

		// Checkout the specific commit/ref
		checkout := exec.Command("git", "-C", tmpDir, "checkout", ref)
		if err3 := checkout.Run(); err3 != nil {
			cleanup()
			return "", nil, fmt.Errorf("checking out %q in %q: %w", ref, src.Repo, err3)
		}
	}

	// Chart lives at src.Path within the repo, or src.Chart, or root
	chartPath := tmpDir
	if src.Path != "" {
		chartPath = filepath.Join(tmpDir, src.Path)
	} else if src.Chart != "" {
		chartPath = filepath.Join(tmpDir, src.Chart)
	}

	if _, err := os.Stat(chartPath); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("chart path %q not found in repo %q", chartPath, src.Repo)
	}

	logger.Info().
		Str("repo", src.Repo).
		Str("ref", ref).
		Str("chartPath", chartPath).
		Msg("merger: git repo cloned")

	return chartPath, cleanup, nil
}

// resolveRemoteChart pulls a chart from a remote Helm repository.
func resolveRemoteChart(src orktypes.HelmSource) (string, func(), error) {
	settings := cli.New()
	cfg := &action.Configuration{}

	repoEntry := &repo.Entry{
		Name: "orkestra-tmp",
		URL:  src.Repo,
	}

	chartRepo, err := repo.NewChartRepository(repoEntry, getter.All(settings))
	if err != nil {
		return "", nil, fmt.Errorf("creating chart repo for %q: %w", src.Repo, err)
	}

	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return "", nil, fmt.Errorf("downloading repo index from %q: %w", src.Repo, err)
	}

	tmpDir, err := os.MkdirTemp("", "orkestra-helm-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}

	cleanup := func() { os.RemoveAll(tmpDir) }

	pull := action.NewPullWithOpts(action.WithConfig(cfg))
	pull.Settings = settings
	pull.RepoURL = src.Repo
	pull.Version = src.Version
	pull.Untar = true
	pull.DestDir = tmpDir

	if _, err := pull.Run(src.Chart); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("pulling chart %q@%s: %w", src.Chart, src.Version, err)
	}

	return filepath.Join(tmpDir, src.Chart), cleanup, nil
}

// renderAndExtract renders a chart from a local path and extracts Katalog CRDs.
func renderAndExtract(src orktypes.HelmSource, chartPath string) ([]orktypes.CRDEntry, error) {
	settings := cli.New()

	// ── Load value files ──────────────────────────────────────────────────────
	valueOpts := &helmvals.Options{}
	for _, vf := range src.ValueFiles {
		resolved, err := resolveEnvVar(vf)
		if err != nil {
			return nil, fmt.Errorf("resolving value file %q: %w", vf, err)
		}

		if strings.HasPrefix(resolved, "http") {
			data, err := utils.LoadFile(resolved)
			if err != nil {
				return nil, fmt.Errorf("fetching remote values %q: %w", resolved, err)
			}
			tmp, err := writeTempFile(data, "orkestra-values-*.yaml")
			if err != nil {
				return nil, err
			}
			defer os.Remove(tmp)
			resolved = tmp
		}

		valueOpts.ValueFiles = append(valueOpts.ValueFiles, resolved)
	}

	vals, err := valueOpts.MergeValues(getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("merging helm values: %w", err)
	}

	for k, v := range src.Values {
		vals[k] = v
	}

	// ── Load and render ───────────────────────────────────────────────────────
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("loading chart from %q: %w", chartPath, err)
	}

	cfg := &action.Configuration{}
	install := action.NewInstall(cfg)
	install.DryRun = true
	install.ReleaseName = "orkestra-render"
	install.ClientOnly = true
	install.IncludeCRDs = true

	release, err := install.Run(chart, vals)
	if err != nil {
		return nil, fmt.Errorf("rendering chart %q: %w", src.Chart, err)
	}

	return extractKatalogCRDs(release.Manifest, src.Chart)
}

// renderAndExtract renders a Helm chart and extracts Katalog CRD definitions.

// extractKatalogCRDs parses rendered Helm output and extracts
// CRD definitions from any template with kind: Katalog.
func extractKatalogCRDs(manifest, chartName string) ([]orktypes.CRDEntry, error) {
	var allCRDs []orktypes.CRDEntry

	// Split on YAML document separator — one chart renders multiple templates
	docs := strings.Split(manifest, "\n---\n")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		katalog, err := parseKatalogDoc([]byte(doc), "helm:"+chartName)
		if err != nil {
			return nil, err // malformed Katalog — hard error
		}
		if katalog == nil {
			continue // not a Katalog — skip
		}

		allCRDs = append(allCRDs, katalog.Spec.CRDs...)

		logger.Debug().
			Str("chart", chartName).
			Int("crds", len(katalog.Spec.CRDs)).
			Msg("merger: extracted CRDs from helm template")
	}

	if len(allCRDs) == 0 {
		return nil, fmt.Errorf(
			"chart %q produced no Katalog templates — "+
				"ensure at least one template has kind: %s",
			chartName, konfig.KatalogKind(),
		)
	}

	return allCRDs, nil
}
