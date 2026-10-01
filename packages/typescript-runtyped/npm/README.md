# @runtyped/typescript

A fork of Microsoft's official TypeScript compiler that extends upstream with
runtime type reflection. Use with [@runtyped/type].

## Why

TypeScript types disappear at run time. With this compiler they don't:

```typescript
import { cast } from '@runtyped/type';

interface User {
  id: number;
  registered: Date;
  username: string;
}

const user = cast<User>(JSON.parse(input));
user.registered instanceof Date; // true
```

No decorators, no schema duplication, no code-generation step: your types are
reflected into run-time values, and [@runtyped/type] puts them to work for
validation, serialization, JSON Schema generation and more.

## Usage

`@runtyped/typescript` is a drop-in replacement for the official compiler.
It is designed to be installed and used exactly as the [typescript] package.
There are no flags to enable and no code changes to make: compiling with this
compiler is the opt-in, and everything that works upstream works exactly as
before.

Install it as a dependency using `npm` or `yarn`:

```sh
npm install --save-dev @runtyped/typescript
```

Call via `npx` or directly:

```sh
npx tsc -p tsconfig.json
./node_modules/.bin/tsc -p tsconfig.json
```

Add it to your `package.json` scripts:

```json
{
  /* ... */
  "scripts": {
    "build": "tsc -p tsconfig.json"
  },
  /* ... */
}
```

Compiled output gains type-reflection data consumed by [@runtyped/type] at
run time; this package itself is only needed at build time.

This package is the compile-time counterpart to [@runtyped/type].

## Binaries

Starting with version 7.0, the TypeScript compiler has been ported to [Go].
As such, this package ships with pre-built binaries for the following targets:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`
- `windows/arm64`

A small JavaScript wrapper selects the right binary at run time.

## Versioning

Versioning of `@runtyped/typescript` is aligned to the reflection format it
emits rather than to upstream TypeScript. The major component is the format
era and moves in lockstep with `@runtyped/type` — same major version means
the same, compatible format era. The minor component carries
format-compatible changes; the patch component carries fixes and rebases,
and every rebase onto a new upstream TypeScript bumps the patch, so rebase
releases remain visible to range-based dependency updates.

The upstream TypeScript version this compiler is built upon is declared as
build metadata and as a `typescriptBase` field in the package manifest —
informational only, never part of version precedence or range matching. The
npm registry strips build metadata from the published version listing, so
the manifest field is the durable carrier. Example:
`@runtyped/typescript@2.0.0+typescript.7.1` is the first release of format
era 2, built on the `7.1` line of upstream [typescript], including
in-development snapshots of upstream `main` at packaging time (upstream's
most recent published release may lag behind).

`@runtyped/typescript` declares a `peerDependencies` requirement on
[@runtyped/type] covering its era, so the package manager itself refuses
mismatched combinations.

For the full scheme, see the [Runtyped versioning strategy].

## Credits

Being a patchset atop of [typescript], `@runtyped/typescript` builds upon all
of the incredible work done by the TypeScript maintainers and contributors.

Additionally, the Runtyped project started out as a selective fork of the
astounding [DeepKit] framework, focusing solely on its type reflection modules.
All credit for Runtyped's approach to type reflection goes to Marc J. Schmidt
([@marcj]). For more information see [Relationship to DeepKit].

## License

APACHE-2.0, just like the official [typescript] package.


[Go]: https://go.dev/
[@marcj]: https://github.com/marcj
[DeepKit]: https://github.com/deepkit/deepkit
[typescript]: https://www.npmjs.com/package/typescript
[@runtyped/type]: https://www.npmjs.com/package/@runtyped/type
[Relationship to DeepKit]: https://github.com/runtyped/runtyped#relationship-to-deepkit
[Runtyped versioning strategy]: https://github.com/runtyped/runtyped#versioning
