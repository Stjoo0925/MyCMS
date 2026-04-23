import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

const cssPath = path.resolve("src/style.css");
const css = fs.readFileSync(cssPath, "utf8");

const rootOpen = css.indexOf(":root {");
const rootColorScheme = css.indexOf("color-scheme: dark;");
const rootClose = css.indexOf("}", rootOpen);
const straySequence = "}\r\n\r\n  color-scheme: dark;\r\n}";

assert.ok(rootOpen >= 0, `Missing :root block in ${cssPath}`);
assert.ok(rootColorScheme > rootOpen, `Missing color-scheme declaration in ${cssPath}`);
assert.ok(
  rootColorScheme < rootClose,
  `color-scheme declaration must stay inside the :root block in ${cssPath}`,
);
assert.equal(
  css.includes(straySequence),
  false,
  `style.css contains a broken root block near the color-scheme declaration`,
);
