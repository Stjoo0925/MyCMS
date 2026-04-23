export function createState() {
  return {
    programs: [],
    pageError: "",
    formError: "",
    formMode: "create",
    editId: "",
    formBusy: false,
    choosing: false,
    pendingIds: new Set(),
    panelOpen: false,
    hasLoaded: false,
  };
}

export function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function setPageError(dom, state, message) {
  state.pageError = message ?? "";
  dom.pageErrorEl.textContent = state.pageError;
  dom.pageErrorEl.classList.toggle("hidden", state.pageError === "");
}

export function setFormError(dom, state, message) {
  state.formError = message ?? "";
  dom.formErrorEl.textContent = state.formError;
  dom.formErrorEl.classList.toggle("hidden", state.formError === "");
}

function writeProgramToForm(dom, program) {
  dom.nameInputEl.value = program?.name ?? "";
  dom.descriptionInputEl.value = program?.description ?? "";
  dom.notesInputEl.value = program?.notes ?? "";
  dom.tagsInputEl.value = (program?.tags ?? []).join(", ");
  dom.pathInputEl.value = program?.path ?? "";
  dom.workingDirectoryInputEl.value = program?.workingDirectory ?? "";
  dom.argsInputEl.value = (program?.args ?? []).join("\n");
  dom.envInputEl.value = (program?.env ?? [])
    .map((item) => `${item.key}=${item.value}`)
    .join("\n");
  dom.runAsAdminInputEl.checked = Boolean(program?.runAsAdmin);
  dom.restartPolicyEl.value = program?.restartPolicy ?? "none";
  dom.restartLimitInputEl.value = String(program?.restartLimit ?? 0);
  dom.restartDelaySecondsInputEl.value = String(
    program?.restartDelaySeconds ?? 0,
  );
}

export function setFormMode(dom, state, mode, program = null) {
  state.formMode = mode;
  state.editId = program?.id ?? "";
  state.panelOpen = true;
  dom.panelTitleEl.textContent =
    mode === "edit" ? "프로그램 수정" : "프로그램 추가";
  dom.panelDescriptionEl.textContent =
    mode === "edit"
      ? "상태 화면을 벗어나지 않고 이름이나 실행 경로를 수정할 수 있습니다."
      : "실행 파일을 등록하고 상태를 계속 확인할 수 있습니다.";
  dom.submitButtonEl.textContent =
    mode === "edit" ? "수정 저장" : "프로그램 저장";
  dom.cancelEditButtonEl.textContent = "닫기";
  dom.panelEl.classList.toggle("hidden", false);
  dom.panelBackdropEl.classList.toggle("hidden", false);

  if (program) {
    writeProgramToForm(dom, program);
  } else {
    dom.formEl.reset();
    dom.restartPolicyEl.value = "none";
    dom.restartLimitInputEl.value = "0";
    dom.restartDelaySecondsInputEl.value = "0";
    dom.runAsAdminInputEl.checked = false;
  }

  setFormError(dom, state, "");
}

export function closeFormPanel(dom, state) {
  state.panelOpen = false;
  state.formMode = "create";
  state.editId = "";
  dom.panelEl.classList.toggle("hidden", true);
  dom.panelBackdropEl.classList.toggle("hidden", true);
  dom.formEl.reset();
  dom.restartPolicyEl.value = "none";
  dom.restartLimitInputEl.value = "0";
  dom.restartDelaySecondsInputEl.value = "0";
  dom.runAsAdminInputEl.checked = false;
  setFormError(dom, state, "");
}

export function updateSummary(dom, state) {
  const running = state.programs.filter(
    (program) => program.status === "RUNNING",
  ).length;
  const stopped = state.programs.filter(
    (program) => program.status === "STOPPED",
  ).length;
  const attention = state.programs.filter(
    (program) => Boolean(program.lastError) || program.status === "ORPHANED",
  ).length;

  dom.summaryRunningEl.textContent = String(running);
  dom.summaryStoppedEl.textContent = String(stopped);
  dom.summaryAttentionEl.textContent = String(attention);
}

export function setPanelState(dom, state, open) {
  state.panelOpen = open;
  dom.panelEl.classList.toggle("hidden", !open);
  dom.panelBackdropEl.classList.toggle("hidden", !open);
}

export function setFormBusy(dom, state, isBusy) {
  state.formBusy = isBusy;
  const disabled = isBusy || state.choosing;
  dom.nameInputEl.disabled = disabled;
  dom.descriptionInputEl.disabled = disabled;
  dom.notesInputEl.disabled = disabled;
  dom.tagsInputEl.disabled = disabled;
  dom.pathInputEl.disabled = disabled;
  dom.workingDirectoryInputEl.disabled = disabled;
  dom.argsInputEl.disabled = disabled;
  dom.envInputEl.disabled = disabled;
  dom.runAsAdminInputEl.disabled = disabled;
  dom.restartPolicyEl.disabled = disabled;
  dom.restartLimitInputEl.disabled = disabled;
  dom.restartDelaySecondsInputEl.disabled = disabled;
  dom.submitButtonEl.disabled = disabled;
  dom.choosePathButtonEl.disabled = disabled;
  dom.cancelEditButtonEl.disabled = disabled;
  dom.createButtonEl.disabled = disabled;
}

export function markPending(state, id, isPending) {
  if (isPending) {
    state.pendingIds.add(id);
    return;
  }
  state.pendingIds.delete(id);
}

function normalizeStatus(status) {
  return String(status ?? "STOPPED").toUpperCase();
}

