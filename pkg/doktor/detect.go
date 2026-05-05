package doktor

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Language represents the primary language detected in the project.
type Language string

const (
	LangGo      Language = "Go"
	LangNode    Language = "Node.js"
	LangJava    Language = "Java"
	LangPython  Language = "Python"
	LangRuby    Language = "Ruby"
	LangRust    Language = "Rust"
	LangUnknown Language = "Unknown"
)

// ProjectInfo holds everything ork doktor discovers about a project directory.
type ProjectInfo struct {
	Dir            string
	HasDockerfile  bool
	DockerfilePath string
	GitCommit      string // short SHA, empty when not a git repo
	Language       Language
	LangMarker     string   // the file that triggered language detection
	Port           string   // from PORT in .env, or language default
	EnvVars        []EnvVar // all parsed .env variables
	Secrets        []EnvVar // IsCfg == false
	Config         []EnvVar // IsCfg == true
	HasFrontend    bool
	AppName        string // derived from directory name
	HasSMTP        bool   // if env has smtp vars
	HasSlack       bool   // if env has slack vars
	License        string // project license read from regular license files
	HasCompose     bool   // docker-compose.yaml found
	UseCompose     bool   // user choice to use compose file
	ComposePath    string // path to docker-compose.yaml
}

// Detect scans a project directory and returns a populated ProjectInfo
// describing everything ork doktor can infer about the application.
//
// Detection includes:
//   - Dockerfile / Containerfile presence
//   - Git commit (short SHA)
//   - Primary programming language and marker file
//   - .env parsing, including config vs secret variables
//   - SMTP/Slack environment variable hints
//   - Application port (from .env or language defaults)
//   - Frontend detection via static dirs or JS frameworks
//   - License detection from common license files
//   - docker-compose.yaml discovery
//
// Missing .env files are not treated as errors. The function returns an error
// only when parsing .env fails or when filesystem access is not possible.
func Detect(dir string) (*ProjectInfo, error) {
	info := &ProjectInfo{Dir: dir}
	info.AppName = filepath.Base(dir)

	// Dockerfile Detection
	dockerfileNames := []string{"Dockerfile", "Containerfile"}
	for _, name := range dockerfileNames {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			info.HasDockerfile = true
			info.DockerfilePath = path
			break
		}
	}

	// Git commit
	info.GitCommit = shortGitCommit(dir)

	// Language detection
	info.Language, info.LangMarker = detectLanguage(dir)

	// Parse .env if present
	envPath := filepath.Join(dir, ".env")
	if fileExists(envPath) {
		vars, err := ParseEnvFile(envPath)
		if err != nil {
			return nil, err
		}
		info.EnvVars = vars
		info.Secrets, info.Config = SplitEnvVars(vars)
		info.HasSMTP = HasSMTP(vars)
		info.HasSlack = HasSlack(vars)
	}

	// Port detection
	info.Port = detectPort(info.EnvVars, info.Language)

	// Frontend detection
	info.HasFrontend = detectFrontend(dir, info.Language)

	// License detection
	info.License = DetectLicense(dir)

	// Compose detection
	if p := DetectComposeFile(dir); p != "" {
		info.HasCompose = true
		info.ComposePath = p
	}

	return info, nil
}

// DetectLicense scans the project directory for common license files
// (LICENSE, LICENSE.txt, LICENSE.md, COPYING, etc). It returns a normalized
// SPDX-style license name based on the first line of the file. If no license
// file is found or cannot be read, an empty string is returned.
func DetectLicense(dir string) string {
	candidates := []string{
		"LICENSE", "LICENSE.txt", "LICENSE.md",
		"COPYING", "COPYING.txt",
	}

	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			f, err := os.Open(path)
			if err != nil {
				return ""
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			if scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				return normalizeLicenseName(line)
			}
		}
	}

	return ""
}

