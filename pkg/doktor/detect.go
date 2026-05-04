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
	Dir           string
	HasDockerfile bool
	GitCommit     string // short SHA, empty when not a git repo
	Language      Language
	LangMarker    string   // the file that triggered language detection
	Port          string   // from PORT in .env, or language default
	EnvVars       []EnvVar // all parsed .env variables
	Secrets       []EnvVar // IsCfg == false
	Config        []EnvVar // IsCfg == true
	HasFrontend   bool
	AppName       string // derived from directory name
	HasSMTP       bool   // if env has smtp vars
	HasSlack      bool   // if env has slack vars
	License       string // project license read from regular license files
}

// Detect examines dir and returns a ProjectInfo. Missing .env is not an error.
func Detect(dir string) (*ProjectInfo, error) {
	info := &ProjectInfo{Dir: dir}
	info.AppName = filepath.Base(dir)

	info.HasDockerfile = fileExists(filepath.Join(dir, "Dockerfile"))

	info.GitCommit = shortGitCommit(dir)

	info.Language, info.LangMarker = detectLanguage(dir)

	// Parse .env if present.
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

	info.Port = detectPort(info.EnvVars, info.Language)
	info.HasFrontend = detectFrontend(dir, info.Language)
	info.License = DetectLicense(dir)

	return info, nil
}

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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
