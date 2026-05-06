package doctor

import (
	"bufio"
	"os"
	"strings"
)

const configMapMarker = "ork:cfg"

// EnvVar is a single variable parsed from a .env file.
type EnvVar struct {
	Key   string
	Value string
	IsCfg bool // true when line carries "# ork:cfg"
}

// ParseEnvFile reads a .env file and returns all non-blank, non-comment lines.
// Variables tagged with "# ork:cfg" on the same line have IsCfg = true;
// all others are treated as secrets.
func ParseEnvFile(path string) ([]EnvVar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var vars []EnvVar
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Strip inline comment to detect ork:cfg, then re-parse key=value.
		isCfg := false
		commentIdx := strings.Index(line, "#")
		if commentIdx >= 0 {
			comment := strings.TrimSpace(line[commentIdx:])
			if strings.Contains(comment, configMapMarker) {
				isCfg = true
			}
			line = strings.TrimSpace(line[:commentIdx])
		}

		if line == "" {
			continue
		}

		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		value := strings.TrimSpace(line[eqIdx+1:])
		// Strip surrounding quotes if present.
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		if key == "" {
			continue
		}
		vars = append(vars, EnvVar{Key: key, Value: value, IsCfg: isCfg})
	}
	return vars, scanner.Err()
}

// SplitEnvVars partitions parsed variables into secrets and config vars.
func SplitEnvVars(vars []EnvVar) (secrets, config []EnvVar) {
	for _, v := range vars {
		if v.IsCfg {
			config = append(config, v)
		} else {
			secrets = append(secrets, v)
		}
	}
	return
}

// HasSMTP returns true if any variable starts with "SMTP_".
func HasSMTP(vars []EnvVar) bool {
	for _, v := range vars {
		if strings.HasPrefix(strings.ToUpper(v.Key), "SMTP_") {
			return true
		}
	}
	return false
}

// HasSlack returns true if any variable starts with "SLACK_".
func HasSlack(vars []EnvVar) bool {
	for _, v := range vars {
		if strings.HasPrefix(strings.ToUpper(v.Key), "SLACK_") {
			return true
		}
	}
	return false
}

// GetEnvValue returns the value of an ENV var or false if not exists
func GetEnvValue(vars []EnvVar, key string) (string, bool) {
	key = strings.ToUpper(key)
	for _, v := range vars {
		if strings.ToUpper(v.Key) == key {
			return v.Value, true
		}
	}
	return "", false
}
