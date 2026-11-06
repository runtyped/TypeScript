# Packaging — @runtyped/typescript

Docker-based build and test pipelines for the npm package shipping the
runtyped compiler's native binaries. The same Dockerfiles run locally today
and in CI/CD tomorrow with no porting:

- `Dockerfile` builds and packs the npm package.
- `Dockerfile.test` runs every test suite, the linters and the format check.

## Why this shape

- **Pure Go, native cross-compilation.** The compiler has zero cgo imports,
  so `GOOS`/`GOARCH` builds run from any machine — no Zig toolchain, no
  emulation, no per-platform build runners.
- **One package, runtime selection.** All six binaries ship in a single
  package (~10 MB gz each, ~60 MB total); `bin/tsc.js` picks the right one
  via `process.platform`/`process.arch`. Nothing is downloaded at install
  time.
- **Docker-only pipeline.** `golang:1.27-alpine` builds, `node:22-alpine`
  assembles and smoke-tests (`tsc --version` must succeed inside the image),
  then `npm pack`.

If the single package ever becomes too heavy, the documented escape hatch is
the esbuild model: per-platform packages (`@runtyped/typescript-linux-x64`,
…) wired via `optionalDependencies` — more packages, not more infrastructure.

## Run the tests

From the repository root, on any machine with Docker:

```sh
docker build -t runtyped-typescript-test -f packages/typescript-runtyped/Dockerfile.test .
docker run --rm runtyped-typescript-test
```

The container runs `npx hereby check:format` followed by
`npx hereby validate --all`, and exits non-zero if either fails. That covers:

- the Go test suites: upstream compiler and language server tests plus the
  runtyped transformer tests (`tsc/internal/transformers/runtyped/...`);
- the extension, TypeScript API (including wasm) and tools tests, and a smoke
  test of the built `tsc`;
- golangci-lint with the repo's custom linters, and dprint formatting.

Expect a full run to take a while (well over ten minutes). Everything needing
the network (npm packages, Go modules, the custom linter build, generated LSP
sources and `@runtyped/type` for the runtime tests) is fetched while building
the image, so the test run itself is self-contained. The build context is
filtered by `Dockerfile.test.dockerignore`, which keeps host artifacts such as
`node_modules/`, `built/` and the cached `tools/custom-gcl` and
`tools/gotestsum` binaries out of the Linux image.

To run a subset, override the command, e.g.:

```sh
# Only the runtyped tests
docker run --rm runtyped-typescript-test go -C tsc test ./internal/transformers/runtyped/...

# Only the linters
docker run --rm runtyped-typescript-test npx hereby lint
```

Rebuild the image after changing sources; the dependency layers are cached, so
rebuilds after code edits are quick.

### The reflection transformer in tests

The shipped compiler always reflects. In Go test binaries the runtyped
transformers are **off by default** (`runtyped.Enabled()` defaults to
`!testing.Testing()`), so the upstream tests run against the stock emit
pipeline and their reference baselines never need runtyped-specific updates.
The runtyped test packages turn the transformers on in their `TestMain`. Keep
it that way when adding tests: runtyped behaviour is tested in
`tsc/internal/transformers/runtyped/`, not by changing upstream baselines.

## Build locally

From the repository root:

```sh
docker build -t runtyped-typescript-pkg -f packages/typescript-runtyped/Dockerfile .
docker run --rm -v "$(pwd)/packages/typescript-runtyped/dist:/out" runtyped-typescript-pkg
```

Output: `packages/typescript-runtyped/dist/runtyped-typescript-<version>.tgz`

## Verify before publishing

```sh
tar -tzf packages/typescript-runtyped/dist/*.tgz            # inspect contents
tar -xzf packages/typescript-runtyped/dist/*.tgz -C /tmp && cd /tmp/package
./bin/tsc.js --version                    # runs the launcher + native binary
```

## Publish

```sh
npm publish packages/typescript-runtyped/dist/*.tgz --access public
```

In CI, publish from the assemble stage with `NPM_TOKEN` provided instead.

## Releasing

1. Bump `version` in `packages/typescript-runtyped/npm/package.json` to upstream version.
2. Rebuild the image, verify, publish.
