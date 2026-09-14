// Package tray 基于 energye/systray（getlantern fork）实现系统托盘常驻。
// 选型原因：getlantern/systray v1.2.2 在 Windows 把左键/右键消息一律弹菜单
// （systray_windows.go: case WM_RBUTTONUP, WM_LBUTTONUP），无法实现
// "左键回主界面"；energye fork 区分 WM_LBUTTONUP/WM_RBUTTONUP 并提供
// SetOnClick / SetOnRClick 回调。
// 注意：Wails v2 无内置托盘（F12）；本包在自己的 goroutine 中运行独立消息循环，
// 与 Wails 主窗口消息循环分属不同线程窗口（R2 风险点，需真机验证）。
package tray

import (
	"runtime"
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
		go func() {
			// 关键修复：Win32 的窗口消息按“线程”排队，消息循环 goroutine 必须绑定到
			// 固定的 OS 线程。systray 只在包 init() 里对“主 goroutine”调用过
			// LockOSThread，而此处 Run 跑在新建的 goroutine 上；若不显式锁定，
			// Go 调度器可能把整个消息循环迁移到另一个 OS 线程（系统休眠唤醒后线程
			// 被大规模重排时极易触发）。迁移之后 GetMessage 再也收不到托盘窗口的
			// 消息 —— 表现为“托盘图标还在，但点击毫无反应”，而 alist 是独立子进程，
			// 服务照常运行，与本现象完全吻合。
			// 注意：这里不能 defer UnlockOSThread()，必须全程保持锁定。
			runtime.LockOSThread()
			systray.Run(onReady, onExit)
		}()
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
	// 必须在独立 goroutine 中执行，原因有二：
	//  1) 回调是同步在 WndProc（即消息循环线程）里调用的，任何阻塞都会让整个托盘卡死；
	//  2) Wails 的 runtime.WindowShow 内部会 runtime.LockOSThread() +
	//     defer runtime.UnlockOSThread()；若在托盘线程上执行，它 defer 的 Unlock
	//     会解除本 goroutine 的线程绑定，重新引入“消息循环被迁走”的 Bug。
	systray.SetOnClick(func(_ systray.IMenu) { go ctrl.TrayShowWindow() })
	// 右键：弹出菜单。ShowMenu 是模态 Win32 操作且隶属于托盘窗口，须在托盘线程上执行。
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

	// energye/systray 用 item.Click(fn) 注册菜单回调（v1.0.3 无 ClickedCh 通道）。
	// 同样一律放到独立 goroutine 执行：菜单回调也是在 WndProc（消息循环线程）里同步
	// 派发的（WM_COMMAND → systrayMenuItemSelected），而 TrayStart/TrayRestart 这类
	// 操作可能耗时数十秒（健康检查最长 20s），直接执行会冻住整个托盘；同时也避免
	// Wails runtime 的 UnlockOSThread 在托盘线程上解除线程绑定。
	miStart.Click(func() { go ctrl.TrayStart() })
	miStop.Click(func() { go ctrl.TrayStop() })
	miRestart.Click(func() { go ctrl.TrayRestart() })
	miOpen.Click(func() { go ctrl.TrayOpenBrowser() })
	miURL.Click(func() { go ctrl.TrayOpenBrowser() })
	miLogs.Click(func() { go ctrl.TrayOpenLogsWindow() })
	miData.Click(func() { go ctrl.TrayOpenDataDir() })
	miAuto.Click(func() {
		go func() {
			ctrl.TrayToggleAutoStart()
			// 菜单状态更新由 systray 内部互斥锁（muMenus 等）保护，可跨 goroutine 调用
			SyncState()
		}()
	})
	miAbout.Click(func() {
		go func() {
			ctrl.TrayShowWindow()
			ctrl.TrayAbout()
		}()
	})
	miQuit.Click(func() { go ctrl.TrayQuit() })

	// 订阅后端状态推送，动态刷新托盘文案
	// （EventsOn 在 trayicon 包不可用，这里由 App.SyncTray 主动调用 SyncState）
	SyncState()
}

func onExit() {
	// 清理托盘资源（systray 自身处理）
}
