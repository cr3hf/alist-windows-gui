---
name: github-open-source-release
description: 把本地项目开源到 GitHub：创建仓库、准备 .gitignore/.gitattributes/LICENSE/README、编写 GitHub Actions 自动构建并发布 Release。适用于"开源这个源码到 GitHub""自动出包发布 release""CI 构建失败排查"。
agent_created: true
---

# 开源项目到 GitHub 并配置自动 Release

## 1. 认证发现（无需 gh CLI）

`gh` 常未安装。优先复用本机缓存的 git 凭据：

```bash
creds=$(printf "protocol=https\nhost=github.com\n" | git credential fill 2>/dev/null)
GH_USER=$(echo "$creds" | sed -n 's/^username=//p')
GH_TOKEN=$(echo "$creds" | sed -n 's/^password=//p')
```

- 用 `curl -s -I -u "$GH_USER:$GH_TOKEN" https://api.github.com/user | grep -i x-oauth-scopes` 确认权限。
- 需要 `repo`（建仓/推送）与 `workflow`（写 Actions）。`gho_` 前缀 = OAuth 令牌。
- **绝不要把 token 打印到输出里**，只在变量中传递。

## 2. 建仓

```bash
curl -u "$GH_USER:$GH_TOKEN" -X POST https://api.github.com/user/repos \
  -H "Accept: application/vnd.github+json" \
  -d '{"name":"<repo>","description":"...","private":false,"has_issues":true,"has_wiki":false}'
```

`auto_init` 保持 false（避免与本地 README/LICENSE 冲突）。

## 3. 仓库准备（两个必踩的坑）

- **`.gitignore` 行尾不能有空格**：`/dist/   # 注释` 会被当成模式 `/dist/   `，导致完全不生效。
  必须把注释放到独立行。用 `cat -A .gitignore` 检查行尾是否为 `$` 而非空格 `$`。
- **加 `.gitattributes`**：Windows 上 `core.autocrlf=true` 会把 `.sh` 转成 CRLF 使脚本失效。
  ```
  * text=auto eol=lf
  *.png binary
  *.ico binary
  *.woff2 binary
  ```
- 大体积第三方二进制（如 exe）**不入库**，改为构建时下载，并在 `.gitignore` 中排除。
- 加 `LICENSE`（与项目许可证一致的全文）+ 根 `README.md`。

## 4. 提交与触发

```bash
git init && git rm -r --cached -f . && git add -A   # 改完 .gitignore 后务必强制重置索引
git commit -m "..."
git remote add origin https://github.com/<user>/<repo>.git
git push -u origin main
git tag -a v1.0.0 -m "Release v1.0.0" && git push origin v1.0.0
```

**重新触发同版本 Release**（构建失败修复后）：删除远端标签再重建同名标签推送：
```bash
git push origin --delete refs/tags/v1.0.0
git tag -d v1.0.0
git tag -a v1.0.0 -m "Release v1.0.0" && git push origin v1.0.0
```

## 5. 排查失败的 Action

```bash
JID=$(curl -s -u "$GH_USER:$GH_TOKEN" "$API/actions/runs/<run_id>/jobs" \
      | python -c "import sys,json;print(json.load(sys.stdin)['jobs'][0]['id'])")
curl -sL -u "$GH_USER:$GH_TOKEN" "$API/actions/jobs/$JID/logs" -o log.txt
grep -nE "##\[error\]|Error:|No such file|not found" log.txt | tail -20
```

注意：沙箱里 `/tmp` 常常不可写，**日志要存到工作区目录**。

## 6. Windows CI 三个高频坑

1. **脚本里绝不要把变量命名为 `TMP`/`TEMP`** —— 它们是 Windows 已导出的环境变量、Go 工具链用作临时工作目录。
   若脚本 `rm -rf` 掉它，后续 `go` 调用会报
   `go: creating work dir: ... The system cannot find the file specified`。用 `WORKTMP` 之类名字。
2. **`//go:embed frontend/dist` 的项目，必须先构建前端再 `go build`**，否则
   `pattern all:frontend/dist: no matching files found`。CI 里顺序：npm ci → npm run build → go build/vet。
   （`wails build` 会自行构建前端，不受此限。）
3. **choco 的 `nsis` 包装到 `C:\Program Files (x86)\NSIS`，且不会在 `C:\ProgramData\chocolatey\bin` 生成 shim**，
   硬编码该路径会 exit 127。改为探测：`$NSIS` 环境变量 → `PATH` →
   `C:/Program Files (x86)/NSIS[/Bin]/makensis.exe`、`C:/Program Files/NSIS[/Bin]/makensis.exe`。

## 7. 发布产物

在 workflow 中用 `softprops/action-gh-release@v2` + `GITHUB_TOKEN`，需 `permissions: contents: write`：

```yaml
on:
  push:
    tags: ['v*']
permissions:
  contents: write
```

构建脚本把产物写到仓库下 `dist/` 再用 `files: dist/*.exe` 上传（路径相对 `GITHUB_WORKSPACE`）。
Git Bash 调用 Windows 原生程序（makensis/powershell）时，路径要用 `cygpath -w` 转换。
