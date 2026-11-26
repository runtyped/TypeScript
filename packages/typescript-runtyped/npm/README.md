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

### Limiting reflection

By default every file is reflected. A top-level `reflection` field in
`tsconfig.json` narrows that down, and is inherited through `extends`:

```jsonc
{
  "reflection": true,          // or "default": reflect everything (same as omitting it)
  "reflection": false,         // or "never": reflect nothing
  "reflection": "explicit",    // only declarations tagged with a /** @reflection */ JSDoc comment
  "reflection": ["src/**/*.ts"] // only files matching these patterns, relative to this tsconfig.json
}
```

With `tsc -p`, the given `tsconfig.json` applies to every file in the program.
Without it, each file uses the nearest `tsconfig.json` above it.

### Type-only imports

A type-only import (`import type { User }` or `import { type User }`) is
dropped from the output, so its module is never loaded at run time. Without
the module, the compiler cannot reach the type's reflection data, so `User` is
reflected as `any`, and anything containing it validates less than it seems
to: a union with a `User` member accepts every value.

To reflect such types, use a plain import (`import { User }`), or set a
top-level `reflectTypeOnlyImports` field in `tsconfig.json`. That field is
inherited through `extends`, and you'll need it with `verbatimModuleSyntax`,
which requires `import type` for types:

```jsonc
{
  "reflectTypeOnlyImports": true
}
```

With it, a type-only import of an interface or type alias is reflected like a
plain import, and the compiled output imports the module at run time. This
only applies when the module is known to carry reflection data: a `.ts` file
that is reflected, or a `.d.ts` file that exports `__ΩUser`. Classes imported
with `import type` are still reflected as `any`.

This package is the compile-time counterpart to [@runtyped/type].

## Migrating from @runtyped/type-compiler

`@runtyped/typescript` supersedes [@runtyped/type-compiler], the JavaScript
transformer that patched the official `typescript` package up to the 6.x
series. Both [@runtyped/type-compiler] and support for TypeScript 6.x are
**deprecated** as of Runtyped 3.0.0: [@runtyped/type] 3.x requires this
compiler, while projects that must stay on TypeScript 6.x can keep using
[@runtyped/type-compiler] 2.x with [@runtyped/type] 2.x.

To migrate, replace `typescript` and `@runtyped/type-compiler` with
`@runtyped/typescript` and drop any call to `runtyped-install-transformer`.
Note that a `tsconfig.json` without a `reflection` field reflected nothing
with [@runtyped/type-compiler] but reflects every file with this compiler;
set `"reflection": false` to keep the old behaviour. See the
[Runtyped changelog] for the full migration notes.

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
emits rather than to upstream TypeScript: any version of [@runtyped/type] with
the same major version can read its output.

The upstream compiler it builds upon is tracked in the `runtyped.typescript`
field of `package.json`: `version` is the upstream release line, while
`commit` and `date` identify the commit of [microsoft/TypeScript]'s `main`
branch that this release is based on.

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
[Runtyped]: https://github.com/runtyped/runtyped
[typescript]: https://www.npmjs.com/package/typescript
[microsoft/TypeScript]: https://github.com/microsoft/TypeScript
[@runtyped/type]: https://www.npmjs.com/package/@runtyped/type
[Relationship to DeepKit]: https://github.com/runtyped/runtyped#relationship-to-deepkit
[Runtyped versioning strategy]: https://github.com/runtyped/runtyped#versioning
[Runtyped changelog]: https://github.com/runtyped/runtyped/blob/main/CHANGELOG.md
[@runtyped/type-compiler]: https://www.npmjs.com/package/@runtyped/type-compiler
