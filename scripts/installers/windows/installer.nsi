Unicode true

!include "MUI2.nsh"
!include "FileFunc.nsh"

!ifndef APP_NAME
    !define APP_NAME "Hamster Wheel"
!endif
!ifndef PUBLISHER
    !define PUBLISHER "Hamster Wheel"
!endif
!ifndef APP_VERSION
    !define APP_VERSION "0.1.0"
!endif
!ifndef PRODUCT_CODE
    !define PRODUCT_CODE "HamsterWheel"
!endif
!ifndef APP_EXE
    !error "APP_EXE is required. Example: /DAPP_EXE=C:\path\to\hamster-wheel.exe"
!endif
!ifndef APP_EXE_NAME
    !define APP_EXE_NAME "hamster-wheel.exe"
!endif
!ifndef OUTPUT_FILE
    !define OUTPUT_FILE "hamster-wheel-installer.exe"
!endif

Name "${APP_NAME}"
OutFile "${OUTPUT_FILE}"
InstallDir "$PROGRAMFILES64\${PUBLISHER}\${APP_NAME}"
RequestExecutionLevel admin
ShowInstDetails show
ShowUninstDetails show

# NSIS version resources require four numeric segments.
VIProductVersion "${APP_VERSION}.0"
VIFileVersion "${APP_VERSION}.0"
VIAddVersionKey "CompanyName" "${PUBLISHER}"
VIAddVersionKey "FileDescription" "${APP_NAME} Installer"
VIAddVersionKey "ProductVersion" "${APP_VERSION}"
VIAddVersionKey "FileVersion" "${APP_VERSION}"
VIAddVersionKey "ProductName" "${APP_NAME}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Install"
    SetOutPath "$INSTDIR"
    File "/oname=${APP_EXE_NAME}" "${APP_EXE}"

    !ifdef WEBVIEW2_BOOTSTRAPPER
        File "/oname=MicrosoftEdgeWebview2Setup.exe" "${WEBVIEW2_BOOTSTRAPPER}"
        ExecWait '"$INSTDIR\MicrosoftEdgeWebview2Setup.exe" /silent /install'
        Delete "$INSTDIR\MicrosoftEdgeWebview2Setup.exe"
    !endif

    CreateDirectory "$SMPROGRAMS\${APP_NAME}"
    CreateShortcut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE_NAME}"
    CreateShortcut "$SMPROGRAMS\${APP_NAME}\Uninstall ${APP_NAME}.lnk" "$INSTDIR\Uninstall.exe"
    CreateShortcut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE_NAME}"

    WriteUninstaller "$INSTDIR\Uninstall.exe"

    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_CODE}" "DisplayName" "${APP_NAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_CODE}" "DisplayVersion" "${APP_VERSION}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_CODE}" "Publisher" "${PUBLISHER}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_CODE}" "DisplayIcon" "$INSTDIR\${APP_EXE_NAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_CODE}" "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_CODE}" "QuietUninstallString" "$\"$INSTDIR\Uninstall.exe$\" /S"

    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_CODE}" "EstimatedSize" "$0"
SectionEnd

Section "Uninstall"
    Delete "$DESKTOP\${APP_NAME}.lnk"
    Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk"
    Delete "$SMPROGRAMS\${APP_NAME}\Uninstall ${APP_NAME}.lnk"
    RMDir "$SMPROGRAMS\${APP_NAME}"

    Delete "$INSTDIR\${APP_EXE_NAME}"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir "$INSTDIR"

    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_CODE}"
SectionEnd
