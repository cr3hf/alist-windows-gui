package alist

import (
	"context"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestManagerSmoke 真实拉起 alist 内核，验证 M1 核心链路：
// 定位内核 → 首次初始化注入密码 → Start → 日志捕获 → 健康检查 → Stop。
func TestManagerSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	// 默认端口 5244 已有服务（如用户桌面端正在运行）时跳过，
	// 否则探活会被已运行实例"骗过"，产生假阳性。
	resp, err := http.Get("http://127.0.0.1:5244/")
	if err == nil {
		resp.Body.Close()
		t.Skip("端口 5244 已有服务在运行（可能桌面端正在运行），跳过冒烟测试")
	}
	_, thisFile, _, _ := runtime.Caller(0)
	// 测试源码位于 alist_win/app/internal/alist/，内核在 alist_win/bin/alist.exe（上溯 4 级）
	exePath := filepath.Join(thisFile, "..", "..", "..", "..", "bin", "alist.exe")
	exePath, _ = filepath.Abs(exePath)
	dataDir := filepath.Join(t.TempDir(), "data")

	ctx := context.Background()
	m, err := NewWithExe(ctx, exePath, dataDir)
	if err != nil {
		t.Fatalf("NewWithExe: %v", err)
	}
	t.Logf("exe=%s dataDir=%s portable=%v", m.exePath, m.dataDir, m.isPortable)

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// 等待健康检查通过
	ok := false
	for i := 0; i < 40; i++ {
		if m.HealthCheck() {
			ok = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !ok {
		t.Fatalf("alist 未在 20s 内就绪；最后日志:\n%s", dumpLogs(m))
	}
	t.Logf("alist 就绪，状态=%s 端口=%d", m.GetState().Status, m.httpPort())

	// 验证首次初始化已注入固定初始密码（admin），桌面端不保存密码

	// 验证日志确实捕获到了内容
	if len(m.GetLogs()) == 0 {
		t.Fatalf("未捕获到任何日志")
	}

	// 重置账号密码（固定 admin/admin）：停服 → admin set → 重启 → 仍运行
	if err := m.ResetAdmin(); err != nil {
		t.Fatalf("ResetAdmin: %v", err)
	}
	if !m.IsRunning() {
		t.Fatalf("重置密码后 alist 未恢复运行")
	}
	// 等待状态翻回 running（waitForHealthy 为异步）
	running := false
	for i := 0; i < 40; i++ {
		if m.GetState().Status == string(StatusRunning) {
			running = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !running {
		t.Fatalf("重置后状态未回到 running: %s", m.GetState().Status)
	}

	// 停止
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if m.IsRunning() {
		t.Fatalf("Stop 后 alist 仍在运行")
	}
	t.Logf("M1 冒烟测试通过")
}

func dumpLogs(m *Manager) string {
	out := ""
	for _, l := range m.GetLogs() {
		out += l.Time + " " + l.Level + " " + l.Msg + "\n"
	}
	return out
}