// detectLanguage inspects well-known language marker files (go.mod,
// package.json, pom.xml, requirements.txt, Gemfile, Cargo.toml) to determine
// the primary programming language of the project. It returns both the detected
// language and the marker file that triggered detection. If no marker is found,
// LangUnknown is returned.
func detectLanguage(dir string) (Language, string) {
	checks := []struct {
		file string
		lang Language
	}{
		{"go.mod", LangGo},
		{"package.json", LangNode},
		{"pom.xml", LangJava},
		{"requirements.txt", LangPython},
		{"Gemfile", LangRuby},
		{"Cargo.toml", LangRust},
	}
	for _, c := range checks {
		if fileExists(filepath.Join(dir, c.file)) {
			return c.lang, c.file
		}
	}
	return LangUnknown, ""
}

// detectPort determines the application's port. It first checks for a PORT
// variable in the parsed .env file. If not present, it falls back to
// language-specific defaults (e.g., Go: 8080, Node: 3000, Python: 8000).
func detectPort(vars []EnvVar, lang Language) string {
	for _, v := range vars {
		if v.Key == "PORT" {
			return v.Value
		}
	}
	// Language defaults when PORT not in .env.
	switch lang {
	case LangGo:
		return "8080"
	case LangNode:
		return "3000"
	case LangJava:
		return "8080"
	case LangPython:
		return "8000"
	case LangRuby:
		return "3000"
	case LangRust:
		return "8080"
	default:
		return "8080"
	}
}

// detectFrontend reports whether the project appears to contain a frontend.
// It checks for common static build directories (build/, dist/, public/) and,
// for Node.js projects, scans package.json for known frontend frameworks such
// as React, Vue, Angular, Next.js, Nuxt, or Svelte.
func detectFrontend(dir string, lang Language) bool {
	// Static build directories suggest a frontend is present.
	for _, d := range []string{"build", "dist", "public"} {
		if dirExists(filepath.Join(dir, d)) {
			return true
		}
	}
	// package.json with a known framework dependency.
	if lang == LangNode && hasFrontendFramework(filepath.Join(dir, "package.json")) {
		return true
	}
	return false
}

// hasFrontendFramework inspects package.json and returns true if it contains
// dependencies for well-known frontend frameworks (react, vue, angular, next,
// nuxt, svelte). It returns false if the file cannot be read or no match is found.
func hasFrontendFramework(pkgJSON string) bool {
	data, err := os.ReadFile(pkgJSON)
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	for _, fw := range []string{"react", "vue", "angular", "next", "nuxt", "svelte"} {
		if strings.Contains(content, `"`+fw) || strings.Contains(content, `"`+fw+`"`) {
			return true
		}
	}
	return false
}

// shortGitCommit returns the short (7-character) Git commit SHA for the project
// directory. It reads .git/HEAD and resolves symbolic refs when necessary.
// If the directory is not a Git repository or the commit cannot be determined,
// an empty string is returned.
func shortGitCommit(dir string) string {
	headPath := filepath.Join(dir, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(string(data))
	if strings.HasPrefix(ref, "ref: ") {
		refFile := filepath.Join(dir, ".git", strings.TrimPrefix(ref, "ref: "))
		commitData, err := os.ReadFile(refFile)
		if err != nil {
			return ""
		}
		commit := strings.TrimSpace(string(commitData))
		if len(commit) >= 7 {
			return commit[:7]
		}
		return commit
	}
	// Detached HEAD — ref itself is the SHA.
	if len(ref) >= 7 {
		return ref[:7]
	}
	return ref
}

// normalizeLicenseName attempts to map the first line of a license file to a
// standard SPDX identifier (MIT, Apache-2.0, GPL, BSD, MPL). If no known
// pattern matches, the raw line (trimmed) is returned.
func normalizeLicenseName(line string) string {
	line = strings.ToLower(line)

	switch {
	case strings.Contains(line, "mit"):
		return "MIT"
	case strings.Contains(line, "apache"):
		return "Apache-2.0"
	case strings.Contains(line, "gpl"):
		return "GPL"
	case strings.Contains(line, "bsd"):
		return "BSD"
	case strings.Contains(line, "mozilla"):
		return "MPL"
	}

	return strings.TrimSpace(line)
}

// fileExists reports whether the given path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirExists reports whether the given path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
