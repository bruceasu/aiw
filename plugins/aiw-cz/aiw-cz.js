import { existsSync } from "node:fs";

const entry = new URL("./dist/index.js", import.meta.url);
if (!existsSync(entry)) {
  process.stderr.write("aiw-cz is not compiled. Run build.bat cz first.\n");
  process.exitCode = 1;
} else {
  await import(entry.href);
}
