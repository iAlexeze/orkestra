//go:build !runtime

package examples

import "embed"

// FS contains all example packs baked into the CLI at build time.
// Imported only by CLI commands — excluded from the runtime binary.
//
// developer
//
//go:embed beginner intermediate advanced security resilience use-cases registry-guide from-controller-runtime ecosystem-composition
var FS embed.FS
