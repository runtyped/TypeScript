package runtyped_test

import (
	"strings"
	"testing"
)

// Regression tests for differences between this transformer and
// @runtyped/type-compiler (the TypeScript 6 transformer) found by compiling
// the runtyped repository with both. Each expectation was checked against
// the output of @runtyped/type-compiler for the same source.

type parityCase struct {
	name string
	// files to compile; the case's assertions apply to the output of app.ts.
	files    map[string]string
	contains []string
	absent   []string
}

func TestTS6Parity(t *testing.T) {
	t.Parallel()
	cases := []parityCase{
		{
			name: "ReflectionNeverOnDeclaration",
			files: map[string]string{"app.ts": `/** @reflection never */
export interface Hidden { a: string }
export interface Uses { h: Hidden }
`},
			// The file-top comment also tags the file itself, so nothing is reflected.
			absent: []string{"__ΩHidden", "__ΩUses"},
		},
		{
			name: "ReflectionNeverOnNonFirstDeclaration",
			files: map[string]string{"app.ts": `export const x = 1;
/** @reflection never */
export interface Hidden { a: string }
export interface Uses { h: Hidden }
`},
			contains: []string{`["Hidden", "h", "Uses", "P\"w!4\"Mw#y"]`},
			absent:   []string{"const __ΩHidden"},
		},
		{
			name: "ReflectionNoOnProperty",
			files: map[string]string{"app.ts": `export class P {
    /** @reflection no */
    secret!: string;
    name!: string;
}
`},
			contains: []string{`["name", "P", "&3!5w\""]`},
		},
		{
			name: "ReflectionNoOnMethod",
			files: map[string]string{"app.ts": `export interface I {
    /** @reflection no */
    fn(a: string): void;
    b: number;
}
`},
			contains: []string{`["b", "I", "P\"'4!Mw\"y"]`},
		},
		{
			name: "ImportedNeverClassIsStillReferenced",
			files: map[string]string{
				"app.ts":  "import { Hidden } from './decl';\nexport interface Use { h: Hidden }\n",
				"decl.ts": "export const x = 1;\n/** @reflection never */\nexport class Hidden { a!: string; }\n",
			},
			contains: []string{`[() => decl_1.Hidden, "h", "Use", "PP7!4\"Mw#y"]`},
		},
		{
			name:     "NestedFunctionTypeIsASibling",
			files:    map[string]string{"app.ts": "export function outer() {\n    function inner(a: string) { return a; }\n    return inner;\n}\n"},
			contains: []string{`inner.__type = ["a", "inner", "P&2!\"/\""];`},
			// Not wrapped in a block, which would scope inner to it.
			absent: []string{"{\n        function inner", "{ function inner"},
		},
		{
			name:     "LocalTypesStayInTheirScope",
			files:    map[string]string{"app.ts": "export function one() { interface S { a: string } return 1; }\nexport function two() { interface S { b: number } return 2; }\n"},
			contains: []string{`function one() { const __ΩS = ["a", "S", "P&4!Mw\"y"];`, `function two() { const __ΩS = ["b", "S", "P'4!Mw\"y"];`},
		},
		{
			name:     "DefiniteAssignmentIsNotOptional",
			files:    map[string]string{"app.ts": "export class D { a!: string; b?: string; }\n"},
			contains: []string{`["a", "b", "D", "&3!&3\"85w#"]`},
		},
		{
			name: "Descriptions",
			files: map[string]string{"app.ts": `/** @description an interface */
export interface Desc {
    /** @description a property */
    p: string;
    /** @description a method */
    m(/** @description a param */ x: number): void;
}
/** @description a class */
export class DescC { /** @description a field */ f: number = 1; }
/** @description an enum */
export enum DescE { A }
`},
			contains: []string{
				`["p", "a property", "x", "m", "a method", "an interface", "Desc", "P&4!?\"P'2#$1$?%M?&w'y"]`,
				`["f", function () { return 1; }, "DescC", "a class", "'3!>\"5w#?$"]`,
				`["A", "an enum", "DescE", "PC!B?\"w#"]`,
			},
		},
		{
			name:     "ParameterDefaultValue",
			files:    map[string]string{"app.ts": "export function isDebug(level: number = 1) { return level; }\n"},
			contains: []string{`["level", () => 1, "isDebug", "P'2!>\"\"/#"]`},
		},
		{
			name:     "IndexSignature",
			files:    map[string]string{"app.ts": "export type Dict = { [name: string]: number };\n"},
			contains: []string{`["Dict", "P&'LMw!y"]`},
		},
		{
			name:     "SelfReferenceIsZero",
			files:    map[string]string{"app.ts": "export interface Linked { next?: Linked; value: number }\n"},
			contains: []string{`[0, "next", "value", "Linked", "Pn!4\"8'4#Mw$y"]`},
		},
		{
			name: "ImportTypeClauseIsTypeOnly",
			files: map[string]string{
				"app.ts": "import type { S } from './s';\nexport interface U { s: S }\n",
				"s.ts":   "export class S { a!: string; }\n",
			},
			contains: []string{`["S", "s", "U", "P\"w!4\"Mw#y"]`},
		},
		{
			name: "ReceiveTypeInUnionParameter",
			files: map[string]string{
				"app.ts": "import { S } from './s';\nexport type ReceiveType<T> = { __meta?: T };\nexport function f<T>(a?: S | ReceiveType<T>) {}\n",
				"s.ts":   "export class S { a!: string; }\n",
			},
			contains: []string{`[() => __ΩReceiveType, "a", "f", "P\"o!\"2\"8\"/#"]`},
		},
		{
			name: "OuterTypeParameters",
			files: map[string]string{"app.ts": `export type ReceiveType<T> = { __meta?: T };
export function a<T>(type?: ReceiveType<T>) { return (data: T) => data; }
export function b<T>(items: T[]) { return (data: T) => data; }
export function c<T extends string>(x: number) { return (data: T) => data; }
export function d<T>(x: number) { return (data: T) => data; }
`},
			contains: []string{
				`[type, "data", "", "Pn!2\"\"/#"]`,
				`[() => items, "data", "", "PPdi!h!!Fqe!!!j2\"\"/#"]`,
				`["data", "", "P&2!\"/\""]`,
				`["data", "", "P!2!\"/\""]`,
			},
		},
		{
			name:     "OpsAbove127AreEncodedAsCodePoints",
			files:    map[string]string{"app.ts": bigUnion(120)},
			contains: []string{`\u0080`},
			absent:   []string{`�`},
		},
		{
			name: "DeclarationFirstMetAsReference",
			files: map[string]string{"app.ts": `export type ClassType<T = any> = new (...args: any[]) => T;
export function create<API>(api: API): ExtractClass<API> { return null as any; }
export type ExtractClass<T> = T extends ClassType<infer K> ? K : never;
`},
			contains: []string{"const __ΩExtractClass = "},
		},
		{
			name: "TypeOfInNeverFunctionIsLeftAlone",
			files: map[string]string{"app.ts": `declare function typeOf<T>(args?: any[], p?: any): any;
export const x = 1;
/** @reflection never */
export function q() { return typeOf<string>(); }
export function r() { return typeOf<string>(); }
`},
			contains: []string{"function q() { return typeOf(); }", `function r() { return typeOf([], ["&"]); }`},
		},
		{
			name: "ObjectLiteralMethods",
			files: map[string]string{"app.ts": `export const v = {
    alpha(s: string): boolean { return /a/.test(s); },
    default(): number { return 1; },
    pattern(type: string) { return (value: any) => value; },
};
`},
			contains: []string{
				`alpha: __assignType(function alpha(s) { return /a/.test(s); }, ["s", "alpha", "P&2!)/\""])`,
				"default() { return 1; }",
				// A method whose body was transformed is kept, like type-compiler.
				"pattern(type) { return __assignType((value) => value,",
			},
		},
		{
			name: "NumericAndComputedPropertyNames",
			files: map[string]string{"app.ts": `export interface Names { 3: boolean; [Symbol.iterator](): void; "quoted": string; }
`},
			contains: []string{`[3, () => Symbol.iterator, "quoted", "Names", "P)4!P$1\"&4#Mw$y"]`},
		},
		{
			name: "IntrinsicTypeAnnotation",
			files: map[string]string{"app.ts": `/** @intrinsic */
export type TypeAnnotation<T extends string, Options = never> = unknown;
`},
			contains: []string{`["T", "Options", "__meta", "TypeAnnotation", "b!!c\"PP!Pe$!e$\"GK4#8Mw$y"]`},
		},
		{
			name: "MethodOverloads",
			files: map[string]string{"app.ts": `export class R {
    get(a: string): void;
    get(a: number): void;
    get(a: any) {}
}
`},
			contains: []string{`["a", "get", "R", "P&2!$0\"5w#"]`},
		},
		{
			name:     "PropertyDefaultsKeepCallSiteTypes",
			files:    map[string]string{"app.ts": "export class M { m = new Map<string, number>(); }\n"},
			contains: []string{`["m", function () { return (Map.Ω = [["&"], ["'"]], new Map()); }, "M", "!3!>\"5w#"]`},
		},
		{
			name: "ReExportsOnlyExistingOmegaSymbols",
			files: map[string]string{
				"app.ts":    "export { FromDts } from './lib';\nexport { Hidden } from './decl';\nexport { Linked } from './linked';\n",
				"lib.d.ts":  "export interface FromDts { a: string }\n",
				"decl.ts":   "export const x = 1;\n/** @reflection never */\nexport interface Hidden { a: string }\n",
				"linked.ts": "export interface Linked { value: number }\n",
			},
			contains: []string{"__ΩLinked"},
			absent:   []string{"__ΩFromDts", "__ΩHidden"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			js := jsOutput(t, compileProject(t, tc.files, "", false), "/app.js")
			for _, want := range tc.contains {
				if !strings.Contains(js, want) {
					t.Errorf("expected output to contain %q\noutput:\n%s", want, js)
				}
			}
			for _, unwanted := range tc.absent {
				if strings.Contains(js, unwanted) {
					t.Errorf("expected output not to contain %q\noutput:\n%s", unwanted, js)
				}
			}
		})
	}
}

// bigUnion returns a type alias with a union of n string literals, whose
// bytecode needs stack indices well above 94 (encoded above code point 127).
func bigUnion(n int) string {
	members := make([]string, n)
	for i := range members {
		members[i] = "'a" + strings.Repeat("x", i%3) + string(rune('0'+i%10)) + string(rune('a'+i/10)) + "'"
	}
	return "export type Big = " + strings.Join(members, " | ") + ";\n"
}
