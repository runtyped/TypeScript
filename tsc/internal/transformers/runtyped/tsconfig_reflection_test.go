package runtyped_test

import (
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/testutil/harnessutil"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

// compileProject compiles every .ts file in files as a program root. Other files
// (tsconfig.json and friends) are only placed on the virtual file system, where
// the transformer can find them. configFilePath mirrors `tsc -p <path>`; an
// empty value mirrors `tsc file.ts`, where the nearest tsconfig.json is used.
func compileProject(t *testing.T, files map[string]string, configFilePath string, declaration bool) *harnessutil.CompilationResult {
	t.Helper()
	var inputFiles []*harnessutil.TestFile
	var fileNames []tspath.RootedFilePath
	for name, content := range files {
		inputFiles = append(inputFiles, &harnessutil.TestFile{UnitName: name, Content: content})
		if strings.HasSuffix(name, ".ts") {
			fileNames = append(fileNames, tspath.RootedFilePathFromNormalized("/"+name))
		}
	}
	options := &core.CompilerOptions{
		Module:           core.ModuleKindCommonJS,
		ModuleResolution: core.ModuleResolutionKindNode10,
		Target:           core.ScriptTargetES2020,
	}
	if declaration {
		options.Declaration = core.TSTrue
	}
	if configFilePath != "" {
		options.ConfigFilePath = tspath.RootedFilePathFromNormalized(configFilePath)
	}
	return harnessutil.CompileFiles(t,
		inputFiles,
		nil,
		harnessutil.TestConfiguration{},
		&tsoptions.ParsedCommandLine{
			ParsedConfig: &tsoptions.ParsedOptions{
				CompilerOptions: options,
				FileNames:       fileNames,
			},
		},
		"/",
		nil,
	)
}

func jsOutput(t *testing.T, result *harnessutil.CompilationResult, name string) string {
	t.Helper()
	file := result.JS.GetOrZero(tspath.RootedFilePathFromNormalized(name))
	if file == nil {
		t.Fatalf("no output for %s", name)
	}
	return file.Content
}

func dtsOutput(t *testing.T, result *harnessutil.CompilationResult, name string) string {
	t.Helper()
	file := result.DTS.GetOrZero(tspath.RootedFilePathFromNormalized(name))
	if file == nil {
		t.Fatalf("no declaration output for %s", name)
	}
	return file.Content
}

// suite1 is a port of @runtyped/type-compiler's tests/setup/suite1 fixture.
var suite1 = map[string]string{
	"suite1/app.ts": `import { WithTypesBackend } from './backend/file3';
import { WithTypes } from './file1';
import { WithoutTypesFrontend } from './frontend/file2';
console.log(WithTypes, WithoutTypesFrontend, WithTypesBackend);
`,
	"suite1/file1.ts":               "export class WithTypes { name!: string; }\n",
	"suite1/frontend/file2.ts":      "export class WithoutTypesFrontend { name!: string; }\n",
	"suite1/backend/file3.ts":       "export class WithTypesBackend { name!: string; }\n",
	"suite1/tsconfig.json":          `{ "compilerOptions": { "strict": true }, "reflection": true, "files": ["app.ts"] }`,
	"suite1/tsconfig.no-types.json": `{ "extends": "./tsconfig.json", "reflection": ["**/file1.ts"] }`,
	"suite1/frontend/tsconfig.json": `{ "extends": "../tsconfig.json", "reflection": false, "files": ["file2.ts"] }`,
	"suite1/backend/tsconfig.json":  `{ "extends": "../tsconfig.json", "files": ["file3.ts"] }`,
}

func TestTsconfigReflection_Suite1(t *testing.T) {
	t.Parallel()

	t.Run("BaseDefault", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, suite1, "/suite1/tsconfig.json", false)
		assertContains(t, jsOutput(t, result, "/suite1/file1.js"), "__type")
		assertContains(t, jsOutput(t, result, "/suite1/backend/file3.js"), "__type")
		// The project's own tsconfig applies to every file it compiles, so
		// frontend/tsconfig.json is not consulted.
		assertContains(t, jsOutput(t, result, "/suite1/frontend/file2.js"), "__type")
	})

	t.Run("BaseNoTypes", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, suite1, "/suite1/tsconfig.no-types.json", false)
		assertContains(t, jsOutput(t, result, "/suite1/file1.js"), "__type")
		assertNotContains(t, jsOutput(t, result, "/suite1/backend/file3.js"), "__type")
		assertNotContains(t, jsOutput(t, result, "/suite1/frontend/file2.js"), "__type")
	})

	t.Run("Frontend", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, suite1, "/suite1/frontend/tsconfig.json", false)
		assertNotContains(t, jsOutput(t, result, "/suite1/frontend/file2.js"), "__type")
	})

	t.Run("Backend", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, suite1, "/suite1/backend/tsconfig.json", false)
		assertContains(t, jsOutput(t, result, "/suite1/backend/file3.js"), "__type")
	})

	t.Run("NearestTsconfigWithoutProject", func(t *testing.T) {
		t.Parallel()
		// Without -p, each file uses the nearest tsconfig.json above it.
		result := compileProject(t, suite1, "", false)
		assertContains(t, jsOutput(t, result, "/suite1/file1.js"), "__type")
		assertContains(t, jsOutput(t, result, "/suite1/backend/file3.js"), "__type")
		assertNotContains(t, jsOutput(t, result, "/suite1/frontend/file2.js"), "__type")
	})
}

