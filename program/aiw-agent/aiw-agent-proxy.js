import { existsSync } from "node:fs";

const entry = new URL("./dist/main.js", import.meta.url);
if (!existsSync(entry)) {
  process.stderr.write("aiw-agent-proxy is not compiled. Run npm install && npm run build in the plugin directory.\n");
  process.exitCode = 1;
} else {
  const { main } = await import(entry.href);
  await main(process.argv.slice(2));
}
