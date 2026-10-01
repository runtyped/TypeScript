# @runtyped/typescript

A fork of Microsoft's official TypeScript compiler that extends upstream with
runtime type reflection. Use with [@runtyped/type].

## Usage

`@runtyped/typescript` is a drop-in replacement for the official compiler. 
It is designed to be installed and used exactly as the [typescript] package.

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

A small javascript wrapper selects the right binary at runtime.

## Versioning

Versioning of `@runtyped/typescript` matches that of the upstream [typescript]
package in its major and minor components. The patch component is reserved for
runtyped itself.

Example: `@runtyped/typescript@7.1.0` is the first release built upon `7.1.x`
versions of upstream [typescript].

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
