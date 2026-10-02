# FD-003 Worker report

The selected approach keeps TypeScript and produces a platform-specific `plugins/aiw-cz/release/` directory. `aiw-cz.js` still loads `dist/index.js`; esbuild bundles the TypeScript entry and JavaScript SDK code. The release script copies locales and required Copilot, Codex, and Koffi runtime assets, plus available third-party licenses. Its manifest lists every file and byte count, and maps resource groups to source packages, platform, and purpose. It rejects `src` and `node_modules` directories. `build.bat plugins` excludes the CZ development tree from the general plugin copy and installs this release directory while preserving `cz.toml` overrides.

Windows x64 baseline before adding esbuild: 623,895,219 bytes in `plugins/aiw-cz`, including 623,816,483 bytes in `node_modules`. After adding esbuild, the comparable development directory excluding `release/` was 635,766,603 bytes. The final release sample is 568,674,974 bytes (67,091,629 bytes smaller than the comparable development directory; approximately 10.6%). Most remaining size is the platform SDK runtimes. Linux uses the same manifest code with platform-specific optional packages but has not been built or run here.

Commands actually run:

- `npm install --save-dev esbuild`: succeeded; esbuild 0.28.2 added to the lockfile.
- `npm run build:release`: first run failed because Koffi does not export `package.json`; after a source fix, the second run succeeded but copied Koffi's `src` directory. Following a narrowed Koffi resource copy and explicit authorization, the final run succeeded and produced the sizes above.
- `npm exec tsc -- --noEmit -p tsconfig.json`: first run found a `process.report` type error; after a source fix, the retry passed.
- `node release/aiw-cz.js --help`: passed in the isolated release directory and printed the expected options and provider order.
- A read-only manifest check found 132 listed files, no missing files, no `src` or `node_modules` directory, and `git diff --check` passed.

Static review traced `build.bat cz` to the release script, `build.bat plugins` to the filtered copy and installer, Go plugin discovery to `aiw-cz.js`, and `src/llm.ts` to bundled Copilot and Codex binaries. The packaged Codex path restores the SDK's `codex-path` PATH prefix. The packaged Copilot path uses its runtime binary and leaves an explicit `COPILOT_CLI_PATH` override intact.

Not run: `build.bat plugins`, installation into `C:\green\aiw`, Linux packaging, the interactive wizard, live provider calls, tests, full builds, deployment, and publication. The `--help` result does not verify provider authentication or execution. Existing unrelated deletions of three Python test files were not touched or counted as FD-003 changes.