function getStatusLabel(status) {
  switch (normalizeStatus(status)) {
    case "RUNNING":
      return "실행 중";
    case "STARTING":
      return "시작 중";
    case "STOPPING":
      return "중지 중";
    case "ORPHANED":
      return "연결 끊김";
    default:
      return "중지됨";
  }
}

function isBusyStatus(status) {
  return (
    normalizeStatus(status) === "STARTING" ||
    normalizeStatus(status) === "STOPPING"
  );
}

function isRunningStatus(status) {
  return normalizeStatus(status) === "RUNNING";
}

function canStartStatus(status) {
  const current = normalizeStatus(status);
  return current === "STOPPED" || current === "ORPHANED";
}

function canEditStatus(status) {
  return !isBusyStatus(status) && !isRunningStatus(status);
}

function formatBytes(bytes) {
  const value = Number(bytes);
  if (!Number.isFinite(value) || value <= 0) {
    return "";
  }

  const units = ["B", "KB", "MB", "GB", "TB"];
  let current = value;
  let unitIndex = 0;

  while (current >= 1024 && unitIndex < units.length - 1) {
    current /= 1024;
    unitIndex += 1;
  }

  const display =
    current >= 10 || unitIndex === 0 ? current.toFixed(0) : current.toFixed(1);
  return `${display} ${units[unitIndex]}`;
}

function formatElapsed(startedAt) {
  const started = Date.parse(startedAt);
  if (Number.isNaN(started)) {
    return "";
  }

  const elapsed = Math.max(0, Date.now() - started);
  const totalSeconds = Math.floor(elapsed / 1000);
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (days > 0) {
    return `실행 ${days}일 ${hours}시간`;
  }
  if (hours > 0) {
    return `실행 ${hours}시간 ${minutes}분`;
  }
  if (minutes > 0) {
    return `실행 ${minutes}분 ${seconds}초`;
  }
  return `실행 ${seconds}초`;
}

export function renderPrograms(dom, state) {
  const items = state.programs;
  updateSummary(dom, state);
  dom.programCountEl.textContent = `${items.length}개`;

  if (!state.hasLoaded) {
    dom.programsEl.innerHTML = `
      <article class="program-skeleton"></article>
      <article class="program-skeleton"></article>
    `;
    return;
  }

  if (items.length === 0) {
    dom.programsEl.innerHTML = `
      <article class="empty-state">
        <p class="empty-kicker">등록된 프로그램이 없습니다</p>
        <h3>첫 프로그램을 추가해 보세요</h3>
        <p>이 화면에서 프로그램의 실행 여부를 바로 확인할 수 있도록 먼저 하나를 등록하세요.</p>
        <button type="button" data-action="open-create" class="btn-primary">첫 프로그램 추가</button>
      </article>
    `;
    return;
  }

  dom.programsEl.innerHTML = items
    .map((program) => {
      const status = normalizeStatus(program.status);
      const isPending = state.pendingIds.has(program.id);
      const isRunning = isRunningStatus(status);
      const isBusy = isPending || isBusyStatus(status);
      const metaParts = [];

      if (program.kind) {
        metaParts.push(
          `<span class="meta-pill">${escapeHtml(program.kind)}</span>`,
        );
      }
      if (program.restartPolicy) {
        metaParts.push(
          `<span class="meta-pill">재시작 ${escapeHtml(program.restartPolicy)}</span>`,
        );
      }
      if (program.runAsAdmin) {
        metaParts.push(`<span class="meta-pill">관리자 권한</span>`);
      }
      if (program.tags?.length) {
        metaParts.push(
          `<span class="meta-pill">태그 ${escapeHtml(program.tags.join(", "))}</span>`,
        );
      }
      if (program.pid > 0) {
        metaParts.push(`<span class="meta-pill">PID ${escapeHtml(program.pid)}</span>`);
      }
      if (program.startedAt && status === "RUNNING") {
        const runtime = formatElapsed(program.startedAt);
        if (runtime) {
          metaParts.push(`<span class="meta-pill">${escapeHtml(runtime)}</span>`);
        }
      }
      if (program.memoryWorkingSetBytes > 0) {
        const memory = formatBytes(program.memoryWorkingSetBytes);
        if (memory) {
          metaParts.push(`<span class="meta-pill">메모리 ${escapeHtml(memory)}</span>`);
        }
      }

      const attentionMarkup = program.lastError
        ? `<div class="card-error"><span class="error-label">확인 필요</span><p>${escapeHtml(program.lastError)}</p></div>`
        : "";

      return `
        <article class="program-card program-card-${status.toLowerCase()}">
          <div class="card-head">
            <div class="card-copy">
              <p class="card-kicker">프로그램</p>
              <h3>${escapeHtml(program.name)}</h3>
              <p class="path-text">${escapeHtml(program.path)}</p>
              <div class="meta-row">${metaParts.join("")}</div>
            </div>
            <span class="status status-${status.toLowerCase()}">${getStatusLabel(status)}</span>
          </div>

          ${attentionMarkup}

          <div class="card-actions">
            <button data-action="start" data-id="${escapeHtml(program.id)}" ${!canStartStatus(status) || isBusy ? "disabled" : ""} class="btn-primary">시작</button>
            <button data-action="stop" data-id="${escapeHtml(program.id)}" class="btn-secondary" ${!isRunning || isBusy ? "disabled" : ""}>중지</button>
            <button data-action="edit" data-id="${escapeHtml(program.id)}" class="ghost-action" ${!canEditStatus(status) ? "disabled" : ""}>수정</button>
            <button data-action="delete" data-id="${escapeHtml(program.id)}" class="btn-danger" ${!canEditStatus(status) ? "disabled" : ""}>삭제</button>
          </div>
        </article>
      `;
    })
    .join("");
}
