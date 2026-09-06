# app（AList 桌面版 后端 + 前端）

本目录是 `alist-windows-gui` 的主程序源码，由 **Wails v2** 驱动：

- `main.go` / `app.go` — 应用入口与 Wails 生命周期（启动 alist、托盘、窗口）。
- `internal/` — 业务逻辑：`alist`（内核管理）、`tray`（系统托盘）、`trayicon`（图标资源）。
- `frontend/` — Vue 3 + TypeScript + Vite 前端（`wailsjs/` 为 Wails 生成的 Go↔JS 绑定）。

## 开发

```bash
cd app
wails dev          # 热重载开发模式（需 Node 22）
wails build -platform windows/amd64   # 生产构建，产物 app/build/bin/app.exe
```

## 说明

- 完整项目说明、构建、许可证与自动发布流程见仓库根目录 [README.md](../README.md)。
- 整仓库以 **AGPL-3.0** 发布；alist 内核同为 AGPL-3.0。
