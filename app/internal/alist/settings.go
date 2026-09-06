package alist

import (
	"encoding/json"
	"os"
)

// Settings 桌面端自身设置（desktop-settings.json 与数据目录同级存储：
// 便携模式在 exe 旁，安装模式在 %LOCALAPPDATA%/alist_win/）。
type Settings struct {
	AutoStartService bool `json:"autoStartService"` // 打开软件后自动启动 alist 服务
}

func loadSettings(path string) Settings {
	s := Settings{AutoStartService: true} // 默认开启
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

func saveSettings(path string, s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// GetSettings 返回设置快照。
func (m *Manager) GetSettings() Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings
}

// SetAutoStartService 设置"打开软件后自动启动 alist 服务"并持久化。
func (m *Manager) SetAutoStartService(on bool) error {
	m.mu.Lock()
	m.settings.AutoStartService = on
	path := m.settingsPath
	m.mu.Unlock()
	return saveSettings(path, m.settings)
}
