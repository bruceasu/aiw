import { pathToFileURL } from "node:url";
import { HOST, PORT } from "./config.js";
import { startService } from "./service.js";

export async function main(args: string[] = process.argv.slice(2)): Promise<void> {
  const [command, ...rest] = args;
  if (!command || command === "--help" || command === "help") {
    process.stdout.write("Usage: aiw agent-proxy start [--port 43127]\n");
    return;
  }
  if (command !== "start") throw new Error("Expected command: start");
  let port = PORT;
  for (let index = 0; index < rest.length; index++) {
    if (rest[index] === "--port" && rest[index + 1]) {
      const parsed = Number(rest[++index]);
      if (!Number.isInteger(parsed) || parsed < 1 || parsed > 65535) throw new Error("--port must be 1-65535");
      port = parsed;
    } else {
      throw new Error(`Unknown option: ${rest[index]}`);
    }
  }
  const service = await startService(port);
  process.stdout.write(`Agent Proxy listening on http://${HOST}:${port}\n`);
  let stopping = false;
  const stop = (): void => {
    if (stopping) return;
    stopping = true;
    void service.close().finally(() => { process.exitCode = 0; });
  };
  process.once("SIGINT", stop);
  process.once("SIGTERM", stop);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error: unknown) => {
    const message = error instanceof Error ? error.message : "startup failed";
    process.stderr.write(`aiw-agent-proxy: ${message}\n`);
    process.exitCode = 1;
  });
}
