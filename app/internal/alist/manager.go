package alist

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

// Status 表示 alist 子进程的运行状态。
type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusError    Status = "error"
)

const (
	maxLogLines      = 5000
	defaultHTTPPort  = 5244
	stopGracePeriod  = 5 * time.Second  // 优雅停止等待窗口
	noConsoleGrace   = 800 * time.Millisecond // 无控制台（GUI）时退而求其次的短暂等待
	healthTimeout    = 20 * time.Second
	alistAccount     = "admin"
	defaultPassword  = "admin" // 首启注入与重置目标：账号密码固定 admin/admin，桌面端不保存密码
)

// LogLine 一条 alist 日志（用于环形缓冲与前端实时推送）。
type LogLine struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

// State 返回给前端的快照。
type State struct {
	Status     string `json:"status"`
	IsPortable bool   `json:"isPortable"`
	AutoStart  bool   `json:"autoStart"`
	AutoStartService bool `json:"autoStartService"`
	Address    string `json:"address"`
	HTTPPort   int    `json:"httpPort"`
	HTTPSPort  int    `json:"httpsPort"`
	URL        string `json:"url"`
	Account    string `json:"account"`
	DataDir    string `json:"dataDir"`
	ExePath    string `json:"exePath"`
}

// Manager 管理 alist 子进程的生命周期。
type Manager struct {
	ctx        context.Context
	exePath    string
	dataDir    string
	isPortable bool

	mu           sync.Mutex
	cmd          *exec.Cmd
	status       Status
	logs         []LogLine
	cfg          *AlistConfig
	settings     Settings // 桌面端设置（desktop-settings.json）
	settingsPath string
	eventsOn     bool // 仅当持有合法的 Wails 生命周期 ctx 时才向前端推送事件
}

// New 构造 Manager：定位 alist.exe、解析数据目录。
func New(ctx context.Context) (*Manager, error) {
	exe, err := resolveAlistExe()
	if err != nil {
		return nil, err
	}
	dataDir, isPortable, err := resolveDataDir()
	if err != nil {
		return nil, err
	}
	return newManager(ctx, exe, dataDir, isPortable, true)
}

// NewWithExe 用显式内核路径构造 Manager（便于测试或自定义内核位置）。
// 该路径下不向前端推送事件（无合法 Wails ctx）。
func NewWithExe(ctx context.Context, exePath, dataDir string) (*Manager, error) {
	return newManager(ctx, exePath, dataDir, false, false)
}

func newManager(ctx context.Context, exe, dataDir string, isPortable, eventsOn bool) (*Manager, error) {
	if _, err := os.Stat(exe); err != nil {
		return nil, fmt.Errorf("alist.exe 不存在: %s", exe)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	settingsPath := filepath.Join(filepath.Dir(dataDir), "desktop-settings.json")
	m := &Manager{
		ctx:          ctx,
		exePath:      exe,
		dataDir:      dataDir,
		settingsPath: settingsPath,
		isPortable:   isPortable,
		eventsOn:     eventsOn,
		status:       StatusStopped,
		settings:     loadSettings(settingsPath),
	}
	if cfg, err := ReadConfig(dataDir); err == nil {
		m.cfg = cfg
	}
	return m, nil
}

// resolveAlistExe 依次尝试若干候选路径定位 alist.exe（含 dev 回退）。
func resolveAlistExe() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exe)
	cands := []string{
		filepath.Join(exeDir, "bin", "alist.exe"),
		filepath.Join(exeDir, "alist.exe"),
		filepath.Join(exeDir, "..", "bin", "alist.exe"),
		filepath.Join(exeDir, "..", "..", "bin", "alist.exe"),
		filepath.Join(exeDir, "..", "..", "..", "bin", "alist.exe"),
		filepath.Join(exeDir, "..", "..", "..", "..", "bin", "alist.exe"),
	}
	for _, c := range cands {
		if abs, err := filepath.Abs(c); err == nil {
			if fi, err := os.Stat(abs); err == nil && !fi.IsDir() {
				return abs, nil
			}
		}
	}
	return "", fmt.Errorf("alist.exe not found (searched near %s)", exeDir)
}

