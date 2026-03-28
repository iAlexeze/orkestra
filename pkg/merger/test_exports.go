package merger

import orktypes "github.com/ialexeze/orkestra/pkg/types"

// Exported for tests
func ExportedGitHubRawURL(repoURL, ref, filePath string) string {
	return githubRawURL(repoURL, ref, filePath)
}

// Exported for tests
func ExportedGitLabRawURL(repoURL, ref, filePath string) string {
	return gitlabRawURL(repoURL, ref, filePath)
}

// ExportedLoadRegistrySource
func ExportedLoadRegistrySource(m *Merger, src orktypes.RegistrySource) ([]orktypes.CRDEntry, error) {
	return m.loadRegistrySource(src)
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
