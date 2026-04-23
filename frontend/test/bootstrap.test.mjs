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
  let innerHTMLValue = "";
  return {
    textContent: "",
    innerHTMLWrites: 0,
    value: "",
    disabled: false,
    checked: false,
    open: false,
    classList: createClassList(),
    listeners: new Map(),
    addEventListener(type, handler) {
      this.listeners.set(type, handler);
    },
    focus() {},
    get innerHTML() {
      return innerHTMLValue;
    },
    set innerHTML(value) {
      innerHTMLValue = value;
      this.innerHTMLWrites += 1;
    },
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
      if (selector === "#program-form") {
        element.reset = () => {};
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
        ChooseProgramPath: async () => "C:\\tools\\ghost-app.bat",
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

refs.get("#create-button").listeners.get("click")();
assert.equal(refs.get("#advanced-options").open, false);

await refs.get("#choose-path").listeners.get("click")();
assert.equal(refs.get("#path-input").value, "C:\\tools\\ghost-app.bat");
assert.equal(refs.get("#name-input").value, "ghost-app");
assert.equal(refs.get("#working-directory-input").value, "C:\\tools");

const renderCount = refs.get("#programs").innerHTMLWrites;
await app.refreshPrograms();
assert.equal(refs.get("#programs").innerHTMLWrites, renderCount);
