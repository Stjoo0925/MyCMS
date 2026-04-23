package programs

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func detectKind(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".bat":
		return KindBatch, nil
	case ".cmd":
		return KindCommand, nil
	case ".exe":
		return KindExecutable, nil
	case ".ps1":
		return KindPowerShell, nil
	case ".py":
		return KindPython, nil
	case ".js":
		return KindNode, nil
	case ".mjs":
		return KindNode, nil
	case ".cjs":
		return KindNode, nil
	case ".jar":
		return KindJar, nil
	default:
		return "", fmt.Errorf("경로는 지원되는 실행 파일 또는 스크립트를 가리켜야 합니다")
	}
}

func buildCommand(entry Entry) (*exec.Cmd, bool, error) {
	if entry.RunAsAdmin {
		return buildElevatedCommand(entry)
	}

	return buildStandardCommand(entry)
}

func buildStandardCommand(entry Entry) (*exec.Cmd, bool, error) {
	args := append([]string(nil), entry.Args...)

	switch entry.Kind {
	case KindBatch, KindCommand:
		return buildBatchCommand(entry, args), false, nil
	case KindExecutable:
		cmd := exec.Command(entry.Path, args...)
		cmd.Dir = entry.WorkingDirectory
		return cmd, false, nil
	case KindPowerShell:
		cmd := exec.Command("powershell.exe", append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", entry.Path}, args...)...)
		cmd.Dir = entry.WorkingDirectory
		return cmd, false, nil
	case KindPython:
		cmd := exec.Command("python", append([]string{entry.Path}, args...)...)
		cmd.Dir = entry.WorkingDirectory
		return cmd, false, nil
	case KindNode:
		cmd := exec.Command("node", append([]string{entry.Path}, args...)...)
		cmd.Dir = entry.WorkingDirectory
		return cmd, false, nil
	case KindJar:
		cmd := exec.Command("java", append([]string{"-jar", entry.Path}, args...)...)
		cmd.Dir = entry.WorkingDirectory
		return cmd, false, nil
	default:
		return nil, false, fmt.Errorf("지원되지 않는 프로그램 종류입니다: %q", entry.Kind)
	}
}

func buildElevatedCommand(entry Entry) (*exec.Cmd, bool, error) {
	filePath, args, err := elevatedTarget(entry)
	if err != nil {
		return nil, false, err
	}

	argList := make([]string, 0, len(args))
	for _, arg := range args {
		argList = append(argList, "'"+escapePowerShellSingleQuoted(arg)+"'")
	}

	command := fmt.Sprintf(
		"Start-Process -FilePath '%s' -ArgumentList @(%s) -WorkingDirectory '%s' -Verb RunAs -Wait",
		escapePowerShellSingleQuoted(filePath),
		strings.Join(argList, ","),
		escapePowerShellSingleQuoted(entry.WorkingDirectory),
	)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", command)
	cmd.Dir = entry.WorkingDirectory
	return cmd, true, nil
}

func elevatedTarget(entry Entry) (string, []string, error) {
	switch entry.Kind {
	case KindBatch, KindCommand:
		return "cmd.exe", batchCommandArgs(entry.Path, entry.Args), nil
	case KindExecutable:
		return entry.Path, append([]string(nil), entry.Args...), nil
	case KindPowerShell:
		return "powershell.exe", append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", entry.Path}, entry.Args...), nil
	case KindPython:
		return "python", append([]string{entry.Path}, entry.Args...), nil
	case KindNode:
		return "node", append([]string{entry.Path}, entry.Args...), nil
	case KindJar:
		return "java", append([]string{"-jar", entry.Path}, entry.Args...), nil
	default:
		return "", nil, fmt.Errorf("지원되지 않는 프로그램 종류입니다: %q", entry.Kind)
	}
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
