//go:build !runtime && !gateway

package cli

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/orkspace/orkestra/pkg/utils"
)

// applyCRFilesIfNeeded applies crFiles declarations via kubectl in order before
// the runtime starts. Only runs outside the cluster (dev mode).
func applyCRFilesIfNeeded(ctx context.Context, katalogPath string, m *merger.Merger) {
	if utils.IsRunningInCluster() {
		return
	}

	katalogDir := filepath.Dir(katalogPath)

	for crdName, entry := range m.All() {
		if !entry.HasCRFiles() {
			continue
		}
		for _, crFile := range entry.CRFiles {
			path := crFile
			if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
				path = filepath.Join(katalogDir, path)
			}

			out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", path).CombinedOutput()
			if err != nil {
				logger.Warn().
					Str("crd", crdName).
					Str("path", path).
					Str("output", strings.TrimSpace(string(out))).
					Err(err).
					Msg("crFile pre-apply failed (continuing)")
			} else {
				logger.Info().
					Str("crd", crdName).
					Str("path", path).
					Msg("crFile applied")
			}
		}
	}
}

// waitForCRDsEstablished waits for any CRDs that have crFiles to be Established
// in the cluster before CRs are applied. Only blocks for CRDs that need it.
func waitForCRDsEstablished(ctx context.Context, m *merger.Merger) {
	if utils.IsRunningInCluster() {
		return
	}

	for crdName, entry := range m.All() {
		if len(entry.CRFiles) == 0 {
			continue
		}

		plural := entry.APITypes.Plural
		group := entry.APITypes.Group
		if plural == "" || group == "" {
			continue
		}

		crdFullName := plural + "." + group

		out, err := exec.CommandContext(ctx, "kubectl", "wait",
			"--for=condition=Established",
			"--timeout=30s",
			"crd/"+crdFullName,
		).CombinedOutput()
		if err != nil {
			logger.Warn().
				Str("crd", crdName).
				Str("name", crdFullName).
				Str("output", strings.TrimSpace(string(out))).
				Err(err).
				Msg("CRD not established in time — applying CRs anyway")
		} else {
			logger.Info().
				Str("crd", crdName).
				Str("name", crdFullName).
				Msg("CRD established")
		}
	}
}

// applyCRDFilesIfNeeded applies any crdFile declarations via kubectl before the
// operator starts. Only runs outside the cluster (dev mode). In production,
// CRDs must be pre-applied by the platform operator.
func applyCRDFilesIfNeeded(ctx context.Context, katalogPath string, m *merger.Merger) {
	if utils.IsRunningInCluster() {
		return
	}

	katalogDir := filepath.Dir(katalogPath)

	for crdName, entry := range m.All() {
		if !entry.HasCRDFile() {
			continue
		}

		path := entry.CRDFile
		if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
			path = filepath.Join(katalogDir, path)
		}

		out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", path).CombinedOutput()
		if err != nil {
			logger.Warn().
				Str("crd", crdName).
				Str("path", path).
				Str("output", strings.TrimSpace(string(out))).
				Err(err).
				Msg("crdFile pre-apply failed (continuing)")
		} else {
			logger.Info().
				Str("crd", crdName).
				Str("path", path).
				Msg("crdFile applied")
		}
	}
}

// applySetupIfNeeded applies setup YAML files via kubectl in order before
// Orkestra starts. Only runs outside the cluster (dev mode).
func applySetupIfNeeded(ctx context.Context, katalogPath string, m *merger.Merger) {
	if utils.IsRunningInCluster() {
		return
	}

	katalogDir := filepath.Dir(katalogPath)

	for crdName, entry := range m.All() {
		if !entry.HasSetup() {
			continue
		}

		for _, setupFile := range entry.Setup {
			path := setupFile
			if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
				path = filepath.Join(katalogDir, path)
			}

			out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", path).CombinedOutput()
			if err != nil {
				logger.Warn().
					Str("crd", crdName).
					Str("path", path).
					Str("output", strings.TrimSpace(string(out))).
					Err(err).
					Msg("setup pre-apply failed (continuing)")
			} else {
				logger.Info().
					Str("crd", crdName).
					Str("path", path).
					Msg("setup applied")
			}
		}
	}
}
