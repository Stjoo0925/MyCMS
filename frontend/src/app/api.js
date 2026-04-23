import {
  ChooseProgramPath,
  ChooseProgramTarget,
  ChooseWorkingDirectory,
  ClearProgramLogs,
  CreateProgram,
  DeleteProgram,
  GetProgram,
  GetProgramLogs,
  ListPrograms,
  ReconnectPrograms,
  StartProgram,
  StopProgram,
  UpdateProgram,
} from "../../wailsjs/go/main/App.js";

function assertWailsRuntimeAvailable() {
  const bridge = globalThis.window?.go?.main?.App;
  if (!bridge) {
    throw new Error("Wails 런타임을 찾을 수 없습니다. Wails 앱에서 이 화면을 열어 주세요.");
  }
}

export async function listPrograms(query = {}) {
  assertWailsRuntimeAvailable();
  return ListPrograms(query);
}

export async function getProgram(id) {
  assertWailsRuntimeAvailable();
  return GetProgram(id);
}

export async function createProgram(payload) {
  assertWailsRuntimeAvailable();
  return CreateProgram(payload);
}

export async function updateProgram(id, payload) {
  assertWailsRuntimeAvailable();
  return UpdateProgram(id, payload);
}

export async function deleteProgram(id) {
  assertWailsRuntimeAvailable();
  return DeleteProgram(id);
}

export async function startProgram(id) {
  assertWailsRuntimeAvailable();
  return StartProgram(id);
}

export async function stopProgram(id) {
  assertWailsRuntimeAvailable();
  return StopProgram(id);
}

export async function getProgramLogs(id, query = {}) {
  assertWailsRuntimeAvailable();
  return GetProgramLogs(id, query);
}

export async function clearProgramLogs(id) {
  assertWailsRuntimeAvailable();
  return ClearProgramLogs(id);
}

export async function reconnectPrograms() {
  assertWailsRuntimeAvailable();
  return ReconnectPrograms();
}

export async function chooseProgramPath() {
  assertWailsRuntimeAvailable();
  return ChooseProgramPath();
}

export async function chooseProgramTarget() {
  assertWailsRuntimeAvailable();
  return ChooseProgramTarget();
}

export async function chooseWorkingDirectory() {
  assertWailsRuntimeAvailable();
  return ChooseWorkingDirectory();
}

export function getErrorMessage(error) {
  return error?.message ?? String(error);
}
