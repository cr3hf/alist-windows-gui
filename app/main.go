package main

import (
	"embed"
	"os"

	"alistwin/internal/singleinstance"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 单实例：重复启动时唤起已有窗口并退出本进程。
	// 注意：singleinstance.windowTitle 需与下方 Title 保持一致。
	if ok, release := singleinstance.Acquire(); !ok {
		os.Exit(0)
	} else {
		defer release()
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "AList 桌面版",
		Width:  860,
		Height: 680,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 246, G: 247, B: 249, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
