// Package tray 基于 energye/systray（getlantern fork）实现系统托盘常驻。
// 选型原因：getlantern/systray v1.2.2 在 Windows 把左键/右键消息一律弹菜单
// （systray_windows.go: case WM_RBUTTONUP, WM_LBUTTONUP），无法实现
// "左键回主界面"；energye fork 区分 WM_LBUTTONUP/WM_RBUTTONUP 并提供
// SetOnClick / SetOnRClick 回调。
// 注意：Wails v2 无内置托盘（F12）；本包在自己的 goroutine 中运行独立消息循环，
// 与 Wails 主窗口消息循环分属不同线程窗口（R2 风险点，需真机验证）。
package tray

import (
	"sync"

	"alistwin/internal/trayicon"

	"github.com/energye/systray"
)

// Controller 由 App 实现，托盘菜单通过它驱动核心能力。
// 所有方法必须可在非主线程 goroutine 中安全调用。
type Controller interface {
	TrayStart()          // 启动 alist
	TrayStop()           // 停止 alist
	TrayRestart()        // 重启 alist
	TrayOpenBrowser()    // 浏览器打开
	TrayShowWindow()     // 显示主窗口
	TrayHideWindow()     // 隐藏主窗口（单击托盘切换）
	TrayToggleAutoStart() bool // 切换开机自启，返回切换后的状态
	TrayIsAutoStart() bool
	TrayOpenDataDir()    // 打开数据目录
	TrayOpenLogsWindow() // 打开日志窗口
	TrayAbout()          // 关于
	TrayQuit()           // 退出（停服 + 退出应用）
	TrayState() (status string, url string, account string)
}

var (
	ctrl      Controller
	startOnce sync.Once
	muState   sync.Mutex

	miStart   *systray.MenuItem
	miStop    *systray.MenuItem
	miAuto    *systray.MenuItem
	miURL     *systray.MenuItem
)

// Start 启动托盘（幂等）。必须在 Wails OnStartup 之后调用（需要可用 ctx）。
func Start(c Controller) {
	ctrl = c
	startOnce.Do(func() {
		go systray.Run(onReady, onExit)
	})
}

// Quit 退出托盘消息循环。
func Quit() { systray.Quit() }

// SyncState 根据最新状态刷新托盘菜单文案（由 App 在状态变化时调用）。
func SyncState() {
	if ctrl == nil {
		return
	}
	muState.Lock()
	defer muState.Unlock()
	status, url, _ := ctrl.TrayState()
	if miStart != nil && miStop != nil {
		switch status {
		case "running", "starting":
			miStart.Disable()
			miStop.Enable()
			miStop.SetTitle("停止")
		default:
			miStart.Enable()
			miStop.Disable()
		}
	}
	if miURL != nil {
		miURL.SetTitle("地址：" + url + "（点击复制）")
	}
	if miAuto != nil {
		if ctrl.TrayIsAutoStart() {
			miAuto.Check()
		} else {
			miAuto.Uncheck()
		}
	}
}

func onReady() {
	systray.SetIcon(trayicon.Icon())
	systray.SetTitle("AList 桌面版")
	systray.SetTooltip("AList 桌面版 — alist 本地服务管理")

	// 左键：回到主界面（显示并聚焦窗口）
	systray.SetOnClick(func(_ systray.IMenu) { ctrl.TrayShowWindow() })
	// 右键：弹出菜单（energye fork 默认不展示菜单，必须显式调用 ShowMenu）
	systray.SetOnRClick(func(menu systray.IMenu) { _ = menu.ShowMenu() })

	miStart = systray.AddMenuItem("启动", "启动 alist 服务")
	miStop = systray.AddMenuItem("停止", "停止 alist 服务")
	miRestart := systray.AddMenuItem("重启", "重启 alist 服务")
	miOpen := systray.AddMenuItem("在浏览器中打开", "打开 alist 管理页面")
	systray.AddSeparator()
	miURL = systray.AddMenuItem("地址：-", "点击复制访问地址")
	miAccount := systray.AddMenuItem("账号：admin", "点击复制账号")
	systray.AddSeparator()
	miLogs := systray.AddMenuItem("查看日志", "打开日志窗口")
	miData := systray.AddMenuItem("打开数据目录", "打开 alist 数据目录")
	miAuto = systray.AddMenuItemCheckbox("开机自启", "开机自动启动（静默到托盘）", false)
	systray.AddSeparator()
	miAbout := systray.AddMenuItem("关于", "版本与开源信息")
	miQuit := systray.AddMenuItem("退出", "停止 alist 并退出")

	miURL.Disable()
	miAccount.Disable()

	// energye/systray 用 item.Click(fn) 注册菜单回调（v1.0.3 无 ClickedCh 通道）
	miStart.Click(func() { ctrl.TrayStart() })
	miStop.Click(func() { ctrl.TrayStop() })
	miRestart.Click(func() { ctrl.TrayRestart() })
	miOpen.Click(func() { ctrl.TrayOpenBrowser() })
	miURL.Click(func() { ctrl.TrayOpenBrowser() })
	miLogs.Click(func() { ctrl.TrayOpenLogsWindow() })
	miData.Click(func() { ctrl.TrayOpenDataDir() })
	miAuto.Click(func() {
		ctrl.TrayToggleAutoStart()
		SyncState()
	})
	miAbout.Click(func() {
		ctrl.TrayShowWindow()
		ctrl.TrayAbout()
	})
	miQuit.Click(func() { ctrl.TrayQuit() })

	// 订阅后端状态推送，动态刷新托盘文案
	// （EventsOn 在 trayicon 包不可用，这里由 App.SyncTray 主动调用 SyncState）
	SyncState()
}

func onExit() {
	// 清理托盘资源（systray 自身处理）
}
