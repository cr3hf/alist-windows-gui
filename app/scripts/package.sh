#!/usr/bin/env bash
# alist_win 打包脚本（M6/M7）：wails build → 阶段目录 → NSIS 安装包 + 便携 zip
# 用法：bash scripts/package.sh
set -euo pipefail

ROOT="C:/Users/One/WorkBuddy/2026-09-05-23-12-17"
APP="$ROOT/alist_win/app"
DIST="$ROOT/alist_win/dist"
ASSETS="$ROOT/alist_win/dist-assets"
NSIS="$ROOT/tools/nsis-3.10/Bin/makensis.exe"
VER="1.0.0"

source "$ROOT/_env.sh"
cd "$APP"

echo "=== [1/5] wails build ==="
"$WAILS" build -platform windows/amd64
ls -la build/bin/

echo "=== [2/5] stage installer ==="
# 沙箱对 rm -rf 有安全删除钩子（失败即中止），改用 mv 隔离旧产物
TS=$(date +%Y%m%d-%H%M%S)
TRASH="$DIST/.trash-$TS"
mkdir -p "$TRASH"
{ [ -d "$DIST/stage-install" ] && mv "$DIST/stage-install" "$TRASH/" ; } || true
{ [ -d "$DIST/stage-portable" ] && mv "$DIST/stage-portable" "$TRASH/" ; } || true
mkdir -p "$DIST/stage-install/bin" "$DIST/stage-portable/bin"
cp build/bin/app.exe "$DIST/stage-install/alist_win.exe"
cp "$ROOT/alist_win/bin/alist.exe" "$DIST/stage-install/bin/"
cp "$ROOT/alist_win/resources/LICENSE-ALIST.txt" "$DIST/stage-install/bin/LICENSE-alist.txt"
cp "$ASSETS/LICENSE.txt" "$ASSETS/README.txt" "$DIST/stage-install/"

echo "=== [3/5] stage portable ==="
cp -r "$DIST/stage-install/." "$DIST/stage-portable/"
touch "$DIST/stage-portable/portable.ini"

echo "=== [4/5] NSIS installer ==="
# nsis 脚本含中文，需 UTF-8 BOM；双 BOM 会导致 makensis 报 Invalid command
# （先剥掉开头所有 BOM，再确保恰好一个）
NSIF=build/windows/installer.nsi
bom3=$(head -c 3 "$NSIF" | od -An -tx1 | tr -d ' \n')
if [ "$bom3" = "efbbbf" ] && [ "$(head -c 6 "$NSIF" | od -An -tx1 | tr -d ' \n')" = "efbbbfefbbbf" ]; then
  tail -c +4 "$NSIF" > "$NSIF.tmp" && mv "$NSIF.tmp" "$NSIF"
fi
if [ "$(head -c 3 "$NSIF" | od -An -tx1 | tr -d ' \n')" != "efbbbf" ]; then
  sed -i '1s/^/\xEF\xBB\xBF/' "$NSIF"
fi
mkdir -p "$DIST"
"$NSIS" "/DOUTFILE=$DIST/alist_win-setup-$VER.exe" \
        "/DSRCDIR=$DIST/stage-install" \
        "/DICON=$APP/internal/trayicon/icon.ico" \
        build/windows/installer.nsi | tail -5

echo "=== [5/5] portable zip ==="
cd "$DIST"
{ [ -e "alist_win-portable-$VER" ] && mv "alist_win-portable-$VER" "$TRASH/" ; } || true
cp -r stage-portable "alist_win-portable-$VER"
if command -v powershell.exe >/dev/null 2>&1; then
  powershell.exe -NoProfile -Command "Compress-Archive -Path 'alist_win-portable-$VER' -DestinationPath 'alist_win-portable-$VER.zip' -Force"
else
  zip -qr "alist_win-portable-$VER.zip" "alist_win-portable-$VER"
fi
{ [ -d "alist_win-portable-$VER" ] && mv "alist_win-portable-$VER" "$TRASH/" ; } || true

echo "=== dist ==="
ls -la "$DIST"
