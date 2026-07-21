!include "MUI2.nsh"

!ifndef APP_VERSION
  !define APP_VERSION "1.0.0"
!endif

!define APP_NAME    "PrintBridge"
!define EXE_NAME    "printbridge.exe"
!define REG_RUN     "Software\Microsoft\Windows\CurrentVersion\Run"
!define REG_UNINST  "Software\Microsoft\Windows\CurrentVersion\Uninstall\PrintBridge"

Name            "${APP_NAME} ${APP_VERSION}"
OutFile         "PrintBridge-Setup.exe"
InstallDir      "$LOCALAPPDATA\Programs\PrintBridge"
InstallDirRegKey HKCU "${REG_UNINST}" "InstallLocation"
RequestExecutionLevel user
Unicode True

; Installer pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN         "$INSTDIR\${EXE_NAME}"
!define MUI_FINISHPAGE_RUN_TEXT    "Iniciar PrintBridge ahora"
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Spanish"

Section "PrintBridge" SEC_MAIN
    SetOutPath "$INSTDIR"
    File "printbridge.exe"

    ; Startup entry
    WriteRegStr HKCU "${REG_RUN}" "PrintBridge" '"$INSTDIR\${EXE_NAME}"'

    ; Add/Remove Programs entry
    WriteUninstaller "$INSTDIR\Uninstall.exe"
    WriteRegStr   HKCU "${REG_UNINST}" "DisplayName"     "${APP_NAME}"
    WriteRegStr   HKCU "${REG_UNINST}" "DisplayVersion"  "${APP_VERSION}"
    WriteRegStr   HKCU "${REG_UNINST}" "Publisher"       "GC"
    WriteRegStr   HKCU "${REG_UNINST}" "InstallLocation" "$INSTDIR"
    WriteRegStr   HKCU "${REG_UNINST}" "UninstallString" '"$INSTDIR\Uninstall.exe" /S'
    WriteRegDWORD HKCU "${REG_UNINST}" "NoModify"        1
    WriteRegDWORD HKCU "${REG_UNINST}" "NoRepair"        1
SectionEnd

Section "Uninstall"
    ; Stop the service if running
    ExecWait 'taskkill /F /IM "${EXE_NAME}"'

    DeleteRegValue HKCU "${REG_RUN}"   "PrintBridge"
    DeleteRegKey   HKCU "${REG_UNINST}"

    Delete "$INSTDIR\${EXE_NAME}"
    Delete "$INSTDIR\config.json"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir  "$INSTDIR"
SectionEnd
