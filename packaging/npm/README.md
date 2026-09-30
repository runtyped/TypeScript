# @runtyped/typescript

The runtyped TypeScript compiler: `tsc` with always-on runtime type
reflection, packaged as pre-built native binaries.

This is a fork of [microsoft/TypeScript](https://github.com/microsoft/TypeScript)'s
native Go compiler (`tsc/`). There is no opt-in flag and no config to flip:
**compiling your project with this `tsc` is the opt-in.** Types become
contracts with runtime reflection; escape hatches (`never`/`explicit`) are
planned via the reflection config package.

## Install

```sh
npm install --save-dev @runtyped/typescript
```

## Use

```sh
# Directly
npx tsc -p tsconfig.json

# Or via package scripts
tsc -p tsconfig.json
```

The package ships binaries for linux/amd64, linux/arm64, darwin/amd64,
darwin/arm64, windows/amd64 and windows/arm64; the launcher (`bin/tsc.js`)
picks the right one for your machine at runtime — nothing is downloaded at
install time.

## Versioning

`<upstream-version>-runtyped.<n>`, where the upstream part matches the
version the compiler binary itself reports (`tsc --version`) — e.g.
`7.1.0-runtyped.0` is the first runtyped release on top of a compiler
reporting 7.1.0-dev.
