// pkg/generate/helper.go
package generate

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func parseKatalog(data []byte) (*generateKatalog, error) {
	var kat generateKatalog
	if err := yaml.Unmarshal(data, &kat); err != nil {
		return nil, fmt.Errorf("parsing katalog: %w", err)
	}
	if len(kat.Spec.CRDs) == 0 {
		return nil, fmt.Errorf("katalog has no CRDs defined")
	}
	return &kat, nil
}
