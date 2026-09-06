//go:build windows

// Package singleinstance 基于 Windows 命名互斥量实现单实例。
// 重复启动时：通知已有实例把窗口带到前台，然后本进程退出。
package singleinstance

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const mutexName = `Global\alist_win_single_instance`

// Acquire 尝试持有单实例互斥量。
// 返回 (true, release) 表示成为唯一实例；release 在退出时调用。
// 返回 (false, nil) 表示已有实例在运行（此时已尝试唤起其窗口）。
func Acquire() (bool, func()) {
	h, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName))
	if h != 0 && err == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(h)
		showExistingWindow()
		return false, nil
	}
	if err != nil && err != windows.ERROR_ALREADY_EXISTS {
		// 创建失败（权限等）：宁可降级为"允许多开"，也不阻断启动
		return true, func() {}
	}
	return true, func() { _ = windows.CloseHandle(h) }
}

var (
	modUser32        = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW  = modUser32.NewProc("FindWindowW")
	procShowWindow   = modUser32.NewProc("ShowWindow")
	procSetForegrnd  = modUser32.NewProc("SetForegroundWindow")
	procIsIconic     = modUser32.NewProc("IsIconic")
)

const swRestore = 9

// showExistingWindow 按窗口标题查找已有实例主窗口并前置显示。
func showExistingWindow() {
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(windowTitle))))
	if hwnd == 0 {
		return
	}
	iconic, _, _ := procIsIconic.Call(hwnd)
	if iconic != 0 {
		_, _, _ = procShowWindow.Call(hwnd, uintptr(swRestore))
	}
	_, _, _ = procSetForegrnd.Call(hwnd)
}

// windowTitle 与 main.go 的 options.App.Title 保持一致。
const windowTitle = "AList 桌面版"
