package main

import (
	"context"
	"embed"
	"fmt"
	"os"

	"audioforge/internal/selftest"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version 是应用版本号。
const Version = "1.0.0"

//go:embed all:frontend/dist
var distFS embed.FS

func main() {
	// 无界面自检：AudioForge.exe --selftest
	for _, arg := range os.Args[1:] {
		if arg == "--selftest" {
			code := selftest.Run(os.Stdout)
			os.Exit(code)
		}
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "音频工坊 AudioForge",
		Width:            1200,
		Height:           840,
		MinWidth:         1000,
		MinHeight:        680,
		BackgroundColour: &options.RGBA{R: 11, G: 13, B: 18, A: 1},
		AssetServer: &assetserver.Options{
			Assets: distFS,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "audioforge-9f2c1d7e",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				runtime.WindowUnminimise(app.ctx)
				runtime.WindowShow(app.ctx)
			},
		},
		OnStartup:  app.startup,
		OnDomReady: app.domReady,
		OnShutdown: app.shutdown,
		Bind:       []interface{}{app},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "启动失败:", err)
	}
}

func (a *App) domReady(ctx context.Context) {
	runtime.OnFileDrop(ctx, a.onFileDrop)
}
