import assert from "node:assert/strict";
import { mountApplication } from "../src/app/template.js";
import { closeFormPanel, createState, renderPrograms, setFormMode } from "../src/app/render.js";

const queriedSelectors = [];
const selectorRefs = new Map();

const root = {
  innerHTML: "",
  querySelector(selector) {
    queriedSelectors.push(selector);
    if (!selectorRefs.has(selector)) {
      selectorRefs.set(selector, { selector, classList: { toggle() {} } });
    }
    return selectorRefs.get(selector);
  },
};

const shellRefs = mountApplication(root);

assert.match(root.innerHTML, /summary-strip/);
assert.match(root.innerHTML, /프로그램 추가/);
assert.match(root.innerHTML, /프로그램 개요/);
assert.match(root.innerHTML, /고급 실행 옵션/);
assert.match(root.innerHTML, /이름과 실행 파일 경로만 입력하면 됩니다/);
assert.match(root.innerHTML, /panel-backdrop/);
assert.deepEqual(queriedSelectors, [
  "#page-error",
  "#programs",
  "#program-count",
  "#summary-running",
  "#summary-stopped",
  "#summary-attention",
  "#panel",
  "#panel-backdrop",
  "#panel-title",
  "#panel-description",
  "#program-form",
  "#panel-title",
  "#form-error",
  "#submit-button",
  "#cancel-edit",
  "#create-button",
  "#choose-path",
  "#advanced-options",
  "#name-input",
  "#description-input",
  "#notes-input",
  "#tags-input",
  "#path-input",
  "#working-directory-input",
  "#args-input",
  "#env-input",
  "#run-as-admin-input",
  "#restart-policy-input",
  "#restart-limit-input",
  "#restart-delay-input",
]);
assert.equal(shellRefs.pageErrorEl, selectorRefs.get("#page-error"));
assert.equal(shellRefs.programsEl, selectorRefs.get("#programs"));
assert.equal(shellRefs.programCountEl, selectorRefs.get("#program-count"));
assert.equal(shellRefs.summaryRunningEl, selectorRefs.get("#summary-running"));
assert.equal(shellRefs.summaryStoppedEl, selectorRefs.get("#summary-stopped"));
assert.equal(shellRefs.summaryAttentionEl, selectorRefs.get("#summary-attention"));
assert.equal(shellRefs.panelEl, selectorRefs.get("#panel"));
assert.equal(shellRefs.panelBackdropEl, selectorRefs.get("#panel-backdrop"));
assert.equal(shellRefs.panelTitleEl, selectorRefs.get("#panel-title"));
assert.equal(shellRefs.panelDescriptionEl, selectorRefs.get("#panel-description"));
assert.equal(shellRefs.formEl, selectorRefs.get("#program-form"));
assert.equal(shellRefs.formTitleEl, selectorRefs.get("#panel-title"));
assert.equal(shellRefs.formErrorEl, selectorRefs.get("#form-error"));
assert.equal(shellRefs.submitButtonEl, selectorRefs.get("#submit-button"));
assert.equal(shellRefs.cancelEditButtonEl, selectorRefs.get("#cancel-edit"));
assert.equal(shellRefs.createButtonEl, selectorRefs.get("#create-button"));
assert.equal(shellRefs.choosePathButtonEl, selectorRefs.get("#choose-path"));
assert.equal(shellRefs.advancedOptionsEl, selectorRefs.get("#advanced-options"));
assert.equal(shellRefs.nameInputEl, selectorRefs.get("#name-input"));
assert.equal(shellRefs.descriptionInputEl, selectorRefs.get("#description-input"));
assert.equal(shellRefs.notesInputEl, selectorRefs.get("#notes-input"));
assert.equal(shellRefs.tagsInputEl, selectorRefs.get("#tags-input"));
assert.equal(shellRefs.pathInputEl, selectorRefs.get("#path-input"));
assert.equal(shellRefs.workingDirectoryInputEl, selectorRefs.get("#working-directory-input"));
assert.equal(shellRefs.argsInputEl, selectorRefs.get("#args-input"));
assert.equal(shellRefs.envInputEl, selectorRefs.get("#env-input"));
assert.equal(shellRefs.runAsAdminInputEl, selectorRefs.get("#run-as-admin-input"));
assert.equal(shellRefs.restartPolicyEl, selectorRefs.get("#restart-policy-input"));
assert.equal(shellRefs.restartLimitInputEl, selectorRefs.get("#restart-limit-input"));
assert.equal(shellRefs.restartDelaySecondsInputEl, selectorRefs.get("#restart-delay-input"));

function createMockElement() {
  let innerHTMLValue = "";
  return {
    textContent: "",
    innerHTMLWrites: 0,
    value: "",
    checked: false,
    disabled: false,
    dataset: {},
    get innerHTML() {
      return innerHTMLValue;
    },
    set innerHTML(value) {
      innerHTMLValue = value;
      this.innerHTMLWrites += 1;
    },
    classList: {
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
    },
  };
}

