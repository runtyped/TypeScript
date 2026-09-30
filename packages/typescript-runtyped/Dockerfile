# Builds and assembles the @runtyped/typescript npm package.
#
# Pure-Go cross-compilation: one Linux builder produces all targets,
# no emulation or platform-specific machines required (CGO_ENABLED=0).
#
# Usage (from the repository root):
#   docker build -t runtyped-typescript-pkg -f packages/typescript-runtyped/Dockerfile .
#   docker run --rm -v "$(pwd)/packages/typescript-runtyped/dist:/out" runtyped-typescript-pkg
#
# Output: packages/typescript-runtyped/dist/runtyped-typescript-<version>.tgz
#
# Publish:
#   npm publish packages/typescript-runtyped/dist/*.tgz --access public
# (or run `npm publish` inside the assemble stage with NPM_TOKEN in CI)

FROM golang:1.27-alpine AS build

WORKDIR /src
COPY ../tsc/ ./tsc/

RUN set -eux; \
    cd /src/tsc/cmd/tsc; \
    mkdir -p /build/binaries; \
    for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do \
        os="${target%/*}"; arch="${target#*/}"; \
        ext=""; [ "$os" = "windows" ] && ext=".exe"; \
        GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 GOWORK=off \
        go build -trimpath -ldflags="-s -w" \
            -o "/build/binaries/tsc-$os-$arch$ext" .; \
    done

FROM node:22-alpine AS assemble

WORKDIR /pkg
COPY ./packages/typescript-runtyped/npm/ ./
COPY --from=build /build/binaries/ ./binaries/

# Smoke test: the linux/amd64 binary must run inside this container.
RUN ./binaries/tsc-linux-amd64 --version

# The tarball is baked into the image at /pkg-out. A bind-mount over /out
# at run time SHADOWS image content, so the container entrypoint copies
# the tarball into /out — the mounted volume — making it appear on the host.
RUN mkdir -p /pkg-out && npm pack --pack-destination /pkg-out

CMD ["sh", "-c", "mkdir -p /out && cp /pkg-out/*.tgz /out/"]
