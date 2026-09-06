package alist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AlistConfig 只解析我们关心的字段（与 alist data/config.json 的 scheme 对齐）。
// 其余字段 alist 自己管理，桌面端不解析、不修改。
type AlistConfig struct {
	Scheme struct {
		Address   string `json:"address"`
		HTTPPort  int    `json:"http_port"`
		HTTPSPort int    `json:"https_port"`
	} `json:"scheme"`
}

// ReadConfig 读取 <dataDir>/config.json，返回监听地址与端口。
// alist 未初始化（首次运行前）时文件不存在，返回零值 + nil（调用方按"未运行/未配置"处理）。
func ReadConfig(dataDir string) (*AlistConfig, error) {
	p := filepath.Join(dataDir, "config.json")
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &AlistConfig{}, nil
		}
		return nil, fmt.Errorf("read config.json: %w", err)
	}
	var c AlistConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config.json: %w", err)
	}
	return &c, nil
}
