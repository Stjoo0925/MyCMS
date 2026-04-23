import {
  chooseProgramPath,
  createProgram,
  deleteProgram,
  getErrorMessage,
  listPrograms,
  startProgram,
  stopProgram,
  updateProgram,
} from "./api.js";
import {
  closeFormPanel,
  createState,
  markPending,
  renderPrograms,
  setFormBusy,
  setFormError,
  setFormMode,
  setPageError,
} from "./render.js";
import { mountApplication } from "./template.js";

function createProgramsSignature(programs) {
  return JSON.stringify(
    programs.map((program) => ({
      id: program.id,
      status: program.status,
      lastError: program.lastError,
      pid: program.pid,
      startedAt: program.startedAt,
      lastExitAt: program.lastExitAt,
      memoryWorkingSetBytes: program.memoryWorkingSetBytes,
      memoryPrivateBytes: program.memoryPrivateBytes,
      restartCount: program.restartCount,
      canReconnect: program.canReconnect,
      updatedAt: program.updatedAt,
      name: program.name,
      path: program.path,
      launchMode: program.launchMode,
    })),
  );
}

function splitCommaSeparated(value) {
  return String(value)
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function splitLines(value) {
  return String(value)
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseEnv(value) {
  return splitLines(value).map((line) => {
    const index = line.indexOf("=");
    if (index <= 0) {
      throw new Error("환경 변수는 KEY=VALUE 형식이어야 합니다.");
    }

    return {
      key: line.slice(0, index).trim(),
      value: line.slice(index + 1),
    };
  });
}

function serializeLines(values) {
  return (values ?? []).join("\n");
}

function serializeEnv(values) {
  return (values ?? []).map((item) => `${item.key}=${item.value}`).join("\n");
}

function deriveWorkingDirectory(programPath) {
  const normalized = String(programPath ?? "").trim().replaceAll("/", "\\");
  if (!normalized) {
    return "";
  }

  const lastSeparator = normalized.lastIndexOf("\\");
  if (lastSeparator <= 0) {
    return "";
  }

  return normalized.slice(0, lastSeparator);
}

function deriveProgramName(programPath) {
  const normalized = String(programPath ?? "").trim().replaceAll("/", "\\");
  if (!normalized) {
    return "";
  }

  const lastSeparator = normalized.lastIndexOf("\\");
  const filename = lastSeparator >= 0 ? normalized.slice(lastSeparator + 1) : normalized;
  const lastDot = filename.lastIndexOf(".");
  if (lastDot <= 0) {
    return filename;
  }

  return filename.slice(0, lastDot);
}

function syncSuggestedFields(dom) {
  const currentPath = dom.pathInputEl.value.trim();
  if (!currentPath) {
    return;
  }

  if (!dom.nameInputEl.value.trim()) {
    dom.nameInputEl.value = deriveProgramName(currentPath);
  }

  if (!dom.workingDirectoryInputEl.value.trim()) {
    dom.workingDirectoryInputEl.value = deriveWorkingDirectory(currentPath);
  }
}

function hasAdvancedSettings(program) {
  if (!program) {
    return false;
  }

  const derivedWorkingDirectory = deriveWorkingDirectory(program.path);
  return Boolean(
    program.description ||
      program.notes ||
      program.tags?.length ||
      (program.workingDirectory && program.workingDirectory !== derivedWorkingDirectory) ||
      program.args?.length ||
      program.env?.length ||
      program.runAsAdmin ||
      (program.restartPolicy && program.restartPolicy !== "none") ||
      Number(program.restartLimit) > 0 ||
      Number(program.restartDelaySeconds) > 0,
  );
}

function readFormInput(dom) {
  return {
    name: dom.nameInputEl.value,
    description: dom.descriptionInputEl.value,
    notes: dom.notesInputEl.value,
    tags: splitCommaSeparated(dom.tagsInputEl.value),
    path: dom.pathInputEl.value,
    workingDirectory: dom.workingDirectoryInputEl.value,
    args: splitLines(dom.argsInputEl.value),
    env: parseEnv(dom.envInputEl.value),
    runAsAdmin: dom.runAsAdminInputEl.checked,
    restartPolicy: dom.restartPolicyEl.value,
    restartLimit: Number.parseInt(dom.restartLimitInputEl.value, 10) || 0,
    restartDelaySeconds: Number.parseInt(dom.restartDelaySecondsInputEl.value, 10) || 0,
  };
}

function setFormValues(dom, program) {
  dom.nameInputEl.value = program?.name ?? "";
  dom.descriptionInputEl.value = program?.description ?? "";
  dom.notesInputEl.value = program?.notes ?? "";
  dom.tagsInputEl.value = (program?.tags ?? []).join(", ");
  dom.pathInputEl.value = program?.path ?? "";
  dom.workingDirectoryInputEl.value = program?.workingDirectory ?? "";
  dom.argsInputEl.value = serializeLines(program?.args);
  dom.envInputEl.value = serializeEnv(program?.env);
  dom.runAsAdminInputEl.checked = Boolean(program?.runAsAdmin);
  dom.restartPolicyEl.value = program?.restartPolicy ?? "none";
  dom.restartLimitInputEl.value = String(program?.restartLimit ?? 0);
  dom.restartDelaySecondsInputEl.value = String(program?.restartDelaySeconds ?? 0);
}

function setCreateDefaults(dom) {
  dom.restartPolicyEl.value = "none";
  dom.restartLimitInputEl.value = "0";
  dom.restartDelaySecondsInputEl.value = "0";
  dom.runAsAdminInputEl.checked = false;
  dom.advancedOptionsEl.open = false;
}

export function bootstrapApplication(root, options = {}) {
  const dom = mountApplication(root);
  const state = createState();
  let refreshInFlight = null;
  let refreshTimerId = null;

  async function refreshPrograms(refreshOptions = {}) {
    if (refreshInFlight) {
      return refreshInFlight;
    }

    refreshInFlight = (async () => {
      try {
        const programs = await listPrograms();
        const nextSignature = createProgramsSignature(programs);
        const shouldRender = !state.hasLoaded || state.programsSignature !== nextSignature;
        state.programs = programs;
        state.programsSignature = nextSignature;
        state.hasLoaded = true;
        if (shouldRender) {
          renderPrograms(dom, state);
        }

        if (!refreshOptions.keepError) {
          setPageError(dom, state, "");
        }

        if (state.formMode === "edit" && state.editId) {
          const current = state.programs.find((program) => program.id === state.editId);
          if (!current) {
            closeFormPanel(dom, state);
          }
        }
      } catch (error) {
        state.hasLoaded = true;
        renderPrograms(dom, state);
        setPageError(dom, state, getErrorMessage(error));
      }
    })().finally(() => {
      refreshInFlight = null;
    });

    return refreshInFlight;
  }

  async function withProgramAction(id, action) {
    markPending(state, id, true);
    renderPrograms(dom, state);
    setPageError(dom, state, "");

    try {
      await action();
    } catch (error) {
      setPageError(dom, state, getErrorMessage(error));
    } finally {
      markPending(state, id, false);
      renderPrograms(dom, state);
      await refreshPrograms({ keepError: true });
    }
  }

  dom.formEl.addEventListener("submit", async (event) => {
    event.preventDefault();

    setFormError(dom, state, "");
    setPageError(dom, state, "");
    setFormBusy(dom, state, true);

    let payload;
    try {
      payload = readFormInput(dom);
    } catch (error) {
      dom.advancedOptionsEl.open = true;
      setFormError(dom, state, getErrorMessage(error));
      setFormBusy(dom, state, false);
      return;
    }

    try {
      if (state.formMode === "edit" && state.editId) {
        await updateProgram(state.editId, payload);
      } else {
        await createProgram(payload);
      }

      closeFormPanel(dom, state);
      await refreshPrograms();
    } catch (error) {
      setFormError(dom, state, getErrorMessage(error));
    } finally {
      setFormBusy(dom, state, false);
    }
  });

  dom.choosePathButtonEl.addEventListener("click", async () => {
    setFormError(dom, state, "");
    state.choosing = true;
    setFormBusy(dom, state, true);

    try {
      const path = await chooseProgramPath();
      if (path) {
        dom.pathInputEl.value = path;
        syncSuggestedFields(dom);
      }
    } catch (error) {
      setFormError(dom, state, getErrorMessage(error));
    } finally {
      state.choosing = false;
      setFormBusy(dom, state, false);
    }
  });

  dom.createButtonEl.addEventListener("click", () => {
    setFormMode(dom, state, "create");
    setCreateDefaults(dom);
    dom.nameInputEl.focus();
  });

  dom.pathInputEl.addEventListener("change", () => {
    syncSuggestedFields(dom);
  });

  dom.cancelEditButtonEl.addEventListener("click", () => {
    closeFormPanel(dom, state);
  });

  dom.panelBackdropEl.addEventListener("click", () => {
    closeFormPanel(dom, state);
  });

  dom.programsEl.addEventListener("click", async (event) => {
    if (!(event.target instanceof Element)) {
      return;
    }

    const button = event.target.closest("button[data-action]");
    if (!button) {
      return;
    }

    const id = button.dataset.id;
    const action = button.dataset.action;
    const program = state.programs.find((item) => item.id === id);

    if (action === "open-create") {
      setFormMode(dom, state, "create");
      setCreateDefaults(dom);
      dom.nameInputEl.focus();
      return;
    }

    if (!id || !action || !program) {
      return;
    }

    if (action === "edit") {
      setFormMode(dom, state, "edit", program);
      setFormValues(dom, program);
      dom.advancedOptionsEl.open = hasAdvancedSettings(program);
      dom.nameInputEl.focus();
      return;
    }

    if (action === "delete") {
      const confirmed = window.confirm(`${program.name} 프로그램을 삭제하시겠습니까?`);
      if (!confirmed) {
        return;
      }
    }

    if (action === "start") {
      await withProgramAction(id, () => startProgram(id));
      return;
    }

    if (action === "stop") {
      await withProgramAction(id, () => stopProgram(id));
      return;
    }

    if (action === "delete") {
      await withProgramAction(id, () => deleteProgram(id));
    }
  });

  renderPrograms(dom, state);

  const refreshIntervalMs = options.refreshIntervalMs ?? 2000;
  if (refreshIntervalMs > 0) {
    refreshTimerId = setInterval(() => {
      void refreshPrograms({ keepError: true });
    }, refreshIntervalMs);
  }

  const ready = refreshPrograms();

  return {
    dom,
    state,
    ready,
    refreshPrograms,
    dispose() {
      if (refreshTimerId !== null) {
        clearInterval(refreshTimerId);
        refreshTimerId = null;
      }
    },
  };
}
