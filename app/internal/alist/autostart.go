package alist

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// --- 开机自启（HKCU Run，无需管理员权限，直接用注册表 API） ---

const (
	runKeyRoot = registry.CURRENT_USER
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	runVal     = "alist_win"
)

// IsAutoStart 读取本产品是否已注册开机自启（值存在且指向本 exe）。
func IsAutoStart() bool {
	exe, e := os.Executable()
	if e != nil {
		return false
	}
	exeNorm := strings.ToLower(strings.ReplaceAll(exe, "/", "\\"))
	k, err := registry.OpenKey(runKeyRoot, runKeyPath, registry.READ)
	if err != nil {
		return false
	}
	defer k.Close()
	val, _, err := k.GetStringValue(runVal)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(val), exeNorm)
}

// SetAutoStart 注册/取消开机自启（--silent 静默启动到托盘）。
func SetAutoStart(enable bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	k, _, err := registry.CreateKey(runKeyRoot, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open run key: %w", err)
	}
	defer k.Close()
	if !enable {
		// 删除键值（失败仅告警，不阻断）
		_ = k.DeleteValue(runVal)
		return nil
	}
	val := fmt.Sprintf("\"%s\" --silent", strings.ReplaceAll(exe, "/", "\\"))
	if err := k.SetStringValue(runVal, val); err != nil {
		return fmt.Errorf("set run value: %w", err)
	}
	return nil
}
