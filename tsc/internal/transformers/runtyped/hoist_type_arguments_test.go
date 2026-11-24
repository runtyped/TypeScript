package runtyped_test

import (
	"strings"
	"testing"
)

// receiveTypeFn declares a function taking its type argument through
// ReceiveType<T>, so that call sites pass it directly.
const receiveTypeFn = "function rt<T>(data?: any, type?: ReceiveType<T>): any { return type; }\n"

func TestHoistTypeArguments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		files    map[string]string
		contains []string
		absent   []string
	}{
		{
			name: "ModuleLevelTypeIsHoistedAndShared",
			files: map[string]string{"app.ts": receiveTypeFn + `interface User { id: number }
export function a(x: any) { return rt<User>(x); }
export function b(x: any) { return rt<User>(x); }`},
			contains: []string{
				"var __ΩΩ0;",
				`return rt(x, (__ΩΩ0 || (__ΩΩ0 = [() => __ΩUser, "n!"])));`,
			},
			absent: []string{"__ΩΩ1"},
		},
		{
			name:     "InlineTypeIsHoisted",
			files:    map[string]string{"app.ts": receiveTypeFn + `export function a(x: any) { return rt<{ id: number; tags: string[] }>(x); }`},
			contains: []string{"var __ΩΩ0;", "rt(x, (__ΩΩ0 || (__ΩΩ0 = ["},
		},
		{
			name:     "SelfContainedNamesAreHoisted",
			files:    map[string]string{"app.ts": receiveTypeFn + `export function a(x: any) { return rt<{ [K in 'a' | 'b']: K }>(x); }`},
			contains: []string{"rt(x, (__ΩΩ0 || (__ΩΩ0 = ["},
		},
		{
			name: "ImportedTypeIsHoisted",
			files: map[string]string{
				"app.ts":  receiveTypeFn + "import { Cfg } from './cfg';\nexport function a(x: any) { return rt<Cfg>(x); }",
				"cfg.ts":  "export interface Cfg { a: number }\n",
				"main.ts": "",
			},
			contains: []string{`(__ΩΩ0 || (__ΩΩ0 = [() => __ΩCfg, "n!"]))`},
		},
		{
			name:   "FunctionTypeParameterIsNotHoisted",
			files:  map[string]string{"app.ts": receiveTypeFn + `export function a<T>(x: any) { return rt<T>(x); }`},
			absent: []string{"__ΩΩ"},
		},
		{
			name:   "TypeParameterInsideInlineTypeIsNotHoisted",
			files:  map[string]string{"app.ts": receiveTypeFn + `export function a<T>(x: any) { return rt<{ value: T }>(x); }`},
			absent: []string{"__ΩΩ"},
		},
		{
			name: "ClassTypeParameterIsNotHoisted",
			files: map[string]string{"app.ts": receiveTypeFn + `export class Box<T> {
    check(x: any) { return rt<T[]>(x); }
}`},
			absent: []string{"__ΩΩ"},
		},
		{
			name: "LocalTypeIsNotHoisted",
			files: map[string]string{"app.ts": receiveTypeFn + `export function a(x: any) {
    interface Local { id: number }
    return rt<Local>(x);
}`},
			absent: []string{"__ΩΩ"},
		},
		{
			name: "ThisTypeIsNotHoisted",
			files: map[string]string{"app.ts": receiveTypeFn + `export class Node {
    check(x: any) { return rt<this>(x); }
}`},
			absent: []string{"__ΩΩ"},
		},
		{
			name:     "NamesDoNotCollideWithSourceIdentifiers",
			files:    map[string]string{"app.ts": receiveTypeFn + "const __ΩΩ0 = 1;\nexport function a(x: any) { return rt<string>(x) + __ΩΩ0; }"},
			contains: []string{"var __ΩΩ1;", `rt(x, (__ΩΩ1 || (__ΩΩ1 = ["&"])))`},
		},
		{
			name:   "TypeOfIsNotHoisted",
			files:  map[string]string{"app.ts": "declare function typeOf<T>(args?: any[], type?: ReceiveType<T>): any;\nexport const t = typeOf<string>();"},
			absent: []string{"__ΩΩ"},
		},
		{
			name:   "CallsWithoutReceiveTypeDeclareNothing",
			files:  map[string]string{"app.ts": "function plain<T>(x: T): T { return x; }\nexport const v = plain<string>('a');"},
			absent: []string{"__ΩΩ"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			js := compileHoistFiles(t, tc.files)
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

// The hoisted variable is declared before any other statement of the module,
// so that the call sites never see it undeclared.
func TestHoistTypeArgumentsDeclaredFirst(t *testing.T) {
	t.Parallel()
	js := compileHoistFile(t, receiveTypeFn+"interface User { id: number }\nexport const u = rt<User>({});")
	decl := strings.Index(js, "var __ΩΩ0;")
	use := strings.Index(js, "rt({}, (__ΩΩ0 ||")
	if decl < 0 || use < 0 || decl > use {
		t.Errorf("expected the declaration before its use\noutput:\n%s", js)
	}
}
