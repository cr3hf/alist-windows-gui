# AList 桌面版 (alist-windows-gui)

一个为 [alist](https://github.com/AlistGo/alist) 文件列表程序提供的 **Windows 桌面外壳**：内置 alist 内核，无需命令行即可在本机快速部署、启动、管理与使用 alist，并常驻系统托盘。

> 本项目仅是一个“外壳 / 启动器”，通过命令行与 HTTP 接口驱动独立的 alist 进程，**不修改 alist 源码**。

---

## 功能特性

- **一键启动 / 停止 / 重启** alist 内核，状态实时显示（运行中 / 启动中 / 已停止）。
- **系统托盘常驻**：左键唤起主界面，右键弹出菜单（启动、停止、重启、打开网页、日志、数据目录、关于、退出）。
- **开机自启** + **自动启动服务**（打开软件即拉起 alist，可独立开关）。
- **实时日志面板**：缓冲刷新（1 秒批量）、上限自动清理、级别筛选、关键字搜索、导出 / 清空。
- **浅色 / 深色双主题**：完整风格化配色（文字、按钮、徽章、日志、弹窗、滚动条等各自独立变量），非单纯换背景。
- **重置密码**：一键将账号密码重置为 `admin / admin`。
- **关于页**：产品版本、alist 内核版本、许可证与上游地址。

---

## 下载与安装

在 [Releases](../../releases) 页面获取两种分发形式：

| 分发形式 | 文件 | 说明 |
| --- | --- | --- |
| 安装包 | `alist_win-setup-<ver>.exe` | 安装到 `%LOCALAPPDATA%\Programs\alist_win`，创建开始菜单与桌面快捷方式，per-user 免管理员权限 |
| 便携包 | `alist_win-portable-<ver>.zip` | 解压即用，目录内含 `portable.ini` 即为便携模式，数据存于同目录 `data` |

> **SmartScreen 提示**：本程序未做代码签名。首次运行若出现 “Windows 已保护你的电脑”，点击 **“更多信息” → “仍要运行”** 即可。

---

## 默认账号

- 默认账号 / 密码均为 **`admin` / `admin`**。
- 本程序 **不保存任何密码**；若在 alist 网页端修改过密码，可在主界面点击 **“重置密码”** 一键还原为 `admin / admin`（服务运行中会短暂重启）。
- 若在网页端改过用户名，密码仍会重置，但用户名需回到网页端改回 `admin`。

---

## 从源码构建

### 环境要求

| 工具 | 版本 | 说明 |
| --- | --- | --- |
| Go | 1.25.0+ | 后端与 Wails CLI |
| Node.js | 22.x | 前端（Vue 3 + TypeScript + Vite） |
| Wails CLI | v2.14.0 | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0` |
| NSIS | 3.x | 仅生成安装包时需要 |
| alist 内核 | v3.64.0 | 构建脚本会自动下载 `alist-windows-amd64.zip` 到 `bin/`（`bin/` 已被 git 忽略，不入库） |

### 本地构建

```bash
# 1) 准备 alist 内核（也可手动放入 bin/alist.exe）
curl -L -o bin/alist.zip https://github.com/AlistGo/alist/releases/download/v3.64.0/alist-windows-amd64.zip
powershell -Command "Expand-Archive -Path 'bin/alist.zip' -DestinationPath 'bin' -Force"

# 2) 构建（等价于 wails build -platform windows/amd64）
cd app
wails build -platform windows/amd64
# 产物：app/build/bin/app.exe

# 3) 打包安装包 + 便携包（需要 NSIS）
bash scripts/package.sh        # 本地脚本，依赖本机 tools/ 中的 NSIS 与 _env.sh
```

> 本地 `scripts/package.sh` 依赖开发者本机的 NSIS 与 `_env.sh`；CI 使用 `.github/scripts/ci-build.sh`，自动下载 alist 并安装 NSIS，二者产物一致。

---

## 自动发布（GitHub Actions）

- 推送到 `main` / 提交 PR：由 `ci.yml` 执行 `go build` / `go vet` / 前端 `npm run build` 做编译校验。
- 推送 **`vX.Y.Z` 标签**：由 `release.yml` 在 `windows-latest` 上自动
  1. 安装 Go / Node / Wails CLI / NSIS；
  2. 下载 alist v3.64.0 内核；
  3. `wails build` → NSIS 安装包 + 便携 zip；
  4. 创建 GitHub Release 并上传两个产物。

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 release.yml → 自动生成 Release 与安装包/便携包
```

---

## 自动化技能（Agent Skill）

[`docs/skills/github-open-source-release/SKILL.md`](./docs/skills/github-open-source-release/SKILL.md)
沉淀了本项目开源与 CI 落地的完整流程，可直接复用到其他项目：

- 无 `gh` CLI 时，如何复用本机缓存凭据调用 GitHub API 建仓 / 推送 / 发布；
- `.gitignore` / `.gitattributes` 的常见陷阱（行尾空格导致规则失效、CRLF 破坏 `.sh` 脚本）；
- Windows CI 的三个高频坑（`TMP` 变量被 Go 工具链占用、`go:embed` 需先构建前端、choco NSIS 不生成 shim）；
- 拉取失败日志、以及重建标签重新触发发布的方法。

> 该文档仅含通用流程与占位符（`<repo>` / `<user>`），不包含任何凭据或个人路径信息。

---

## 许可证

- 本桌面外壳（alist-windows-gui）以 **GNU Affero 通用公共许可证 v3.0（AGPL-3.0）** 发布。完整文本见仓库根目录 [`LICENSE`](./LICENSE)。
- 第三方组件 **alist** 为 **AGPL-3.0**（Copyright AlistGo 及其贡献者），许可证全文随包分发于 `bin/LICENSE-alist.txt`，源码见 <https://github.com/AlistGo/alist>。
- 其他依赖：Wails v2（MIT）、energye/systray（Apache-2.0）、golang.org/x/sys（BSD-3-Clause）等，各自许可证文本见其源码仓库。

依据 AGPL-3.0 第 13 条，本仓库即为对应源码获取方式；本程序未修改 alist 源码，如未来修改后再分发，将同样以 AGPL-3.0 提供。

---

## 免责声明

本程序按“现状”提供，不附带任何明示或默示的担保。使用本程序存储、分享数据所产生的行为与后果由使用者自行承担。
