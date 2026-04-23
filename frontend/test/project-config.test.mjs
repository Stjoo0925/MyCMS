import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const packageJson = JSON.parse(fs.readFileSync(path.resolve("package.json"), "utf8"));
assert.equal(packageJson.type, "module", "frontend/package.json must declare ESM mode");
assert.ok(packageJson.scripts?.test, "frontend/package.json must define a test script");

const wailsJson = JSON.parse(fs.readFileSync(path.resolve("..", "wails.json"), "utf8"));
const repoRoot = path.resolve(fileURLToPath(new URL("../..", import.meta.url)));
const appIconScript = path.join(repoRoot, "scripts", "copy-appicon.ps1").replaceAll("\\", "/");
assert.equal(
  wailsJson["wailsjsdir"],
  "frontend/wailsjs",
  "wails.json should use a project-relative Wails JS output directory",
);
assert.ok(
  wailsJson.preBuildHooks?.["windows/*"]?.includes(appIconScript),
  "wails.json should invoke the app icon copy script",
);
assert.equal(
  path.resolve(wailsJson.projectdir),
  repoRoot,
  "wails.json should point projectdir at the repository root",
);
const iconScript = fs.readFileSync(path.resolve("..", "scripts", "copy-appicon.ps1"), "utf8");
assert.ok(
  iconScript.includes("assets/appicon.png"),
  "copy-appicon.ps1 should use assets/appicon.png as the source icon",
);
assert.ok(
  iconScript.includes("build/appicon.png"),
  "copy-appicon.ps1 should write the icon into build/appicon.png",
);
assert.ok(
  iconScript.includes("build/windows/icon.ico"),
  "copy-appicon.ps1 should remove build/windows/icon.ico so Wails regenerates it",
);
assert.equal(wailsJson["frontend:install"], "npm install");
assert.equal(wailsJson["frontend:dev"], "npm run dev");
assert.equal(wailsJson["frontend:dev:build"], "npm run build");
assert.equal(wailsJson["frontend:dev:install"], "npm install");
