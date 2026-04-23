import assert from "node:assert/strict";
import path from "node:path";
import { pathToFileURL } from "node:url";

globalThis.window = {};

const moduleUrl = pathToFileURL(path.resolve("src/app/api.js")).href;
const api = await import(`${moduleUrl}?t=${Date.now()}`);

await assert.rejects(
  api.listPrograms(),
  /Wails 런타임을 찾을 수 없습니다/,
);

await assert.rejects(
  api.getProgram("abc"),
  /Wails 런타임을 찾을 수 없습니다/,
);

await assert.rejects(
  api.getProgramLogs("abc", { limit: 10 }),
  /Wails 런타임을 찾을 수 없습니다/,
);
