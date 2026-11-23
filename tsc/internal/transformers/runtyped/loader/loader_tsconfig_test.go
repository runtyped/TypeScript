package loader_test

import (
	"os"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/transformers/runtyped/loader"
)

// Ports of loader.spec.ts's tsconfig-driven cases. The loader reads
// tsconfig.json from the real file system, so each test writes one to a
// temporary directory.

const tsconfigSource = `export interface Config { a: string; }
`

// writeTsconfig writes a tsconfig.json with the given content to a fresh
// temporary directory and returns that directory.
func writeTsconfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/tsconfig.json", []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func transformAt(t *testing.T, l *loader.Loader, code, path string) string {
	t.Helper()
	result, err := l.Transform(code, path)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	return result
}

func TestLoader_TsconfigWithReflection(t *testing.T) {
	t.Parallel()
	dir := writeTsconfig(t, `{ "reflection": true }`)
	l := loader.NewLoader(loader.LoaderOptions{TsConfig: dir + "/tsconfig.json"})
	result := transformAt(t, l, tsconfigSource, dir+"/app.ts")
	if !strings.Contains(result, "__ΩConfig") {
		t.Errorf("expected reflection, got:\n%s", result)
	}
}

func TestLoader_TsconfigReflectionDisabled(t *testing.T) {
	t.Parallel()
	dir := writeTsconfig(t, `{ "reflection": false }`)
	l := loader.NewLoader(loader.LoaderOptions{TsConfig: dir + "/tsconfig.json"})
	result := transformAt(t, l, tsconfigSource, dir+"/app.ts")
	if strings.Contains(result, "__Ω") {
		t.Errorf("expected no reflection, got:\n%s", result)
	}
}

// Unlike @runtyped/type-compiler, a tsconfig.json without a `reflection` field
// reflects everything: compiling with runtyped is the opt-in.
func TestLoader_TsconfigWithoutReflectionField(t *testing.T) {
	t.Parallel()
	dir := writeTsconfig(t, `{ "compilerOptions": {} }`)
	l := loader.NewLoader(loader.LoaderOptions{TsConfig: dir + "/tsconfig.json"})
	result := transformAt(t, l, tsconfigSource, dir+"/app.ts")
	if !strings.Contains(result, "__ΩConfig") {
		t.Errorf("expected reflection, got:\n%s", result)
	}
}

func TestLoader_OptionOverridesTsconfig(t *testing.T) {
	t.Parallel()
	dir := writeTsconfig(t, `{ "reflection": true }`)
	l := loader.NewLoader(loader.LoaderOptions{
		TsConfig:   dir + "/tsconfig.json",
		Reflection: loader.ReflectionNever,
	})
	result := transformAt(t, l, tsconfigSource, dir+"/app.ts")
	if strings.Contains(result, "__Ω") {
		t.Errorf("expected the option to disable reflection, got:\n%s", result)
	}
}

func TestLoader_NearestTsconfig(t *testing.T) {
	t.Parallel()
	dir := writeTsconfig(t, `{ "reflection": false }`)
	l := loader.NewLoader(loader.LoaderOptions{})
	result := transformAt(t, l, tsconfigSource, dir+"/app.ts")
	if strings.Contains(result, "__Ω") {
		t.Errorf("expected the nearest tsconfig.json to disable reflection, got:\n%s", result)
	}
}
