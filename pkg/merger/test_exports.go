package merger

func ExportedGitHubRawURL(repoURL, ref, filePath string) string {
	return githubRawURL(repoURL, ref, filePath)
}

func ExportedGitLabRawURL(repoURL, ref, filePath string) string {
	return gitlabRawURL(repoURL, ref, filePath)
}

func ExportedIsGitHubURL(u string) bool {
	return isGitHubURL(u)
}

func ExportedIsGitLabURL(u string) bool {
	return isGitLabURL(u)
}

func ExportedValidatePatternStructure(dir, url, version string) error {
	return validatePatternStructure(dir, url, version)
}