// resolveDataDir 解析数据目录：存在 exe 同目录 portable.ini 则为便携模式，
// 数据放 <exeDir>/data；否则为安装模式，放 %LOCALAPPDATA%/alist_win/data。
// 返回绝对路径（已实测：绝对 --data 可消除 alist 对 cwd 的相对路径依赖）。
func resolveDataDir() (string, bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", false, err
	}
	exeDir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(exeDir, "portable.ini")); err == nil {
		return filepath.Join(exeDir, "data"), true, nil
	}
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		local = os.Getenv("APPDATA")
	}
	if local == "" {
		local = os.TempDir()
	}
	return filepath.Join(local, "alist_win", "data"), false, nil
}

func (m *Manager) setState(s Status) {
	m.mu.Lock()
	m.status = s
	m.mu.Unlock()
	m.emitState()
}

func (m *Manager) emitState() {
	if !m.eventsOn {
		return
	}
	runtime.EventsEmit(m.ctx, "alist:state", m.GetState())
}

func (m *Manager) emitLog(line LogLine) {
	m.mu.Lock()
	m.logs = append(m.logs, line)
	if len(m.logs) > maxLogLines {
		m.logs = m.logs[len(m.logs)-maxLogLines:]
	}
	m.mu.Unlock()
	if !m.eventsOn {
		return
	}
	runtime.EventsEmit(m.ctx, "alist:log", line)
}

// Start 启动 alist。首次初始化时注入固定初始密码 admin（桌面端不保存任何密码）。
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.cmd != nil && m.cmd.Process != nil {
		m.mu.Unlock()
		return fmt.Errorf("alist 已在运行")
	}
	needInit := !configExists(m.dataDir)
	var inject string
	if needInit {
		inject = defaultPassword
	}
	m.mu.Unlock()

	m.setState(StatusStarting)
	if err := m.spawn(inject); err != nil {
		m.setState(StatusError)
		return err
	}
	go m.waitForHealthy()
	return nil
}

func (m *Manager) spawn(injectPwd string) error {
	cmd := exec.Command(m.exePath, "server", "--data", m.dataDir, "--log-std")
	// 独立进程组（便于发送 CTRL_BREAK_EVENT）；HideWindow 防止 GUI 父进程
	// 拉起控制台子进程时弹出黑色 CMD 窗口（用户误触/关闭会杀掉服务）。
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	if injectPwd != "" {
		cmd.Env = append(os.Environ(), "ALIST_ADMIN_PASSWORD="+injectPwd)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start alist: %w", err)
	}
	m.mu.Lock()
	m.cmd = cmd
	m.mu.Unlock()
	go m.pump(stdout, "stdout")
	go m.pump(stderr, "stderr")
	return nil
}

func (m *Manager) pump(r io.Reader, src string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		lvl, msg := parseLevel(line)
		if strings.TrimSpace(msg) == "" {
			continue
		}
		m.emitLog(LogLine{Time: time.Now().Format("15:04:05"), Level: lvl, Msg: msg})
	}
}

// parseLevel 从 alist 日志行粗略提取级别（INFO/WARN/ERROR/DEBUG），否则归为 INFO。
func parseLevel(line string) (string, string) {
	up := strings.ToUpper(line)
	switch {
	case strings.Contains(up, "ERROR"), strings.Contains(up, "FATAL"):
		return "error", line
	case strings.Contains(up, "WARN"):
		return "warn", line
	case strings.Contains(up, "DEBUG"):
		return "debug", line
	default:
		return "info", line
	}
}

// Stop 停止 alist：优先向进程组发 CTRL_BREAK_EVENT 触发 alist 优雅 Shutdown，
// 超时（或本进程无控制台无法投递事件）则 TerminateProcess 强杀。
func (m *Manager) Stop() error {
	m.mu.Lock()
	cmd := m.cmd
	m.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	m.setState(StatusStopping)

	grace := stopGracePeriod
	if err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(cmd.Process.Pid)); err != nil {
		// GUI 进程通常无控制台，事件无法投递 → 缩短等待后强杀。
		grace = noConsoleGrace
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(grace):
		_ = cmd.Process.Kill()
		<-done
	}
	m.mu.Lock()
	m.cmd = nil
	m.mu.Unlock()
	m.setState(StatusStopped)
	return nil
}

