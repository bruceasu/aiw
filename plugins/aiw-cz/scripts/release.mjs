import { build } from "esbuild";
import { cpSync, existsSync, lstatSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { arch, platform, report } from "node:process";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const plugin = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const release = join(plugin, "release");
const platformId = platform === "linux" && !report?.getReport()?.header?.glibcVersionRuntime
  ? `linuxmusl-${arch}` : `${platform}-${arch}`;
const triples = {
  "win32-x64": "x86_64-pc-windows-msvc",
  "win32-arm64": "aarch64-pc-windows-msvc",
  "linux-x64": "x86_64-unknown-linux-musl",
  "linux-arm64": "aarch64-unknown-linux-musl",
  "linuxmusl-x64": "x86_64-unknown-linux-musl",
  "linuxmusl-arm64": "aarch64-unknown-linux-musl",
};

function packageRoot(name) {
  try {
    return dirname(require.resolve(`${name}/package.json`));
  } catch (error) {
    if (error?.code !== "ERR_PACKAGE_PATH_NOT_EXPORTED") throw error;
    return dirname(require.resolve(name));
  }
}

function requireFile(path, label) {
  if (!existsSync(path) || !statSync(path).isFile()) throw new Error(`${label} missing: ${path}`);
}

function bytes(path) {
  if (!existsSync(path)) return 0;
  if (statSync(path).isFile()) return statSync(path).size;
  return readdirSync(path).reduce((total, name) => total + bytes(join(path, name)), 0);
}

function files(path, root = path) {
  return readdirSync(path, { withFileTypes: true }).flatMap((entry) => {
    const child = join(path, entry.name);
    return entry.isDirectory() ? files(child, root) : [{ path: relative(root, child).replaceAll("\\", "/"), bytes: statSync(child).size }];
  });
}

function rejectDevelopmentDirectories(path) {
  for (const entry of readdirSync(path, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    if (entry.name === "src" || entry.name === "node_modules") {
      throw new Error(`development directory in CZ release: ${join(path, entry.name)}`);
    }
    rejectDevelopmentDirectories(join(path, entry.name));
  }
}

async function createRelease() {
  if (!triples[platformId]) throw new Error(`unsupported CZ release platform: ${platformId}`);
  const baseline = readdirSync(plugin).filter((name) => name !== "release")
    .reduce((total, name) => total + bytes(join(plugin, name)), 0);
  const copilotPackage = `@github/copilot-sdk-${platformId}`;
  const codexPackage = `@openai/codex-${platform === "linux" ? `linux-${arch}` : platformId}`;
  const copilotSource = packageRoot(copilotPackage);
  const codexSource = packageRoot(codexPackage);
  const copilotBinary = join(copilotSource, "prebuilds", platformId,
    platform === "win32" ? "copilot-runtime.exe" : "copilot-runtime");
  requireFile(copilotBinary, "Copilot runtime");
  requireFile(join(dirname(copilotBinary), "runtime.node"), "Copilot native runtime");
  requireFile(join(codexSource, "vendor", triples[platformId], "bin",
    platform === "win32" ? "codex.exe" : "codex"), "Codex CLI");

  rmSync(release, { recursive: true, force: true });
  mkdirSync(join(release, "dist"), { recursive: true });
  await build({
    entryPoints: [join(plugin, "src", "index.ts")],
    outfile: join(release, "dist", "index.js"),
    bundle: true,
    platform: "node",
    format: "esm",
    target: "node22",
    // Koffi loads a platform-specific .node file relative to its own module.
    plugins: [{ name: "external-koffi", setup(buildApi) {
      buildApi.onResolve({ filter: /^koffi$/ }, () => ({ path: "../vendor/koffi/runtime/koffi/index.js", external: true }));
    } }],
  });

  cpSync(join(plugin, "aiw-cz.js"), join(release, "aiw-cz.js"));
  cpSync(join(plugin, "locales"), join(release, "locales"), { recursive: true });
  cpSync(copilotSource, join(release, "runtime", "copilot"), { recursive: true });
  cpSync(codexSource, join(release, "runtime", "codex"), { recursive: true });
  const koffiRoot = packageRoot("koffi");
  const koffiRuntime = join(release, "vendor", "koffi", "runtime", "koffi");
  mkdirSync(koffiRuntime, { recursive: true });
  const koffiSource = readFileSync(join(koffiRoot, "src", "koffi", "index.js"), "utf8");
  const staticImport = 'from "./src/static.js"';
  if (!koffiSource.includes(staticImport)) throw new Error("Koffi runtime layout changed");
  writeFileSync(join(koffiRuntime, "index.js"), koffiSource.replace(staticImport, 'from "./static.js"'));
  cpSync(join(koffiRoot, "src", "koffi", "src", "static.js"), join(koffiRuntime, "static.js"));
  cpSync(join(koffiRoot, "LICENSE.txt"), join(release, "vendor", "koffi", "LICENSE.txt"));
  const koffiPlatform = `@koromix/koffi-${platform}-${arch}`;
  cpSync(packageRoot(koffiPlatform), join(release, "vendor", "@koromix", `koffi-${platform}-${arch}`), { recursive: true });
  for (const [name, source] of [
    ["codex-sdk-LICENSE", join(plugin, "node_modules", "@openai", "codex-sdk", "LICENSE")],
    ["openai-LICENSE", join(plugin, "node_modules", "openai", "LICENSE")],
    ["smol-toml-LICENSE", join(plugin, "node_modules", "smol-toml", "LICENSE")],
  ]) {
    requireFile(source, `${name} license`);
    mkdirSync(join(release, "licenses"), { recursive: true });
    cpSync(source, join(release, "licenses", name));
  }
  writeFileSync(join(release, "package.json"), '{"type":"module"}\n');
  rejectDevelopmentDirectories(release);
  const manifest = {
    platform: platformId,
    baselineBytes: baseline,
    resources: [
      { path: "aiw-cz.js", purpose: "AIW plugin entry" },
      { path: "dist/index.js", purpose: "bundled TypeScript and JavaScript dependencies" },
      { path: "locales/", purpose: "built-in commit wizard translations" },
      { path: "runtime/copilot/", source: copilotPackage, purpose: "Copilot CLI, native runtime, and platform assets" },
      { path: "runtime/codex/", source: codexPackage, purpose: "Codex CLI and platform assets" },
      { path: "vendor/koffi/", source: "koffi", purpose: "Copilot in-process native loader and license" },
      { path: `vendor/@koromix/koffi-${platform}-${arch}/`, source: koffiPlatform, purpose: "Koffi native module" },
      { path: "licenses/", purpose: "licenses for bundled JavaScript dependencies" },
      { path: "package.json", purpose: "Node.js ESM marker" },
    ],
    files: files(release),
  };
  writeFileSync(join(release, "release-manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);
  console.log(`aiw-cz ${platformId}: baseline ${baseline} bytes; release ${bytes(release)} bytes`);
}

function installRelease(targetArg) {
  const target = resolve(targetArg);
  if (target === plugin || dirname(target) === plugin || target.toLowerCase() !== join(dirname(target), "aiw-cz").toLowerCase()
      || dirname(target).split(/[\\/]/).at(-1)?.toLowerCase() !== "plugins") {
    throw new Error("CZ install target must be an aiw-cz directory below plugins");
  }
  if (!existsSync(join(release, "release-manifest.json"))) throw new Error("build the CZ release first");
  const manifest = JSON.parse(readFileSync(join(release, "release-manifest.json"), "utf8"));
  if (manifest.platform !== platformId) throw new Error(`CZ release is for ${manifest.platform}, not ${platformId}`);
  if (existsSync(target) && lstatSync(target).isSymbolicLink()) throw new Error("CZ install target cannot be a symlink");
  mkdirSync(target, { recursive: true });
  // Remove only known generated/development paths. Keep local cz.toml overrides.
  for (const name of ["src", "node_modules", "dist", "runtime", "vendor", "scripts", "tsconfig.json", "package-lock.json"]) {
    rmSync(join(target, name), { recursive: true, force: true });
  }
  cpSync(release, target, { recursive: true, force: true });
  console.log(`installed aiw-cz release at ${target}`);
}

if (process.argv[2] === "--install" && process.argv.length === 4) installRelease(process.argv[3]);
else if (process.argv.length === 2) await createRelease();
else throw new Error("usage: node scripts/release.mjs [--install <plugins/aiw-cz directory>]");
