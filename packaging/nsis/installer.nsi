; NSIS Installer for ccswresp (Windows)
; Build with:
;   makensis installer.nsi

!define PRODUCT_NAME "ccswresp"
!define PRODUCT_VERSION "1.0.0"
!define PRODUCT_PUBLISHER "hoganyu"
!define PRODUCT_WEB_SITE "https://github.com/hoganyu/ccswresp"
!define PRODUCT_DIR_REGKEY "Software\Microsoft\Windows\CurrentVersion\App Paths\ccswresp.exe"

SetCompressor lzma

; --- MUI ---
!include "MUI2.nsh"
!include "EnvVarUpdate.nsh"

!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "ccswresp-setup-${PRODUCT_VERSION}.exe"
InstallDir "$PROGRAMFILES\ccswresp"
InstallDirRegKey HKLM "${PRODUCT_DIR_REGKEY}" ""
ShowInstDetails show
ShowUnInstDetails show

Section "Install"
  SetOutPath "$INSTDIR"

  ; Copy all files
  File /r "..\..\cli.js"
  File /r "..\..\index.js"
  File /r "..\..\lib"
  File /r "..\..\package.json"
  File /r "..\..\env_example"
  File /r "..\..\scripts"

  ; Create wrapper batch script
  FileOpen $0 "$INSTDIR\ccswresp.bat" w
  FileWrite $0 '@echo off$\r$\n'
  FileWrite $0 'node "$INSTDIR\cli.js" %*$\r$\n'
  FileClose $0

  ; Add to PATH
  ${EnvVarUpdate} $0 "PATH" "A" "HKLM" "$INSTDIR"

  ; Register uninstaller
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "Publisher" "${PRODUCT_PUBLISHER}"

  ; Create uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Create start menu shortcuts
  CreateDirectory "$SMPROGRAMS\ccswresp"
  CreateShortCut "$SMPROGRAMS\ccswresp\ccswresp.lnk" "$INSTDIR\ccswresp.bat" "--help" "$INSTDIR\ccswresp.bat" 0
  CreateShortCut "$SMPROGRAMS\ccswresp\Uninstall.lnk" "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\ccswresp.bat"
  Delete "$INSTDIR\cli.js"
  Delete "$INSTDIR\index.js"
  RMDir /r "$INSTDIR\lib"
  Delete "$INSTDIR\package.json"
  Delete "$INSTDIR\env_example"
  RMDir /r "$INSTDIR\scripts"
  Delete "$INSTDIR\node_modules\*"
  RMDir "$INSTDIR\node_modules"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  ; Remove from PATH
  ${un.EnvVarUpdate} $0 "PATH" "R" "HKLM" "$INSTDIR"

  ; Remove shortcuts
  RMDir /r "$SMPROGRAMS\ccswresp"

  ; Remove registry entries
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
  DeleteRegKey HKLM "${PRODUCT_DIR_REGKEY}"
SectionEnd
