package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"alistwin/internal/alist"
	"alistwin/internal/tray"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 版本信息（M6：与 NSIS / 便携包保持一致）。
const (
	productVersion = "1.0.0"
	kernelVersion  = "v3.64.0"
	productName    = "AList 桌面版"
	upstreamURL    = "https://github.com/AlistGo/alist"
	licenseName    = "AGPL-3.0"
)

// App 结构体：前端通过它调用后端能力；同时实现 tray.Controller 供托盘菜单驱动。
type App struct {
	ctx      context.Context
	manager  *alist.Manager
	silent   bool
	quitting bool // 托盘"退出"置位后，OnBeforeClose 放行真正关闭
}

// NewApp 创建应用实例。
func NewApp() *App { return &App{} }

// startup 在应用启动时调用：初始化管理器、托盘，处理静默启动。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	for _, arg := range os.Args {
		if strings.TrimSpace(arg) == "--silent" {
			a.silent = true
		}
	}
	m, err := alist.New(ctx)
	if err != nil {
		println("alist 初始化失败:", err.Error())
		return
	}
	a.manager = m

	// 后端状态变化 → 刷新托盘文案
	runtime.EventsOn(ctx, "alist:state", func(_ ...interface{}) {
		tray.SyncState()
	})

	tray.Start(a)

	// 静默模式（开机自启）：隐藏窗口到托盘。
	if a.silent {
		runtime.WindowHide(ctx)
	}

	// 自动启动服务：--silent 或设置开启时拉起 alist。
	// 延迟 1s 再启动，避免与主窗口/托盘初始化抢占资源导致启动不稳定（用户实测反馈）。
	if a.silent || m.GetSettings().AutoStartService {
		go func() {
			time.Sleep(1 * time.Second)
			_ = m.Start()
		}()
	}
}

// shutdown 应用退出：停托盘、停 alist。
func (a *App) shutdown(ctx context.Context) {
	tray.Quit()
	if a.manager != nil {
		_ = a.manager.Stop()
	}
}

// beforeClose 关闭窗口 = 隐藏到托盘（M4）；仅托盘"退出"才真正结束。
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

// --- 前端绑定 ---

// IsReady 管理器是否就绪（alist.exe 已定位）。
func (a *App) IsReady() bool { return a.manager != nil }

// Start 启动 alist。
func (a *App) Start() error {
	if a.manager == nil {
		return errNotReady
	}
	return a.manager.Start()
}

// Stop 停止 alist。
func (a *App) Stop() error {
	if a.manager == nil {
		return errNotReady
	}
	return a.manager.Stop()
}

// Restart 重启 alist。
func (a *App) Restart() error {
	if a.manager == nil {
		return errNotReady
	}
	return a.manager.Restart()
}

// OpenBrowser 在默认浏览器打开 alist 管理页面。
func (a *App) OpenBrowser() error {
	if a.manager == nil {
		return errNotReady
	}
	state := a.manager.GetState()
	if state.Status != string(alist.StatusRunning) {
		return errNotRunning
	}
	runtime.BrowserOpenURL(a.ctx, state.URL)
	return nil
}

// GetState 返回当前状态快照。
func (a *App) GetState() alist.State {
	if a.manager == nil {
		return alist.State{Status: string(alist.StatusError)}
	}
	return a.manager.GetState()
}

// GetLogs 返回日志环形缓冲。
func (a *App) GetLogs() []alist.LogLine {
	if a.manager == nil {
		return nil
	}
	return a.manager.GetLogs()
}

// ResetAdmin 将管理员账号密码重置为 admin / admin（桌面端不保存任何密码）。
func (a *App) ResetAdmin() error {
	if a.manager == nil {
		return errNotReady
	}
	return a.manager.ResetAdmin()
}

// SetAutoStart 设置/取消开机自启。
func (a *App) SetAutoStart(on bool) error { return alist.SetAutoStart(on) }

// SetAutoStartService 设置"打开软件后自动启动 alist 服务"。
func (a *App) SetAutoStartService(on bool) error {
	if a.manager == nil {
		return errNotReady
	}
	return a.manager.SetAutoStartService(on)
}

// ShowWindow 显示主窗口。
func (a *App) ShowWindow() { runtime.WindowShow(a.ctx) }

// HideWindow 隐藏主窗口。
func (a *App) HideWindow() { runtime.WindowHide(a.ctx) }

// QuitApp 真正退出（托盘"退出"调用；带二次确认在前端/托盘侧完成）。
func (a *App) QuitApp() {
	a.quitting = true
	tray.Quit()
	if a.manager != nil {
		_ = a.manager.Stop()
	}
	runtime.Quit(a.ctx)
}

// OpenDataDir 用资源管理器打开数据目录。
func (a *App) OpenDataDir() error {
	if a.manager == nil {
		return errNotReady
	}
	return exec.Command("explorer", a.manager.GetState().DataDir).Start()
}

// GetAbout 返回关于信息（版本/内核/许可证/上游地址）。
func (a *App) GetAbout() map[string]string {
	return map[string]string{
		"product":     productName,
		"version":     productVersion,
		"kernel":      kernelVersion,
		"upstream":    upstreamURL,
		"license":     licenseName,
		"thirdParty":  "alist v" + kernelVersion + " (AGPL-3.0) — 许可证全文见安装目录 bin/LICENSE-alist.txt",
		"disclaimer":  "本程序不附带任何激活/授权机制，也不保存任何密码；账号密码默认 admin/admin，可在网页端修改，如遗忘可用\"重置密码\"一键还原。",
	}
}

// IsPortable 是否便携模式。
func (a *App) IsPortable() bool {
	if a.manager == nil {
		return false
	}
	return a.manager.GetState().IsPortable
}

// --- tray.Controller 实现 ---

func (a *App) TrayStart() {
	if a.manager == nil {
		return
	}
	_ = a.manager.Start()
	tray.SyncState()
}

func (a *App) TrayStop() {
	if a.manager == nil {
		return
	}
	_ = a.manager.Stop()
	tray.SyncState()
}

func (a *App) TrayRestart() {
	if a.manager == nil {
		return
	}
	_ = a.manager.Restart()
	tray.SyncState()
}

func (a *App) TrayOpenBrowser() { _ = a.OpenBrowser() }

func (a *App) TrayShowWindow() { runtime.WindowShow(a.ctx) }

func (a *App) TrayHideWindow() { runtime.WindowHide(a.ctx) }

func (a *App) TrayToggleAutoStart() bool {
	cur := alist.IsAutoStart()
	_ = alist.SetAutoStart(!cur)
	return !cur
}

func (a *App) TrayIsAutoStart() bool { return alist.IsAutoStart() }

func (a *App) TrayOpenDataDir() { _ = a.OpenDataDir() }

func (a *App) TrayOpenLogsWindow() { runtime.WindowShow(a.ctx) }

func (a *App) TrayAbout() {
	runtime.WindowShow(a.ctx)
	runtime.EventsEmit(a.ctx, "app:about")
}

func (a *App) TrayQuit() { a.QuitApp() }

func (a *App) TrayState() (status string, url string, account string) {
	if a.manager == nil {
		return "stopped", "", "admin"
	}
	s := a.manager.GetState()
	return s.Status, s.URL, s.Account
}

var (
	errNotReady   = errorString("alist 未就绪（未找到 alist.exe）")
	errNotRunning = errorString("alist 尚未运行")
)

type errorString string

func (e errorString) Error() string { return string(e) }
