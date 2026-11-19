//go:build !windows && !linux

package app

import (
	"os"

	"lootnest/internal/core"
)

func systemEnv(a *App) *core.Env {
	return &core.Env{Getenv: os.Getenv, Home: os.Getenv("HOME"), DataDir: DataDir()}
}

func systemDataDir() string { return "" }
