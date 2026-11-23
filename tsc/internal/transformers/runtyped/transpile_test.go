package runtyped_test

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/transpile"
)

// Single-file transpilation has no project and a file system that panics on
// any lookup outside the input file, so tsconfig.json must not be consulted.
func TestTranspile_ReflectsWithoutTsconfigLookup(t *testing.T) {
	t.Parallel()
	source := "export interface Shape { c: number; }\n"

	module := transpile.TranspileModule(t.Context(), source, transpile.Options{})
	assertContains(t, module.OutputText, "__ΩShape")

	declaration := transpile.TranspileDeclaration(t.Context(), source, transpile.Options{})
	assertContains(t, declaration.OutputText, "__ΩShape")
}
