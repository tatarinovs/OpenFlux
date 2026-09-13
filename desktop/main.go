package main

import (
	"context"
	"embed"
	_ "embed"

	"OpenFlux/pkg/config"
	"OpenFlux/pkg/core"
	"os"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIconBytes []byte

var (
	appContext context.Context
	mToggle    *systray.MenuItem
)

func setupSystray() {
	systray.Run(func() {
		systray.SetIcon(trayIconBytes)
		systray.SetTitle("OpenFlux")
		systray.SetTooltip("OpenFlux - Стелс-туннель")

		mShow := systray.AddMenuItem("Показать окно", "Развернуть главное окно")
		mToggle = systray.AddMenuItem("Подключить", "Включить / Отключить туннель")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Выход", "Закрыть OpenFlux")

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					if appContext != nil {
						runtime.WindowShow(appContext)
					}
				case <-mToggle.ClickedCh:
					if core.Get().IsRunning() {
						_ = core.Get().Stop()
						mToggle.SetTitle("Подключить")
						systray.SetTooltip("OpenFlux - Отключен")
					} else {
						cfg := config.Get()
						_ = core.Get().Start(cfg)
						mToggle.SetTitle("Отключить")
						systray.SetTooltip("OpenFlux - Подключен")
					}
				case <-mQuit.ClickedCh:
					_ = core.Get().Stop()
					systray.Quit()
					if appContext != nil {
						runtime.Quit(appContext)
					}
					return
				}
			}
		}()
	}, func() {
		_ = core.Get().Stop()
	})
}

func main() {
	app := NewApp()
	cfg := config.Load()

	var exitNodeFlag bool
	for _, arg := range os.Args[1:] {
		if arg == "--exit-node" || arg == "-exit-node" || arg == "--exitnode" {
			exitNodeFlag = true
		}
	}

	startHidden := cfg.StartMinimized
	if exitNodeFlag {
		cfg.Mode = "exitnode"
		startHidden = true
	}

	err := wails.Run(&options.App{
		Title:             "OpenFlux",
		Width:             880,
		Height:            680,
		MinWidth:          800,
		MinHeight:         620,
		StartHidden:       startHidden,
		HideWindowOnClose: false,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			currentCfg := config.Get()
			if currentCfg.CloseToTray || exitNodeFlag {
				runtime.WindowHide(ctx)
				return true
			}
			return false
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 1}, // Slate-900
		OnStartup: func(ctx context.Context) {
			appContext = ctx
			app.startup(ctx)
			go setupSystray()

			if exitNodeFlag {
				_ = config.Save(cfg)
				go func() {
					time.Sleep(500 * time.Millisecond)
					_ = core.Get().Start(cfg)
					if mToggle != nil {
						mToggle.SetTitle("Отключить")
						systray.SetTooltip("OpenFlux - Выходная нода активна")
					}
				}()
			}
		},
		OnShutdown: func(ctx context.Context) {
			systray.Quit()
			app.shutdown(ctx)
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "openflux-windows-desktop-instance",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				if appContext != nil {
					runtime.WindowShow(appContext)
				}
			},
		},
		Windows: &windows.Options{
			BackdropType:         windows.Mica,
			Theme:                windows.SystemDefault,
			WebviewIsTransparent: true,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