function createDom() {
  const formEl = {
    resetCalled: 0,
    reset() {
      this.resetCalled += 1;
    },
  };

  return {
    pageErrorEl: createMockElement(),
    formErrorEl: createMockElement(),
    formTitleEl: createMockElement(),
    submitButtonEl: createMockElement(),
    cancelEditButtonEl: createMockElement(),
    createButtonEl: createMockElement(),
    choosePathButtonEl: createMockElement(),
    advancedOptionsEl: createMockElement(),
    nameInputEl: createMockElement(),
    descriptionInputEl: createMockElement(),
    notesInputEl: createMockElement(),
    tagsInputEl: createMockElement(),
    pathInputEl: createMockElement(),
    workingDirectoryInputEl: createMockElement(),
    argsInputEl: createMockElement(),
    envInputEl: createMockElement(),
    runAsAdminInputEl: createMockElement(),
    restartPolicyEl: createMockElement(),
    restartLimitInputEl: createMockElement(),
    restartDelaySecondsInputEl: createMockElement(),
    formEl,
    programCountEl: createMockElement(),
    programsEl: createMockElement(),
    summaryRunningEl: createMockElement(),
    summaryStoppedEl: createMockElement(),
    summaryAttentionEl: createMockElement(),
    panelEl: createMockElement(),
    panelBackdropEl: createMockElement(),
    panelTitleEl: createMockElement(),
    panelDescriptionEl: createMockElement(),
    emptyActionButtonEl: createMockElement(),
  };
}

const dom = createDom();
const state = createState();

assert.equal(state.hasLoaded, false);

renderPrograms(dom, state);
assert.match(dom.programsEl.innerHTML, /program-skeleton/);

state.programs = [
  { id: "a", name: "Survey Sync", path: "C:/sync.bat", launchMode: "cmd", status: "RUNNING", lastError: "", pid: 4242, startedAt: "2026-04-23T12:00:00.000Z", memoryWorkingSetBytes: 134217728 },
  { id: "b", name: "Nightly Backup", path: "D:/backup.bat", status: "STOPPED", lastError: "Exit code 1" },
  { id: "c", name: "Import Job", path: "E:/import.bat", status: "STARTING", lastError: "" },
  { id: "d", name: "Shutdown Job", path: "F:/shutdown.bat", status: "STOPPING", lastError: "" },
  { id: "e", name: "Detached Job", path: "G:/detached.bat", status: "ORPHANED", lastError: "Process missing" },
];
state.hasLoaded = true;

renderPrograms(dom, state);

assert.equal(dom.summaryRunningEl.textContent, "1");
assert.equal(dom.summaryStoppedEl.textContent, "1");
assert.equal(dom.summaryAttentionEl.textContent, "2");
assert.match(dom.programsEl.innerHTML, /Survey Sync/);
assert.match(dom.programsEl.innerHTML, /확인 필요/);
assert.match(dom.programsEl.innerHTML, /status-starting/);
assert.match(dom.programsEl.innerHTML, /status-stopping/);
assert.match(dom.programsEl.innerHTML, /status-orphaned/);
assert.match(dom.programsEl.innerHTML, /시작 중/);
assert.match(dom.programsEl.innerHTML, /중지 중/);
assert.match(dom.programsEl.innerHTML, /연결 끊김/);
assert.match(dom.programsEl.innerHTML, /PID 4242/);
assert.match(dom.programsEl.innerHTML, /메모리/);
assert.match(dom.programsEl.innerHTML, /실행 cmd/);
assert.match(dom.programsEl.innerHTML, /실행/);
assert.match(dom.programsEl.innerHTML, /card-actions/);
assert.doesNotMatch(dom.programsEl.innerHTML, /card-secondary-actions/);

const renderWriteCount = dom.programsEl.innerHTMLWrites;
renderPrograms(dom, state);
assert.equal(dom.programsEl.innerHTMLWrites, renderWriteCount);

state.programs = [];
renderPrograms(dom, state);
assert.match(dom.programsEl.innerHTML, /첫 프로그램 추가/);

setFormMode(dom, state, "edit", { id: "a", name: "Survey Sync", path: "C:/sync.bat" });
assert.equal(dom.panelTitleEl.textContent, "프로그램 수정");
assert.equal(dom.panelDescriptionEl.textContent, "상태 화면을 벗어나지 않고 이름이나 실행 경로를 수정할 수 있습니다.");
assert.equal(dom.cancelEditButtonEl.textContent, "닫기");
assert.equal(state.panelOpen, true);

closeFormPanel(dom, state);
assert.equal(state.panelOpen, false);
assert.equal(state.formMode, "create");
assert.equal(state.editId, "");
assert.equal(dom.panelEl.classList.contains("hidden"), true);
assert.equal(dom.panelBackdropEl.classList.contains("hidden"), true);
assert.equal(dom.formEl.resetCalled > 0, true);
