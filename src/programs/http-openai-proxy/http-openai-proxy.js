import { existsSync } from "node:fs";

const entry = new URL("./dist/main.js", import.meta.url);
if (!existsSync(entry)) {
  process.stderr.write("http-openai-proxy is not compiled. Run npm run build in this directory.\n");
  process.exitCode = 1;
} else {
  const { main } = await import(entry.href);
  await main(process.argv.slice(2));
}
