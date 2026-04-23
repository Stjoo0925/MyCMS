import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

const packageJson = JSON.parse(fs.readFileSync(path.resolve("package.json"), "utf8"));
assert.equal(packageJson.type, "module", "frontend/package.json must declare ESM mode");
assert.ok(packageJson.scripts?.test, "frontend/package.json must define a test script");

const wailsJson = JSON.parse(fs.readFileSync(path.resolve("..", "wails.json"), "utf8"));
assert.equal(
  wailsJson["wailsjsdir"],
  "frontend/wailsjs",
  "wails.json should use a project-relative Wails JS output directory",
);
assert.equal(
  path.resolve(wailsJson.projectdir),
  path.resolve(".."),
  "wails.json should point projectdir at the repository root",
);
assert.equal(wailsJson["frontend:install"], "npm install");
assert.equal(wailsJson["frontend:dev"], "npm run dev");
assert.equal(wailsJson["frontend:dev:build"], "npm run build");
assert.equal(wailsJson["frontend:dev:install"], "npm install");
