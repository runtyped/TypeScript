package runtyped_test

import (
	"maps"
	"testing"
)

// TestTypeOnlyImportOfReflectedType covers runtyped/runtyped#11. By default a
// type-only import of an interface or type alias is reflected as any, like
// @runtyped/type-compiler does, so its module is never loaded at runtime. With
// `"reflectTypeOnlyImports": true` in tsconfig.json it imports __ΩX like a
// value import does, when the module is known to define __ΩX.
func TestTypeOnlyImportOfReflectedType(t *testing.T) {
	t.Parallel()

	const app = "export type Event = M | { method: 'tick' };\n"
	const omegaRef = "() => __ΩM"
	const omegaImport = "import { __ΩM } from "
	const anyRef = `"M", 'tick'`
	const enabled = `{"reflectTypeOnlyImports": true}`

	sources := []struct {
		name   string
		files  map[string]string
		module string
	}{
		{
			name: "ImportTypeFromReflectedTS",
			files: map[string]string{
				"app.ts": "import type { M } from './m';\n" + app,
				"m.ts":   "export interface M { method: 'message' }\n",
			},
			module: "'./m'",
		},
		{
			name: "InlineTypeSpecifierFromReflectedTS",
			files: map[string]string{
				"app.ts": "import { type M } from './m';\n" + app,
				"m.ts":   "export type M = { method: 'message' };\n",
			},
			module: "'./m'",
		},
		{
			name: "ImportTypeFromDTSWithOmega",
			files: map[string]string{
				"app.ts": "import type { M } from './m';\n" + app,
				"m.d.ts": "export interface M { method: 'message' }\nexport declare type __ΩM = any[];\n",
			},
			module: "'./m'",
		},
		{
			name: "ImportTypeFromPackageSpecifier",
			files: map[string]string{
				"app.ts": "import type { M } from '@repro/events';\n" + app,
				"node_modules/@repro/events/package.json": `{"name":"@repro/events","types":"index.d.ts"}`,
				"node_modules/@repro/events/index.d.ts":   "export type { M, __ΩM } from './types.js';\n",
				"node_modules/@repro/events/types.d.ts":   "export interface M { method: 'message' }\nexport declare type __ΩM = any[];\n",
			},
			module: "'@repro/events'",
		},
	}

	with := func(files map[string]string, extra map[string]string) map[string]string {
		out := maps.Clone(files)
		maps.Copy(out, extra)
		return out
	}

	for _, src := range sources {
		t.Run(src.name, func(t *testing.T) {
			t.Parallel()

			t.Run("Enabled", func(t *testing.T) {
				t.Parallel()
				appJS := compileFilesJS(t, with(src.files, map[string]string{"tsconfig.json": enabled}))["/app.js"]
				t.Logf("app.js:\n%s", appJS)
				assertContains(t, appJS, omegaRef)
				assertContains(t, appJS, omegaImport+src.module)
				assertNotContains(t, appJS, anyRef)
			})

			// Without a tsconfig.json, and with the option absent or false,
			// the type stays any and the module is not imported.
			for name, tsconfig := range map[string]string{
				"NoTsconfig":     "",
				"OptionAbsent":   `{}`,
				"OptionDisabled": `{"reflectTypeOnlyImports": false}`,
			} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					files := src.files
					if tsconfig != "" {
						files = with(files, map[string]string{"tsconfig.json": tsconfig})
					}
					appJS := compileFilesJS(t, files)["/app.js"]
					t.Logf("app.js:\n%s", appJS)
					assertContains(t, appJS, anyRef)
					assertNotContains(t, appJS, "__ΩM")
				})
			}
		})
	}

	// A value import is reflected whatever the option says.
	t.Run("ValueImportUnaffected", func(t *testing.T) {
		t.Parallel()
		appJS := compileFilesJS(t, map[string]string{
			"app.ts":        "import { M } from './m';\n" + app,
			"m.ts":          "export interface M { method: 'message' }\n",
			"tsconfig.json": `{"reflectTypeOnlyImports": false}`,
		})["/app.js"]
		assertContains(t, appJS, omegaImport+"'./m'")
	})

	// Without __ΩM in the .d.ts the module may not be built with runtime
	// types, so even with the option the type stays any.
	t.Run("EnabledDTSWithoutOmega", func(t *testing.T) {
		t.Parallel()
		appJS := compileFilesJS(t, map[string]string{
			"app.ts":        "import type { M } from './m';\n" + app,
			"m.d.ts":        "export interface M { method: 'message' }\n",
			"tsconfig.json": enabled,
		})["/app.js"]
		t.Logf("app.js:\n%s", appJS)
		assertNotContains(t, appJS, "__ΩM")
	})

	// The option is inherited through extends, and a config overrides the
	// ones it extends.
	t.Run("Extends", func(t *testing.T) {
		t.Parallel()
		base := map[string]string{
			"src/app.ts": "import type { M } from './m';\n" + app,
			"src/m.ts":   "export interface M { method: 'message' }\n",
			"base.json":  enabled,
		}
		inherited := compileFilesJS(t, with(base, map[string]string{
			"src/tsconfig.json": `{"extends": "../base.json"}`,
		}))["/src/app.js"]
		assertContains(t, inherited, omegaImport+"'./m'")

		overridden := compileFilesJS(t, with(base, map[string]string{
			"src/tsconfig.json": `{"extends": "../base.json", "reflectTypeOnlyImports": false}`,
		}))["/src/app.js"]
		assertNotContains(t, overridden, "__ΩM")
	})
}