// Restart 先停后起。
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	return m.Start()
}

// IsRunning 子进程是否存活。
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cmd != nil && m.cmd.Process != nil
}

// HealthCheck 通过 HTTP 探活判断 alist 是否就绪。
func (m *Manager) HealthCheck() bool {
	port := m.httpPort()
	if port <= 0 {
		return false
	}
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

func (m *Manager) waitForHealthy() {
	deadline := time.Now().Add(healthTimeout)
	for time.Now().Before(deadline) {
		if m.HealthCheck() {
			// 刷新端口/地址缓存
			if cfg, err := ReadConfig(m.dataDir); err == nil {
				m.mu.Lock()
				m.cfg = cfg
				m.mu.Unlock()
			}
			m.setState(StatusRunning)
			return
		}
		time.Sleep(400 * time.Millisecond)
	}
	m.setState(StatusError)
}

func (m *Manager) httpPort() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cfg != nil && m.cfg.Scheme.HTTPPort > 0 {
		return m.cfg.Scheme.HTTPPort
	}
	return defaultHTTPPort
}

func (m *Manager) httpAddress() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cfg != nil && m.cfg.Scheme.Address != "" {
		return m.cfg.Scheme.Address
	}
	return "0.0.0.0"
}

// ResetAdmin 将管理员账号密码重置为 admin / admin：必须停服后再执行 admin 子命令
// （避免 SQLite 并发写），完成后按需重启。alist 内核 `admin set` 按角色定位管理员，
// 用户名即使被网页端修改过也能重置密码；若用户名被改过，需在网页端自行改回 admin。
func (m *Manager) ResetAdmin() error {
	wasRunning := m.IsRunning()
	if wasRunning {
		if err := m.Stop(); err != nil {
			return err
		}
	}
	adminCmd := exec.Command(m.exePath, "admin", "set", defaultPassword, "--data", m.dataDir)
	// admin 是控制台子命令，同样隐藏其 CMD 窗口。
	adminCmd.SysProcAttr = &windows.SysProcAttr{HideWindow: true}
	if out, err := adminCmd.CombinedOutput(); err != nil {
		// 重置失败，尽量恢复原运行状态
		if wasRunning {
			_ = m.Start()
		}
		return fmt.Errorf("admin set 失败: %v: %s", err, out)
	}
	if wasRunning {
		return m.Start()
	}
	m.emitState()
	return nil
}

// GetState 返回前端快照。
func (m *Manager) GetState() State {
	m.mu.Lock()
	status := m.status
	cfg := m.cfg
	autoSvc := m.settings.AutoStartService
	m.mu.Unlock()

	addr := "0.0.0.0"
	httpPort := defaultHTTPPort
	httpsPort := -1
	if cfg != nil {
		if cfg.Scheme.Address != "" {
			addr = cfg.Scheme.Address
		}
		if cfg.Scheme.HTTPPort > 0 {
			httpPort = cfg.Scheme.HTTPPort
		}
		httpsPort = cfg.Scheme.HTTPSPort
	}
	clickAddr := addr
	if clickAddr == "0.0.0.0" || clickAddr == "" {
		clickAddr = "127.0.0.1"
	}
	url := fmt.Sprintf("http://%s:%d/", clickAddr, httpPort)
	return State{
		Status:     string(status),
		IsPortable: m.isPortable,
		AutoStart:  IsAutoStart(),
		AutoStartService: autoSvc,
		Address:    addr,
		HTTPPort:   httpPort,
		HTTPSPort:  httpsPort,
		URL:        url,
		Account:    alistAccount,
		DataDir:    m.dataDir,
		ExePath:    m.exePath,
	}
}

// GetLogs 返回日志环形缓冲快照。
func (m *Manager) GetLogs() []LogLine {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]LogLine, len(m.logs))
	copy(out, m.logs)
	return out
}

func configExists(dataDir string) bool {
	_, err := os.Stat(filepath.Join(dataDir, "config.json"))
	return err == nil
}
