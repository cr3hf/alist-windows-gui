// Package trayicon 内嵌托盘图标资源。
package trayicon

import _ "embed"

//go:embed icon.ico
var iconICO []byte

// Icon 返回托盘 .ico 原始字节。
func Icon() []byte { return iconICO }
