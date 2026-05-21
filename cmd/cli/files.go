package cli

import "os"

// defaultFilePaths returns the default katalog file if one exists in the
// current directory and no -f flag was provided. Tries katalog.yaml first,
// then komposer.yaml — the same precedence as Docker's Dockerfile / compose.yaml.
func defaultFilePaths() []string {
	for _, name := range []string{"katalog.yaml", "komposer.yaml"} {
		if _, err := os.Stat(name); err == nil {
			return []string{name}
		}
	}
	return nil
}

const errNoKatalog = "no katalog.yaml or komposer.yaml found in current directory\n" +
	"pass -f <file> or create one with ork init"
