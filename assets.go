// Package restored carries the data files that ship inside the binary: the two JSON
// Schema documents, the bundled recipes, and the hint catalog.
//
// It exists at the repository root because go:embed cannot reach outside its own
// package directory, and because there must be exactly one copy of each of these
// files. A copy under internal/ would drift from the one CI validates.
package restored

import "embed"

// Schemas holds schema/recipe.schema.json and schema/compose-safety.schema.json.
//
//go:embed schema/recipe.schema.json schema/compose-safety.schema.json
var Schemas embed.FS

// Recipes holds the bundled recipe directories, one per application.
//
//go:embed all:recipes
var Recipes embed.FS

// Hints holds docs/hints.yaml, the error pattern catalog.
//
//go:embed docs/hints.yaml
var Hints embed.FS
