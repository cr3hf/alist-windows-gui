#!/usr/bin/env bash
# CI 构建脚本：在 GitHub windows-latest 上生成安装包与便携包。
# 前置条件（由 release.yml 安装）：Go 1.25、Node 22、wails CLI(on PATH)、NSIS(on PATH)。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$SCRIPT_DIR/../.." && pwd)"

APP="$REPO/app"
DIST="$REPO/dist"
ASSETS="$REPO/dist-assets"
RES="$REPO/resources"
BIN="$REPO/bin"

VER="${VERSION:-1.0.0}"
VER="${VER#v}"            # 去掉标签前缀 v

ALIST_VERSION="v3.64.0"   # 随分发锁定的 alist 内核版本

echo "==> Repo : $REPO"
echo "==> Ver  : $VER"
echo "==> alist: $ALIST_VERSION"

# 1) 下载 alist 内核（AGPL-3.0）到 bin/（不入库，CI 每次拉取）
if [ ! -f "$BIN/alist.exe" ]; then
  echo "==> 下载 alist $ALIST_VERSION ..."
  mkdir -p "$BIN" "$DIST"
  # 注意：绝不可命名为 TMP / TEMP —— 那是 Windows 已导出的环境变量，
  # Go 工具链会将其作为临时工作目录；若在此删除会导致 wails build 报
  # "go: creating work dir: ... The system cannot find the file specified"。
  WORKTMP="$REPO/.ci-tmp"
  rm -rf "$WORKTMP"; mkdir -p "$WORKTMP"
  curl -sSL -o "$WORKTMP/alist.zip" \
    "https://github.com/AlistGo/alist/releases/download/$ALIST_VERSION/alist-windows-amd64.zip"
  WORKTMP_WIN="$(cygpath -w "$WORKTMP")"
  powershell.exe -NoProfile -Command "Expand-Archive -Path '$WORKTMP_WIN/alist.zip' -DestinationPath '$WORKTMP_WIN' -Force"
  cp "$WORKTMP/alist.exe" "$BIN/alist.exe"
  rm -rf "$WORKTMP"
fi

# 2) Wails 构建（Windows/amd64，纯 Go，无 CGO）
echo "==> Wails build (windows/amd64)"
command -v wails >/dev/null 2>&1 || { echo "wails 不在 PATH 上"; exit 1; }
( cd "$APP" && wails build -platform windows/amd64 )
ls -la "$APP/build/bin/"

# 3) 暂存安装目录
echo "==> 暂存 (stage)"
TS="$(date +%Y%m%d-%H%M%S)"
TRASH="$DIST/.trash-$TS"
mkdir -p "$TRASH"
[ -d "$DIST/stage-install" ] && mv "$DIST/stage-install" "$TRASH/" || true
[ -d "$DIST/stage-portable" ] && mv "$DIST/stage-portable" "$TRASH/" || true
mkdir -p "$DIST/stage-install/bin" "$DIST/stage-portable/bin"
cp "$APP/build/bin/app.exe"        "$DIST/stage-install/alist_win.exe"
cp "$BIN/alist.exe"                 "$DIST/stage-install/bin/"
cp "$RES/LICENSE-ALIST.txt"         "$DIST/stage-install/bin/LICENSE-alist.txt"
cp "$ASSETS/LICENSE.txt" "$ASSETS/README.txt" "$DIST/stage-install/"
cp -r "$DIST/stage-install/."       "$DIST/stage-portable/"
touch "$DIST/stage-portable/portable.ini"

# 4) NSIS 安装包（确保 installer.nsi 恰好一个 UTF-8 BOM）
echo "==> NSIS 安装包"
# 解析 makensis 路径：优先 $NSIS 环境变量 → PATH → 常见安装位置。
# 注意：choco 的 nsis 包部署到 "C:\Program Files (x86)\NSIS" 且不会生成
# chocolatey\bin 下的 shim，因此不能假设 C:\ProgramData\chocolatey\bin\makensis.exe 存在。
resolve_makensis() {
  if [ -n "${NSIS:-}" ] && [ -f "$NSIS" ]; then echo "$NSIS"; return 0; fi
  local found
  found="$(command -v makensis 2>/dev/null || true)"
  if [ -n "$found" ]; then echo "$found"; return 0; fi
  for cand in \
    "/c/ProgramData/chocolatey/bin/makensis.exe" \
    "/c/Program Files (x86)/NSIS/makensis.exe" \
    "/c/Program Files (x86)/NSIS/Bin/makensis.exe" \
    "/c/Program Files/NSIS/makensis.exe" \
    "/c/Program Files/NSIS/Bin/makensis.exe" ; do
    if [ -f "$cand" ]; then echo "$cand"; return 0; fi
  done
  echo ""
}
NSIS="$(resolve_makensis)"
if [ -z "$NSIS" ]; then
  echo "!! 未找到 makensis.exe（NSIS）。请安装 NSIS，或用 NSIS 环境变量指向 makensis.exe"
  exit 127
fi
echo "makensis => $NSIS"
NSIF="$APP/build/windows/installer.nsi"
bom3=$(head -c 3 "$NSIF" | od -An -tx1 | tr -d ' \n')
if [ "$bom3" = "efbbbf" ] && [ "$(head -c 6 "$NSIF" | od -An -tx1 | tr -d ' \n')" = "efbbbfefbbbf" ]; then
  tail -c +4 "$NSIF" > "$NSIF.tmp" && mv "$NSIF.tmp" "$NSIF"
fi
if [ "$(head -c 3 "$NSIF" | od -An -tx1 | tr -d ' \n')" != "efbbbf" ]; then
  sed -i '1s/^/\xEF\xBB\xBF/' "$NSIF"
fi
mkdir -p "$DIST"
NSIS_WIN="$(cygpath -w "$NSIS")"
OUT_WIN="$(cygpath -w "$DIST/alist_win-setup-$VER.exe")"
SRC_WIN="$(cygpath -w "$DIST/stage-install")"
ICON_WIN="$(cygpath -w "$APP/internal/trayicon/icon.ico")"
"$NSIS_WIN" "/DOUTFILE=$OUT_WIN" "/DSRCDIR=$SRC_WIN" "/DICON=$ICON_WIN" "$(cygpath -w "$NSIF")"

# 5) 便携 zip
echo "==> 便携 zip"
cd "$DIST"
rm -rf "alist_win-portable-$VER"
cp -r stage-portable "alist_win-portable-$VER"
PORTABLE_DIR_WIN="$(cygpath -w "$DIST/alist_win-portable-$VER")"
ZIP_WIN="$(cygpath -w "$DIST/alist_win-portable-$VER.zip")"
powershell.exe -NoProfile -Command "Compress-Archive -Path '$PORTABLE_DIR_WIN' -DestinationPath '$ZIP_WIN' -Force"
rm -rf "alist_win-portable-$VER"

echo "==> 产物:"
ls -la "$DIST/alist_win-setup-$VER.exe" "$DIST/alist_win-portable-$VER.zip"
