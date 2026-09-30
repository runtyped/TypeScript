# Packaging — @runtyped/typescript

Docker-based build pipeline for the npm package shipping the runtyped
compiler's native binaries. The same Dockerfile runs locally today and in
CI/CD tomorrow with no porting.

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
