import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

const frontendSrcDir = path.resolve("frontend/src");
const frontendIndexPath = path.resolve("frontend/index.html");

function collectJavaScriptFiles(directory) {
  const entries = fs.readdirSync(directory, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const fullPath = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...collectJavaScriptFiles(fullPath));
      continue;
    }

    if (entry.isFile() && fullPath.endsWith(".js")) {
      files.push(fullPath);
    }
  }

  return files;
}

function extractRelativeImports(source) {
  const matches = source.matchAll(
    /import\s+(?:[^"'()]+\s+from\s+)?["'](\.{1,2}\/[^"']+)["']/g,
  );
  return [...matches].map((match) => match[1]);
}

function hasExplicitJavaScriptExtension(specifier) {
  return specifier.endsWith(".js") || specifier.endsWith(".mjs");
}

const javascriptFiles = collectJavaScriptFiles(frontendSrcDir);
const offenders = [];

for (const filePath of javascriptFiles) {
  const source = fs.readFileSync(filePath, "utf8");
  const imports = extractRelativeImports(source);

  for (const specifier of imports) {
    if (!hasExplicitJavaScriptExtension(specifier)) {
      offenders.push(`${path.relative(frontendSrcDir, filePath)} -> ${specifier}`);
    }
  }
}

assert.deepEqual(
  offenders,
  [],
  `Relative JavaScript imports must include .js or .mjs extensions:\n${offenders.join("\n")}`,
);

const indexHtml = fs.readFileSync(frontendIndexPath, "utf8");
const assetReferences = [...indexHtml.matchAll(/(?:src|href)=["']([^"']+)["']/g)].map(
  (match) => match[1],
);
const absoluteAssetReferences = assetReferences.filter((reference) => reference.startsWith("/"));

assert.deepEqual(
  absoluteAssetReferences,
  [],
  `HTML asset references must be relative paths:\n${absoluteAssetReferences.join("\n")}`,
);
