import assert from "node:assert/strict";
import path from "node:path";
import { pathToFileURL } from "node:url";

function createClassList() {
  return {
    values: new Set(),
    toggle(name, force) {
      if (force === undefined) {
        if (this.values.has(name)) {
          this.values.delete(name);
          return false;
        }
        this.values.add(name);
        return true;
      }
      if (force) {
        this.values.add(name);
        return true;
      }
      this.values.delete(name);
      return false;
    },
    contains(name) {
      return this.values.has(name);
    },
  };
}

function createElement() {
  return {
    textContent: "",
    innerHTML: "",
    value: "",
    disabled: false,
    classList: createClassList(),
    addEventListener() {},
    focus() {},
  };
}

const refs = new Map();
const root = {
  innerHTML: "",
  querySelector(selector) {
    if (!refs.has(selector)) {
      const element = createElement();
      if (selector === "#panel" || selector === "#panel-backdrop") {
        element.classList.toggle("hidden", true);
      }
      refs.set(selector, element);
    }
    return refs.get(selector);
  },
};

globalThis.document = {
  querySelector(selector) {
    if (selector === "#app") {
      return root;
    }
    return null;
  },
};

globalThis.window = {
  confirm() {
    return true;
  },
  go: {
    main: {
      App: {
        ListPrograms: async () => [],
        CreateProgram: async () => ({}),
        UpdateProgram: async () => ({}),
        DeleteProgram: async () => undefined,
        StartProgram: async () => undefined,
        StopProgram: async () => undefined,
        ChooseProgramPath: async () => "",
      },
    },
  },
};

globalThis.setInterval = () => 1;
globalThis.clearInterval = () => {};

const moduleUrl = pathToFileURL(path.resolve("src/app/bootstrap.js")).href;
const { bootstrapApplication } = await import(`${moduleUrl}?t=${Date.now()}`);

const app = bootstrapApplication(root, { refreshIntervalMs: 0 });
await app.ready;

assert.equal(app.state.panelOpen, false);
assert.equal(refs.get("#panel").classList.contains("hidden"), true);
assert.equal(refs.get("#panel-backdrop").classList.contains("hidden"), true);
