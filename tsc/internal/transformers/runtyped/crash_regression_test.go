package runtyped_test

import (
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/testutil/harnessutil"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

// compileFilesJS compiles the given files (keyed by rooted path) as ES modules
// and returns every emitted JS file keyed by its rooted path.
func compileFilesJS(t *testing.T, files map[string]string) map[string]string {
	t.Helper()
	var inputFiles []*harnessutil.TestFile
	for name, content := range files {
		inputFiles = append(inputFiles, &harnessutil.TestFile{UnitName: name, Content: content})
	}
	result := harnessutil.CompileFiles(t,
		inputFiles,
		nil,
		harnessutil.TestConfiguration{},
		tsoptions.NewParsedCommandLine(
			&core.CompilerOptions{
				Module: core.ModuleKindESNext,
				Target: core.ScriptTargetESNext,
			},
			nil,
			nil,
			"/",
			tspath.CaseSensitive,
		),
		"/",
		nil,
	)
	outputs := make(map[string]string)
	for key, value := range result.JS.Entries() {
		outputs[key.AsString()] = value.Content
	}
	return outputs
}

// TestInferInsideMappedTypeInExtendsClause covers an `infer` declaration that is
// reached from inside a co-routine (here, the mapped type) while its conditional
// frame lives in an enclosing op list. Reduced from the upstream conformance test
// inferConditionalConstraintMappedMember.ts, which used to panic with
// "slice bounds out of range" in pushOpAtFrame.
func TestInferInsideMappedTypeInExtendsClause(t *testing.T) {
	t.Parallel()
	js := compileSingleFileJS(t, `
type KeysWithoutStringIndex<T> =
    { [K in keyof T]: string extends K ? never : K } extends { [_ in keyof T]: infer U }
    ? U
    : never;
`)
	if !strings.Contains(js, "__ΩKeysWithoutStringIndex") {
		t.Fatalf("expected __ΩKeysWithoutStringIndex in output, got:\n%s", js)
	}
}

// TestInferInNestedConditionalInExtendsClause is reduced from the upstream
// conformance test inferTypes1.ts (type T63), which used to panic in pushOpAtFrame.
func TestInferInNestedConditionalInExtendsClause(t *testing.T) {
	t.Parallel()
	js := compileSingleFileJS(t, `
type T63<T> = T extends ((infer A) extends infer B ? infer C : infer D) ? string : number;
`)
	if !strings.Contains(js, "__ΩT63") {
		t.Fatalf("expected __ΩT63 in output, got:\n%s", js)
	}
}

// TestArrowFunctionReceiveTypeResetsOmega checks that an arrow function assigned
// to a variable resets its Ω slot like function declarations do. The arrow node
// is rewritten by the transformer, so its parent must come from the parse tree.
func TestArrowFunctionReceiveTypeResetsOmega(t *testing.T) {
	t.Parallel()
	js := compileSingleFileJS(t, `
const fn = <T>(type?: ReceiveType<T>) => { return type; };
`)
	if !strings.Contains(js, "type = fn.Ω") {
		t.Fatalf("expected the ReceiveType default to read fn.Ω, got:\n%s", js)
	}
	if !strings.Contains(js, "fn.Ω = undefined") {
		t.Fatalf("expected fn.Ω to be reset at the start of the body, got:\n%s", js)
	}
}

// TestReExportWithStringLiteralNames covers ES2022 arbitrary module namespace
// identifiers in re-exports, whose names are string literals rather than
// identifiers. Reduced from the upstream conformance tests
// arbitraryModuleNamespaceIdentifiers_*.ts, which used to panic in
// visitExportDeclaration.
func TestReExportWithStringLiteralNames(t *testing.T) {
	t.Parallel()
	outputs := compileFilesJS(t, map[string]string{
		"/types.ts": `
export interface Foo { a: string }
export const value = 1;
`,
		"/app.ts": `
export { "value" as "my value", Foo as "my Foo" } from "./types";
export { "my value" as renamed } from "./app2";
`,
		"/app2.ts": `
export { value as "my value" } from "./types";
`,
	})
	appJS := outputs["/app.js"]
	if !strings.Contains(appJS, `"my value"`) {
		t.Fatalf("expected the string-literal re-export to be preserved, got:\n%s", appJS)
	}
}

// TestQualifiedTypeNameOnDefaultImport covers a qualified type reference whose
// left side is a default import (`ajvModule.Ajv`), which used to crash.
func TestQualifiedTypeNameOnDefaultImport(t *testing.T) {
	t.Parallel()
	const app = `
import ajvModule from 'ajv';

export const errorsToString = (ajv: ajvModule.Ajv): string => {
  return ajv.errors?.map(e => e.message).join(', ') || '';
};
`
	t.Run("unresolved module", func(t *testing.T) {
		t.Parallel()
		outputs := compileFilesJS(t, map[string]string{"/app.ts": app})
		if !strings.Contains(outputs["/app.js"], "errorsToString") {
			t.Fatalf("expected errorsToString in output, got:\n%s", outputs["/app.js"])
		}
	})
	t.Run("resolved module", func(t *testing.T) {
		t.Parallel()
		outputs := compileFilesJS(t, map[string]string{
			"/app.ts":                        app,
			"/node_modules/ajv/package.json": `{ "name": "ajv", "types": "index.d.ts" }`,
			"/node_modules/ajv/index.d.ts": `
declare var ajv: { (): ajv.Ajv };
declare namespace ajv {
  interface ErrorObject { message?: string }
  interface Ajv { errors?: Array<ErrorObject> | null }
}
export = ajv;
`,
		})
		if !strings.Contains(outputs["/app.js"], "errorsToString") {
			t.Fatalf("expected errorsToString in output, got:\n%s", outputs["/app.js"])
		}
	})
}
