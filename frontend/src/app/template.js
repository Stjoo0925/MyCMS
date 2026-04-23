export function mountApplication(root) {
  root.innerHTML = `
    <main class="shell">
      <header class="topbar">
        <div class="topbar-copy">
          <p class="eyebrow">명령줄 프로그램 관리자 대시보드</p>
          <h1>MyCMS</h1>
        </div>
        <button id="create-button" type="button" class="btn-primary">프로그램 추가</button>
      </header>

      <div id="page-error" class="banner hidden" role="alert"></div>

      <section class="summary-strip" aria-label="프로그램 개요">
        <article class="summary-card">
          <span class="summary-label">실행 중</span>
          <strong id="summary-running">0</strong>
        </article>
        <article class="summary-card">
          <span class="summary-label">중지됨</span>
          <strong id="summary-stopped">0</strong>
        </article>
        <article class="summary-card">
          <span class="summary-label">확인 필요</span>
          <strong id="summary-attention">0</strong>
        </article>
      </section>

      <section class="board-head">
        <div>
          <p class="section-kicker">프로그램 개요</p>
          <h2 id="board-title">현재 프로그램</h2>
        </div>
        <span id="program-count" class="count-chip">0개</span>
      </section>

      <section id="programs" class="programs" aria-live="polite"></section>

      <div id="panel-backdrop" class="panel-backdrop hidden"></div>
      <aside id="panel" class="editor-panel hidden" aria-label="프로그램 편집기">
        <div class="panel-head">
          <div>
            <p class="section-kicker">프로그램 편집기</p>
            <h2 id="panel-title">프로그램 추가</h2>
            <p id="panel-description" class="panel-description">실행 파일을 등록하고 상태를 계속 확인할 수 있습니다.</p>
            <p class="panel-tip">기본적으로 이름과 실행 파일 경로만 입력하면 됩니다.</p>
          </div>
          <button id="cancel-edit" class="ghost-action" type="button">닫기</button>
        </div>

        <form id="program-form" class="form">
          <label class="field">
            <span>이름</span>
            <input id="name-input" name="name" type="text" autocomplete="off" placeholder="식별하기 쉬운 이름" required />
          </label>

          <label class="field">
            <span>실행 파일 경로</span>
            <div class="path-row">
              <input id="path-input" name="path" type="text" autocomplete="off" placeholder="C:\\tools\\runner.bat" required />
              <button id="choose-path" class="btn-secondary" type="button">파일 선택</button>
            </div>
            <small class="field-hint">배치 파일, 스크립트, exe 모두 등록할 수 있습니다.</small>
          </label>

          <details id="advanced-options" class="advanced-fields">
            <summary class="advanced-summary">고급 실행 옵션</summary>
            <p class="advanced-copy">평소에는 닫아 두고, 실행 조건을 조정할 때만 여세요.</p>

            <label class="field">
              <span>작업 폴더</span>
              <input id="working-directory-input" name="workingDirectory" type="text" autocomplete="off" placeholder="비워 두면 실행 파일 기준으로 추정" />
            </label>

            <label class="field">
              <span>인수</span>
              <textarea id="args-input" name="args" rows="3" autocomplete="off" placeholder="한 줄에 하나씩 입력"></textarea>
            </label>

            <label class="field">
              <span>환경 변수</span>
              <textarea id="env-input" name="env" rows="4" autocomplete="off" placeholder="KEY=value"></textarea>
            </label>

            <label class="field checkbox-field">
              <input id="run-as-admin-input" name="runAsAdmin" type="checkbox" />
              <span>관리자 권한으로 실행</span>
            </label>

            <label class="field">
              <span>재시작 정책</span>
              <select id="restart-policy-input" name="restartPolicy">
                <option value="none">없음</option>
                <option value="on-failure">실패 시</option>
                <option value="always">항상</option>
              </select>
            </label>

            <label class="field">
              <span>재시작 제한</span>
              <input id="restart-limit-input" name="restartLimit" type="number" min="0" step="1" value="0" />
            </label>

            <label class="field">
              <span>재시작 지연 초</span>
              <input id="restart-delay-input" name="restartDelaySeconds" type="number" min="0" step="1" value="0" />
            </label>

            <label class="field">
              <span>설명</span>
              <textarea id="description-input" name="description" rows="3" autocomplete="off" placeholder="짧은 설명"></textarea>
            </label>

            <label class="field">
              <span>메모</span>
              <textarea id="notes-input" name="notes" rows="3" autocomplete="off" placeholder="추가 메모"></textarea>
            </label>

            <label class="field">
              <span>태그</span>
              <input id="tags-input" name="tags" type="text" autocomplete="off" placeholder="ops, nightly, backup" />
            </label>
          </details>

          <div id="form-error" class="field-error hidden" role="alert"></div>

          <div class="form-actions">
            <button id="submit-button" type="submit" class="btn-primary">저장</button>
          </div>
        </form>
      </aside>
    </main>
  `;

  return {
    pageErrorEl: root.querySelector("#page-error"),
    programsEl: root.querySelector("#programs"),
    programCountEl: root.querySelector("#program-count"),
    summaryRunningEl: root.querySelector("#summary-running"),
    summaryStoppedEl: root.querySelector("#summary-stopped"),
    summaryAttentionEl: root.querySelector("#summary-attention"),
    panelEl: root.querySelector("#panel"),
    panelBackdropEl: root.querySelector("#panel-backdrop"),
    panelTitleEl: root.querySelector("#panel-title"),
    panelDescriptionEl: root.querySelector("#panel-description"),
    formEl: root.querySelector("#program-form"),
    formTitleEl: root.querySelector("#panel-title"),
    formErrorEl: root.querySelector("#form-error"),
    submitButtonEl: root.querySelector("#submit-button"),
    cancelEditButtonEl: root.querySelector("#cancel-edit"),
    createButtonEl: root.querySelector("#create-button"),
    choosePathButtonEl: root.querySelector("#choose-path"),
    advancedOptionsEl: root.querySelector("#advanced-options"),
    nameInputEl: root.querySelector("#name-input"),
    descriptionInputEl: root.querySelector("#description-input"),
    notesInputEl: root.querySelector("#notes-input"),
    tagsInputEl: root.querySelector("#tags-input"),
    pathInputEl: root.querySelector("#path-input"),
    workingDirectoryInputEl: root.querySelector("#working-directory-input"),
    argsInputEl: root.querySelector("#args-input"),
    envInputEl: root.querySelector("#env-input"),
    runAsAdminInputEl: root.querySelector("#run-as-admin-input"),
    restartPolicyEl: root.querySelector("#restart-policy-input"),
    restartLimitInputEl: root.querySelector("#restart-limit-input"),
    restartDelaySecondsInputEl: root.querySelector("#restart-delay-input"),
  };
}
