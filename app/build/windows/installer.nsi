; AList 桌面版 — NSIS 安装脚本（M6）
; 用法：makensis /DOUTFILE=... /DSRCDIR=... /DICON=... installer.nsi
; 布局符合方案 7.1：安装到 %LOCALAPPDATA%\Programs\alist_win，per-user 免管理员权限。

Unicode true

!define PRODUCT_NAME "AList 桌面版"
!define PRODUCT_VER "1.0.0"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\alist_win"
!define RUN_PATH "Software\Microsoft\Windows\CurrentVersion\Run"

Name "${PRODUCT_NAME} ${PRODUCT_VER}"
OutFile "${OUTFILE}"
InstallDir "$LOCALAPPDATA\Programs\alist_win"
RequestExecutionLevel user
SetCompressor /SOLID lzma
Icon "${ICON}"
UninstallIcon "${ICON}"

Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

Section "Install"
  ; 覆盖安装前结束可能运行的实例（V1 简化策略）
  nsExec::Exec 'taskkill /IM alist.exe /F'
  nsExec::Exec 'taskkill /IM alist_win.exe /F'
  Sleep 500

  SetOutPath $INSTDIR
  File "${SRCDIR}\alist_win.exe"
  File "${SRCDIR}\LICENSE.txt"
  File "${SRCDIR}\README.txt"

  SetOutPath "$INSTDIR\bin"
  File "${SRCDIR}\bin\alist.exe"
  File "${SRCDIR}\bin\LICENSE-alist.txt"

  WriteUninstaller "$INSTDIR\uninstall.exe"

  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\alist_win.exe"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\卸载 ${PRODUCT_NAME}.lnk" "$INSTDIR\uninstall.exe"
  CreateShortCut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\alist_win.exe"

  WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${PRODUCT_VER}"
  WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "alist_win"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\alist_win.exe"
  WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "${UNINST_KEY}" "QuietUninstallString" "$INSTDIR\uninstall.exe /S"
SectionEnd

Section "Uninstall"
  nsExec::Exec 'taskkill /IM alist.exe /F'
  nsExec::Exec 'taskkill /IM alist_win.exe /F'
  Sleep 500

  ; 清理开机自启（方案 5.4：卸载清理）
  DeleteRegValue HKCU "${RUN_PATH}" "alist_win"

  Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\卸载 ${PRODUCT_NAME}.lnk"
  RMDir "$SMPROGRAMS\${PRODUCT_NAME}"
  Delete "$DESKTOP\${PRODUCT_NAME}.lnk"
  DeleteRegKey HKCU "${UNINST_KEY}"

  MessageBox MB_YESNO|MB_ICONQUESTION "是否同时删除用户数据（alist 数据目录）？$\n$LOCALAPPDATA\alist_win" IDNO skip_data
    RMDir /r "$LOCALAPPDATA\alist_win"
  skip_data:

  Delete "$INSTDIR\uninstall.exe"
  RMDir /r "$INSTDIR"
SectionEnd
