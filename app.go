package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"mycms/internal/programs"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	programs *programs.Service
	initErr  error
}

func NewApp() *App {
	configPath, err := programs.DefaultConfigPath()
	if err != nil {
		return &App{initErr: err}
	}

	runtimePath, err := programs.DefaultRuntimePath()
	if err != nil {
		return &App{initErr: err}
	}

	service, err := programs.NewService(
		programs.NewJSONStore(configPath),
		nil,
		programs.WithRuntimeStore(programs.NewJSONRuntimeStore(runtimePath)),
	)
	if err != nil {
		return &App{initErr: err}
	}

	return &App{
		programs: service,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.programs != nil && a.initErr == nil {
		if err := a.programs.ReconnectPrograms(); err != nil {
			a.initErr = err
		}
	}
}

func (a *App) ListPrograms(query programs.ListQuery) ([]programs.View, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.programs.ListPrograms(query)
}

func (a *App) GetProgram(id string) (programs.View, error) {
	if a.initErr != nil {
		return programs.View{}, a.initErr
	}
	return a.programs.GetProgram(id)
}

func (a *App) CreateProgram(input programs.Input) (programs.View, error) {
	if a.initErr != nil {
		return programs.View{}, a.initErr
	}
	return a.programs.CreateProgram(input)
}

func (a *App) UpdateProgram(id string, input programs.Input) (programs.View, error) {
	if a.initErr != nil {
		return programs.View{}, a.initErr
	}
	return a.programs.UpdateProgram(id, input)
}

func (a *App) DeleteProgram(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	return a.programs.DeleteProgram(id)
}

func (a *App) StartProgram(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	return a.programs.StartProgram(id)
}

func (a *App) StopProgram(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	return a.programs.StopProgram(id)
}

func (a *App) GetProgramLogs(id string, query programs.LogQuery) (programs.LogView, error) {
	if a.initErr != nil {
		return programs.LogView{}, a.initErr
	}
	return a.programs.GetProgramLogs(id, query)
}

func (a *App) ClearProgramLogs(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	return a.programs.ClearProgramLogs(id)
}

func (a *App) ReconnectPrograms() error {
	if a.initErr != nil {
		return a.initErr
	}
	return a.programs.ReconnectPrograms()
}

func (a *App) ChooseProgramPath() (string, error) {
	return a.ChooseProgramTarget()
}

func (a *App) ChooseProgramTarget() (string, error) {
	if a.initErr != nil {
		return "", a.initErr
	}
	if a.ctx == nil {
		return "", errors.New("앱 컨텍스트가 아직 준비되지 않았습니다")
	}

	defaultDirectory := ""
	if configPath, err := programs.DefaultConfigPath(); err == nil {
		defaultDirectory = filepath.Dir(configPath)
	}

	if defaultDirectory != "" {
		if _, err := os.Stat(defaultDirectory); err != nil {
			defaultDirectory = ""
		}
	}

	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "프로그램 실행 파일 선택",
		DefaultDirectory: defaultDirectory,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "지원 대상 (*.bat;*.cmd;*.exe;*.ps1;*.py;*.js;*.jar)",
				Pattern:     "*.bat;*.cmd;*.exe;*.ps1;*.py;*.js;*.jar",
			},
		},
	})
}

func (a *App) ChooseWorkingDirectory() (string, error) {
	if a.initErr != nil {
		return "", a.initErr
	}
	if a.ctx == nil {
		return "", errors.New("앱 컨텍스트가 아직 준비되지 않았습니다")
	}

	defaultDirectory := ""
	if configPath, err := programs.DefaultConfigPath(); err == nil {
		defaultDirectory = filepath.Dir(configPath)
	}

	if defaultDirectory != "" {
		if _, err := os.Stat(defaultDirectory); err != nil {
			defaultDirectory = ""
		}
	}

	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "작업 디렉터리 선택",
		DefaultDirectory: defaultDirectory,
	})
}
