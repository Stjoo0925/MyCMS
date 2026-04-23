const tests = [
  "./style-syntax.test.mjs",
  "./project-config.test.mjs",
  "./status-first-ui.test.mjs",
  "./wails-bridge.test.mjs",
  "./bootstrap.test.mjs",
];

for (const testFile of tests) {
  await import(new URL(testFile, import.meta.url));
}
