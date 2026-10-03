import { mkdtemp, open, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { spawn } from "node:child_process";

function commandWords(value) {
  if (!value?.trim()) throw new Error("请设置 EDITOR，例如 notepad.exe 或 vi。");
  const words = [];
  let word = "";
  let quote;
  let started = false;
  for (const character of value) {
    if (quote) {
      if (character === quote) quote = undefined;
      else word += character;
    } else if (character === '"' || character === "'") {
      quote = character;
      started = true;
    } else if (/\s/.test(character)) {
      if (started) { words.push(word); word = ""; started = false; }
    } else {
      word += character;
      started = true;
    }
  }
  if (quote) throw new Error("EDITOR 中的引号未闭合。");
  if (started) words.push(word);
  if (!words[0]) throw new Error("EDITOR 必须包含编辑器程序。");
  return words;
}

export async function editPrompt(initial, maxBytes) {
  const [executable, ...args] = commandWords(process.env.EDITOR);
  let directory;
  try {
    directory = await mkdtemp(join(tmpdir(), "aiw-ai-editor-"));
    const filename = join(directory, "prompt.txt");
    await writeFile(filename, initial, { encoding: "utf8", mode: 0o600 });
    const result = await new Promise((done) => {
      const child = spawn(executable, [...args, filename], { stdio: "inherit", shell: false });
      child.once("error", () => done("start_failed"));
      child.once("close", (code, signal) => done(code === 0 && !signal ? "ok" : "exit_failed"));
    });
    if (result === "start_failed") throw new Error("无法启动 EDITOR；Windows 请使用真实 .exe，不使用 .cmd/.ps1。");
    if (result !== "ok") throw new Error("编辑器未正常退出，未发送请求。");
    const handle = await open(filename, "r");
    let bytes;
    try {
      const buffer = Buffer.alloc(maxBytes + 1);
      let size = 0;
      while (size < buffer.length) {
        const read = await handle.read(buffer, size, buffer.length - size, size);
        if (!read.bytesRead) break;
        size += read.bytesRead;
      }
      bytes = buffer.subarray(0, size);
    } finally {
      await handle.close();
    }
    if (bytes.length > maxBytes) throw new Error("编辑后的提示词过大。");
    try { return new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
    catch { throw new Error("编辑后的提示词必须是有效 UTF-8。"); }
  } catch (error) {
    // Do not expose filesystem errors containing private paths.
    if (error.code || error instanceof RangeError || error instanceof TypeError) {
      throw new Error("无法创建或读取编辑器临时文件，未发送请求。");
    }
    throw error;
  } finally {
    if (directory) {
      try { await rm(directory, { recursive: true, force: true }); }
      catch { throw new Error("无法清理编辑器临时文件，未发送请求。"); }
    }
  }
}