func TestTsconfigReflection_Modes(t *testing.T) {
	t.Parallel()
	source := `export class Plain { a!: string; }
/** @reflection */
export class Tagged { b!: string; }
export interface Shape { c: number; }
`

	t.Run("NoReflectionKeyReflectsEverything", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, map[string]string{
			"app.ts":        source,
			"tsconfig.json": `{ "compilerOptions": {} }`,
		}, "/tsconfig.json", false)
		js := jsOutput(t, result, "/app.js")
		assertContains(t, js, "__ΩShape")
		assertContains(t, js, "__type")
	})

	t.Run("NoTsconfigReflectsEverything", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, map[string]string{"app.ts": source}, "", false)
		js := jsOutput(t, result, "/app.js")
		assertContains(t, js, "__ΩShape")
		assertContains(t, js, "__type")
	})

	for _, value := range []string{"false", `"never"`} {
		t.Run("Disabled_"+value, func(t *testing.T) {
			t.Parallel()
			result := compileProject(t, map[string]string{
				"app.ts":        source,
				"tsconfig.json": `{ "reflection": ` + value + ` }`,
			}, "/tsconfig.json", false)
			js := jsOutput(t, result, "/app.js")
			assertNotContains(t, js, "__Ω")
			assertNotContains(t, js, "__type")
		})
	}

	t.Run("Explicit", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, map[string]string{
			"app.ts":        source,
			"tsconfig.json": `{ "reflection": "explicit" }`,
		}, "/tsconfig.json", false)
		js := jsOutput(t, result, "/app.js")
		assertNotContains(t, js, "__ΩShape")
		assertContains(t, js, "Tagged.__type")
		if strings.Count(js, "__type") != 1 {
			t.Errorf("expected exactly one reflected class (Tagged), got:\n%s", js)
		}
	})

	t.Run("PathPatterns", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, map[string]string{
			"src/model.ts":  source,
			"lib/util.ts":   source,
			"tsconfig.json": `{ "reflection": ["src/**/*.ts"] }`,
		}, "/tsconfig.json", false)
		assertContains(t, jsOutput(t, result, "/src/model.js"), "__ΩShape")
		assertNotContains(t, jsOutput(t, result, "/lib/util.js"), "__Ω")
	})

	t.Run("ExcludeViaDeepkitCompilerOptions", func(t *testing.T) {
		t.Parallel()
		result := compileProject(t, map[string]string{
			"src/model.ts":  source,
			"src/skip.ts":   source,
			"tsconfig.json": `{ "reflection": true, "deepkitCompilerOptions": { "exclude": ["src/skip.ts"] } }`,
		}, "/tsconfig.json", false)
		assertContains(t, jsOutput(t, result, "/src/model.js"), "__ΩShape")
		assertNotContains(t, jsOutput(t, result, "/src/skip.js"), "__Ω")
	})
}

// A reflected file must not import __Ω symbols from a file that is not
// reflected, since that file never defines them.
func TestTsconfigReflection_CrossFile(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"src/app.ts": `import { Shape } from '../lib/shape';
export interface User { shape: Shape; }
`,
		"lib/shape.ts":  "export interface Shape { c: number; }\n",
		"tsconfig.json": `{ "reflection": ["src/**/*.ts"] }`,
	}
	result := compileProject(t, files, "/tsconfig.json", true)

	app := jsOutput(t, result, "/src/app.js")
	assertContains(t, app, "__ΩUser")
	assertNotContains(t, app, "__ΩShape")

	assertNotContains(t, jsOutput(t, result, "/lib/shape.js"), "__ΩShape")
	assertNotContains(t, dtsOutput(t, result, "/lib/shape.d.ts"), "__ΩShape")
	assertContains(t, dtsOutput(t, result, "/src/app.d.ts"), "__ΩUser")
}
