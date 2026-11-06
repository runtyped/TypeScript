# AGENTS.md — runtyped fork

Notes for coding agents working on **runtyped**, a fork of Microsoft's native
(Go) TypeScript compiler that adds runtime type reflection. This file covers the
fork-specific parts. Read `.github/copilot-instructions.md` (repo root) as well:
it has the upstream rules (hereby tasks, compiler tests, baselines, PR format).

## What the fork is

- **Product:** the `@runtyped/typescript` npm package, a drop-in `tsc` that
  emits reflection data (`static __type = [...]` on classes, `__Ω<Name>` arrays
  for types, `__assignType(fn, [...])` for functions). The runtime counterpart
  is `@runtyped/type`. See `npm/README.md` (user-facing) and `README.md`
  (packaging).
- **Origin:** a Go port of DeepKit's `@deepkit/type-compiler`. When the intent
  of some code is unclear, the DeepKit TypeScript implementation is the
  reference, but it has bugs too (e.g. it relied on JS `splice` clamping an
  out-of-range index; see "Op lists and frames" below).
- **Branch model:** the `runtyped` branch carries a small set of commits that is
  regularly **rebased onto `origin/main`** (upstream). Keep the upstream
  footprint minimal so rebases stay cheap.

## Where things live

| Path | What |
|---|---|
| `tsc/internal/transformers/runtyped/` | The reflection transformer (Go package `runtyped`) |
| `  runtyped.go` | `reflectionTransformer`: AST visitor, entry `NewReflectionTransformer` |
| `  compiler.go` | `typeCompiler`: turns type nodes into op programs, resolves declarations/imports |
| `  program.go` | `compilerProgram`: op lists, co-routines, frames, stack; `buildPackStruct` |
| `  ops.go` | `Op*` constants; **must match `@runtyped/type-spec`'s `ReflectionOp` exactly** |
| `  declaration_transformer.go` | Appends `export declare type __ΩX = any[]` to `.d.ts` output |
| `  enabled.go` | On/off switch for the transformers (see "Tests") |
| `  helpers.go`, `jsdoc.go` | Shared helpers (`parseTreeParent`, `getIdentifierName`, `@reflection` JSDoc) |
| `  config/` | tsconfig `reflection` option resolution (port of DeepKit's config/resolver) |
| `  loader/` | `Loader`: one-file-at-a-time API for bundlers (port of `DeepkitLoader`) |
| `packages/typescript-runtyped/` | Packaging: `Dockerfile`, `Dockerfile.test`, `npm/` (package.json, `bin/tsc.js` launcher, user README) |

**Upstream files the fork touches** (keep this list short, and mark additions
with a runtyped comment):

- `tsc/internal/compiler/emitter.go`: adds `runtyped.NewReflectionTransformer`
  first in the script transformer chain (it must run before type erasure) and
  `runtyped.NewDeclarationTransformer` after the declaration transformers, both
  guarded by `runtyped.Enabled()`.
- `tsc/internal/transformers/chain.go`: `TransformOptions.SourceFiles` and
  `TransformOptions.ReflectionMode`.

Check the footprint with:
`git --no-pager diff --stat origin/main -- tsc ':(exclude)tsc/internal/transformers/runtyped'`

## Tests

- **Upstream tests run without the transformer.** `runtyped.Enabled()` defaults
  to `!testing.Testing()`: the shipped compiler always reflects, but every Go
  test binary gets the stock emit pipeline, so upstream reference baselines
  never contain reflection output. **Never accept baselines that contain
  `__type`, `__Ω` or `__assignType`**; that means the switch is broken.
- **Runtyped tests opt in** via `TestMain` (`runtyped/main_test.go`,
  `runtyped/loader/main_test.go`) calling `runtyped.SetEnabled(true)`. Any new
  test package that needs the transformer needs the same `TestMain`.
- Test runtyped behaviour with Go tests in `tsc/internal/transformers/runtyped/`
  using the helpers in the `runtyped_test` package: `compileSingleFileJS`
  (`harness_smoke_test.go`), `compileFilesJS` (`crash_regression_test.go`),
  `compileHoistFiles`, plus `assertContains`/`assertNotContains`. Do **not** add
  runtyped cases to `tsc/testdata/tests/cases/`.
- Reproduce bugs with a failing regression test first (see
  `crash_regression_test.go` for the pattern).
- `TestHoist_Runtime_*` execute output with node and need `@runtyped/type` at
  `/app/node_modules`. They **skip** when it can't be resolved and only really
  run inside the `Dockerfile.test` image.
- Quick loop:
  ```sh
  go -C ./tsc vet ./internal/transformers/runtyped/... ./internal/compiler/
  go -C ./tsc test -count=1 ./internal/transformers/runtyped/...
  go -C ./tsc test -race -count=1 ./internal/transformers/runtyped/...
  ```
- Full validation, anywhere Docker is available (see `README.md`):
  ```sh
  docker build -t runtyped-typescript-test -f packages/typescript-runtyped/Dockerfile.test .
  docker run --rm runtyped-typescript-test
  ```
  Natively: `npx hereby check:format && npx hereby validate --all`. Note that
  `validate` runs `format` (it rewrites files) rather than `check:format`.

## Invariants and pitfalls in the transformer

- **Never read `.Parent` off visitor output.** Factory `Update*` functions don't
  copy `Parent`, so any rewritten node has `Parent == nil`. Walk up via the parse
  tree: `parseTreeParent(emitContext, node)` / `tx.parseTreeParent` /
  `tc.parent`, which map back with `EmitContext().ParseNode`. The custom lint
  rule `forbidparentaccess` enforces this; the one allowed raw access is inside
  `parseTreeParent`, with a `//nolint:customlint // reason`.
- **Read other files' symbol tables read-only.** Files are emitted concurrently.
  Use `node.Locals()` (and nil-safe `SymbolTable.Lookup`/`Get`), never
  `ast.GetLocals`, which lazily *creates* the table and races (`-race` catches
  it).
- **Op lists and frames (`program.go`).** There is a main op list plus a stack
  of active co-routines (used for conditional branches, mapped types,
  distributive conditionals). A `frame` records the op list it belongs to
  (`frame.coRoutine`, nil = main) and its start `opIndex`. `pushOpAtFrame`
  inserts into *the frame's* list, not the active one; `infer` declares its
  variable on an enclosing conditional frame. `pushCoRoutine` activates the
  co-routine **before** pushing its implicit frame.
- **Export names can be string literals** (ES2022 `export { "a b" } from ...`).
  Don't call `AsIdentifier()` on export or import specifier names; check
  `Kind` or use `getIdentifierName`.
- Upstream conformance tests (`tsc/testdata/tests/cases/conformance/**`) are a
  great fuzz corpus. If the transformer ever crashes, temporarily enable it for
  the test runner (add `runtyped.SetEnabled(true)` to the existing `TestMain`
  in `tsc/internal/testrunner/testmain_test.go`, never committed) and run
  `go -C ./tsc test -run 'TestLocal' ./internal/testrunner 2>&1 | grep -A30 "Panic on"`.

## Lint conventions (`npx hereby lint`, must report 0 issues)

- Paths: use `tspath`, not `path`/`filepath` (`forbidigo`). Don't convert
  strings to `tspath.RootedFilePath` directly (`typedpaths`); use constructors
  (`RootedFilePathFromNormalized`, `TryRootedFilePathFromAbsolute`,
  `ToRootedFilePath`) and accessors (`.AsString()`, `.BaseName()`,
  `.Directory()`). Untyped string constants (`"/app.ts"`) are fine.
- JSON: `github.com/microsoft/TypeScript/tsc/internal/json` (encoding/json v2
  wrapper), not `encoding/json` (`depguard`).
- Tests: `t.Parallel()` in every test and subtest (`paralleltest`),
  `t.TempDir()` for temp files (`usetesting`).
- `//nolint:<linter> // <reason>` only with a real justification, as upstream
  does.
- Formatting is dprint (gofumpt for Go): `npx dprint fmt <files>`.

## Rebasing onto upstream

1. `git fetch` and rebase `runtyped` onto `origin/main`.
2. `go -C ./tsc vet ./...` lists the breakages. The compiler stops at 10 errors
   per package, so iterate, or grep for patterns.
3. Typical upstream churn seen so far: `FileName()` became
   `tspath.RootedFilePath`; `ast.SymbolTable` became `*SymbolMap` (use
   `Lookup`/`Get`, not indexing); `Symbol` fields became methods
   (`Declarations()`, `Members()`, `Exports()`); `vfs.FS` takes rooted path types
   and has `CaseSensitivity()`; `compiler.NewCompilerHost` lost its current
   directory parameter; `ParsedCommandLine` must be built with
   `tsoptions.NewParsedCommandLine` (private `baseDirectory`); test helpers like
   `tsoptionstest.VfsParseConfigHost` were removed (use
   `tsoptions.NewDefaultParseConfigHost(tsoptionstest.NewVFS(...))`). Look at
   how upstream callers changed (`git log -S <symbol> origin/main`) and mirror
   them.
4. Run the quick loop, lint, then the Docker validation.
5. When something fails, check whether pristine upstream fails too before
   debugging it. Export it read-only:
   `mkdir -p built/upstream && git archive origin/main -- tsc/go.mod tsc/go.sum tsc/cmd tsc/internal tsc/testdata/fixtures | tar -x -C built/upstream`,
   then run `GOWORK=off go -C ./tsc test ...` inside it (same trick with
   `origin/runtyped` for "was this broken before the rebase?"). `built/` is
   gitignored; delete the export afterwards.

## Running in a sandboxed agent terminal (macOS)

Without the right permissions these commands fail in confusing ways:

- Go needs write access to `~/Library/Caches/go-build` and `~/go/pkg` (module
  cache, `sumdb`), plus hosts `proxy.golang.org`, `sum.golang.org`,
  `storage.googleapis.com`. The custom linter build also needs `github.com`,
  `*.githubusercontent.com`, `codeberg.org`, and `~/Library/Caches/golangci-lint`.
- npm/dprint need `registry.npmjs.org`, `*.npmjs.org`, `plugins.dprint.dev`,
  `~/.npm` and `~/Library/Caches/dprint`. Run `npm ci` first if `node_modules/`
  is missing (the `astnav` tests and dprint depend on it).
- `hereby generate` (part of `build`/`validate`) uses Node's built-in `fetch`,
  which ignores the sandbox proxy unless `NODE_USE_ENV_PROXY=1` is set.
- Shell substitutions (`$VAR`, `$(...)`, `for` loops with `${x}`) are rejected
  by the agent terminal; use literal paths.
- Expected sandbox-only failures, not real bugs: `fswatch`/`lspwatcher` (FSEvents
  can't start) and `ipc` (`bind: invalid argument`, the temp path exceeds the
  unix socket length limit). `internal/api` hung until the 10-minute timeout on
  both this branch and pristine `origin/main`, so treat it as upstream or
  environmental unless proven otherwise.

## Packaging

- `Dockerfile` cross-compiles six `CGO_ENABLED=0` binaries and runs `npm pack`;
  `npm/bin/tsc.js` selects the binary at run time.
- `Dockerfile.test` (+ `Dockerfile.test.dockerignore`, which BuildKit applies
  automatically) runs `check:format` and `validate --all` on Linux. The ignore
  file must keep host artifacts out: `node_modules`, `built/`, and especially
  `tools/custom-gcl`/`tools/gotestsum` (macOS binaries cached against a
  `.hash` file, which Linux would otherwise try to execute).
- Versioning follows the reflection format, not upstream TypeScript; the
  upstream base is recorded in `npm/package.json` under `runtyped.typescript`.
  `peerDependencies["@runtyped/type"]` is also what `Dockerfile.test` installs
  for the runtime tests.

## House rules

- Commit prefixes on this branch: `runtyped:` (compiler/transformer),
  `packaging:` (this directory), `docs:`.
- Don't commit, branch or rebase unless asked; when asked, prefer small commits
  (rebase fixes separate from behaviour changes).
- Don't edit upstream code beyond the hooks listed above. If an upstream change
  seems necessary, prefer an extension point in the runtyped package.
