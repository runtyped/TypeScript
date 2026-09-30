#!/usr/bin/env node
"use strict";

/**
 * @runtyped/typescript launcher.
 *
 * The npm package ships pre-built native binaries of the runtyped tsc
 * (a fork of microsoft/TypeScript's native Go compiler with always-on
 * runtime type reflection). This shim selects the binary matching the
 * current platform and architecture and executes it.
 */

const { spawnSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const PLATFORMS = { linux: "linux", darwin: "darwin", win32: "windows" };
const ARCHES = { x64: "amd64", arm64: "arm64" };

const goPlatform = PLATFORMS[process.platform];
const goArch = ARCHES[process.arch];

if (!goPlatform || !goArch) {
  console.error(
    `@runtyped/typescript: unsupported platform ${process.platform}/${process.arch}.\n` +
      `Supported: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64.`
  );
  process.exit(1);
}

const executable = `tsc-${goPlatform}-${goArch}${goPlatform === "windows" ? ".exe" : ""}`;
const binaryPath = path.join(__dirname, "..", "binaries", executable);

if (!fs.existsSync(binaryPath)) {
  console.error(
    `@runtyped/typescript: binary ${executable} is missing from this installation (corrupted package?).\n` +
      `Try reinstalling: npm install @runtyped/typescript`
  );
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
});

if (result.error) {
  console.error(`@runtyped/typescript: failed to execute ${executable}: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
