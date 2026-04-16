package merger

import orktypes "github.com/orkspace/orkestra/pkg/types"

// Exported for tests
func ExportedGitHubRawURL(repoURL, ref, filePath string) string {
	return githubRawURL(repoURL, ref, filePath)
}

// Exported for tests
func ExportedGitLabRawURL(repoURL, ref, filePath string) string {
	return gitlabRawURL(repoURL, ref, filePath)
}

// ExportedLoadRegistrySource loads from the deprecated catalog-map registry protocol
// (sources.registry with katalog: map[string]RegistryRef).
// Tests that verify catalog-map-based registry loading use this export.
func ExportedLoadRegistrySource(m *Merger, src orktypes.RegistrySource) (map[string]orktypes.CRDEntry, error) {
	return m.loadRegistrySourceDeprecated(src)
}

// ExportedIsGitHubURL
func ExportedIsGitHubURL(u string) bool {
	return isGitHubURL(u)
}

// ExportedIsGitLab
func ExportedIsGitLabURL(u string) bool {
	return isGitLabURL(u)
}

// ExportedValidatePatternStructure
func ExportedValidatePatternStructure(dir, url, version string) error {
	return validatePatternStructure(dir, url, version)
}
