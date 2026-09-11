package main

import (
	"context"

	"OpenFlux/pkg/autostart"
	"OpenFlux/pkg/config"
	"OpenFlux/pkg/core"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	_ = core.Get().Stop()
}

func (a *App) GetConfig() config.Config {
	cfg := config.Load()
	cfg.AutoStart = autostart.IsEnabled()
	return cfg
}

func (a *App) SaveConfig(cfg config.Config) error {
	if err := config.Save(cfg); err != nil {
		return err
	}
	_ = autostart.Set(cfg.AutoStart)
	return nil
}

func (a *App) Connect() error {
	cfg := config.Get()
	return core.Get().Start(cfg)
}

func (a *App) Disconnect() error {
	return core.Get().Stop()
}

func (a *App) GetStatus() core.ConnectionStatus {
	return core.Get().GetStatus()
}

func (a *App) GetLogs() string {
	return core.Get().GetLogs()
}
