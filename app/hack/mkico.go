//go:build ignore

// 将 build/appicon.png 封装为 Windows .ico（PNG-in-ICO），供 systray 与 NSIS 使用。
// 用法：go run hack/mkico.go
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	root, _ := os.Getwd()
	pngPath := filepath.Join(root, "build", "appicon.png")
	png, err := os.ReadFile(pngPath)
	if err != nil {
		fmt.Println("read png:", err)
		os.Exit(1)
	}
	out := make([]byte, 0, len(png)+22)
	// ICONDIR
	hdr := []byte{0, 0, 1, 0, 1, 0}
	out = append(out, hdr...)
	// ICONDIRENTRY (16 bytes)
	entry := make([]byte, 16)
	entry[0] = 0 // width 256 -> 0
	entry[1] = 0 // height 256 -> 0
	entry[2] = 0 // colors
	entry[3] = 0 // reserved
	binary.LittleEndian.PutUint16(entry[4:], 1)    // planes
	binary.LittleEndian.PutUint16(entry[6:], 32)   // bit count
	binary.LittleEndian.PutUint32(entry[8:], uint32(len(png)))
	binary.LittleEndian.PutUint32(entry[12:], 22) // offset to image
	out = append(out, entry...)
	out = append(out, png...)

	dest := filepath.Join(root, "internal", "trayicon", "icon.ico")
	_ = os.MkdirAll(filepath.Dir(dest), 0o755)
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		fmt.Println("write ico:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", dest, len(out))
}
