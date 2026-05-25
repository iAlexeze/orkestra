//go:build !runtime

package examples

import "embed"

// FS contains all example packs baked into the CLI at build time.
// Imported only by CLI commands — excluded from the runtime binary.
//
// developer
//
//go:embed beginner intermediate advanced security use-cases Makefile setup-kind.sh load.sh
var FS embed.FS
