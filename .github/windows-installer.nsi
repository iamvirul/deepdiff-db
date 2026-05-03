; DeepDiff DB — Windows Installer
; Built by the release workflow; __APP_VERSION__ is substituted at compile time.
;
; Installs deepdiffdb.exe to %ProgramFiles%\DeepDiffDB, adds that directory
; to the system PATH, and registers the app in Add/Remove Programs.

Unicode true
SetCompressor /SOLID lzma

!define APP_NAME    "DeepDiff DB"
!define APP_VERSION "__APP_VERSION__"
!define PUBLISHER   "iamvirul"
!define HOMEPAGE    "https://iamvirul.github.io/deepdiff-db/"
!define INSTALL_DIR "$PROGRAMFILES64\DeepDiffDB"
!define UNINSTKEY   "Software\Microsoft\Windows\CurrentVersion\Uninstall\DeepDiffDB"
!define REGKEY      "Software\DeepDiffDB"

Name            "${APP_NAME} ${APP_VERSION}"
OutFile         "deepdiff-db-v__APP_VERSION__-windows-amd64-installer.exe"
InstallDir      "${INSTALL_DIR}"
InstallDirRegKey HKLM "${REGKEY}" "InstallDir"
RequestExecutionLevel admin

; ── Pages ──────────────────────────────────────────────────────────────────
Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

; ── Install ────────────────────────────────────────────────────────────────
Section "DeepDiff DB" SecMain
  SectionIn RO
  SetOutPath "$INSTDIR"

  File "deepdiffdb.exe"
  File "README.md"
  File "deepdiffdb.config.yaml.example"

  ; Add install dir to system PATH (append only when not already present)
  ReadRegStr $0 HKLM \
    "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
  StrCpy $1 $0 "" -1          ; last char of current PATH
  StrCmp $1 ";" 0 +2
    StrCpy $0 $0 -1            ; strip trailing semicolon
  ; Check whether INSTDIR already appears (simple prefix/suffix check is fine
  ; because users rarely change the install location)
  Push "$0"
  Push "$INSTDIR"
  Call StrContains
  Pop $1
  StrCmp $1 "" 0 path_skip
    WriteRegExpandStr HKLM \
      "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
      "Path" "$0;$INSTDIR"
    SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 \
      "STR:Environment" /TIMEOUT=5000
  path_skip:

  ; Write uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; Add/Remove Programs entry
  WriteRegStr   HKLM "${UNINSTKEY}" "DisplayName"     "${APP_NAME} ${APP_VERSION}"
  WriteRegStr   HKLM "${UNINSTKEY}" "DisplayVersion"  "${APP_VERSION}"
  WriteRegStr   HKLM "${UNINSTKEY}" "Publisher"       "${PUBLISHER}"
  WriteRegStr   HKLM "${UNINSTKEY}" "URLInfoAbout"    "${HOMEPAGE}"
  WriteRegStr   HKLM "${UNINSTKEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr   HKLM "${UNINSTKEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegDWORD HKLM "${UNINSTKEY}" "NoModify"        1
  WriteRegDWORD HKLM "${UNINSTKEY}" "NoRepair"        1
  WriteRegStr   HKLM "${REGKEY}"    "InstallDir"      "$INSTDIR"
SectionEnd

; ── Uninstall ──────────────────────────────────────────────────────────────
Section "Uninstall"
  Delete "$INSTDIR\deepdiffdb.exe"
  Delete "$INSTDIR\README.md"
  Delete "$INSTDIR\deepdiffdb.config.yaml.example"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir  "$INSTDIR"

  ; Remove install dir from system PATH
  ReadRegStr $0 HKLM \
    "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
  ; Strip "$INSTDIR;" and ";$INSTDIR" variants
  Push "$0"
  Push "$INSTDIR;"
  Push ""
  Call StrReplaceAll
  Pop $0
  Push "$0"
  Push ";$INSTDIR"
  Push ""
  Call StrReplaceAll
  Pop $0
  WriteRegExpandStr HKLM \
    "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
    "Path" "$0"
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 \
    "STR:Environment" /TIMEOUT=5000

  DeleteRegKey HKLM "${UNINSTKEY}"
  DeleteRegKey HKLM "${REGKEY}"
SectionEnd

; ── Helper: StrContains ────────────────────────────────────────────────────
; Stack: [haystack] [needle] → [match-start or ""]
; Searches haystack for needle; pushes "" if not found.
Function StrContains
  Exch $1   ; needle
  Exch
  Exch $0   ; haystack
  Push $2
  Push $3
  Push $4
  StrLen $3 $1         ; needle length
  StrLen $4 $0         ; haystack length
  IntOp $4 $4 - $3
  StrCpy $2 0
  loop:
    IntCmp $2 $4 done_notfound done_notfound 0
    StrCpy $R0 $0 $3 $2
    StrCmp $R0 $1 done_found 0
    IntOp $2 $2 + 1
    Goto loop
  done_found:
    StrCpy $R0 $0 "" $2
    Goto done
  done_notfound:
    StrCpy $R0 ""
  done:
  Pop $4
  Pop $3
  Pop $2
  Pop $0
  Pop $1
  Push $R0
FunctionEnd

; ── Helper: StrReplaceAll ──────────────────────────────────────────────────
; Stack (top→bottom): [haystack] [find] [replace] → [result]
Function StrReplaceAll
  Exch $2   ; replace
  Exch
  Exch $1   ; find
  Exch 2
  Exch $0   ; haystack
  Push $3   ; working copy
  Push $4   ; needle length
  Push $5   ; result accumulator
  Push $6   ; position
  StrCpy $5 ""
  StrLen $4 $1
  StrCpy $3 $0
  loop:
    StrLen $6 $3
    IntCmp $6 0 done 0 0
    StrCpy $6 $3 $4
    StrCmp $6 $1 found 0
      StrCpy $6 $3 1
      StrCpy $5 "$5$6"
      StrCpy $3 $3 "" 1
      Goto loop
    found:
      StrCpy $5 "$5$2"
      StrCpy $3 $3 "" $4
      Goto loop
  done:
  StrCpy $0 $5
  Pop $6
  Pop $5
  Pop $4
  Pop $3
  Pop $2
  Pop $1
  Exch $0
FunctionEnd
