Unicode true
!define WAILS_INSTALL_SCOPE "user"
!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName" "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "FileVersion" "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "LegalCopyright" "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "ProductName" "${INFO_PRODUCTNAME}"

ManifestDPIAware true
!include "MUI.nsh"
!define MUI_ICON "..\\icon.ico"
!define MUI_UNICON "..\\icon.ico"
!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "..\\..\\..\\bin\\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$LOCALAPPDATA\\Programs\\${INFO_PRODUCTNAME}"
ShowInstDetails show

Function .onInit
  !insertmacro wails.checkArchitecture
FunctionEnd

Section
  !insertmacro wails.setShellContext
  !insertmacro wails.webview2runtime
  SetOutPath $INSTDIR
  !insertmacro wails.files
  ; Register Torrade as an available handler. Windows requires the user to
  ; choose the default application through its Default Apps UI.
  WriteRegStr HKCU "Software\\Classes\\Torrade.Torrent" "" "Torrade torrent file"
  WriteRegStr HKCU "Software\\Classes\\Torrade.Torrent\\DefaultIcon" "" "$INSTDIR\\${PRODUCT_EXECUTABLE},0"
  WriteRegStr HKCU "Software\\Classes\\Torrade.Torrent\\shell\\open\\command" "" "$\"$INSTDIR\\${PRODUCT_EXECUTABLE}$\" $\"%1$\""
  WriteRegStr HKCU "Software\\Classes\\.torrent\\OpenWithProgids" "Torrade.Torrent" ""
  WriteRegStr HKCU "Software\\Classes\\magnet" "" "URL:Magnet Link"
  WriteRegStr HKCU "Software\\Classes\\magnet" "URL Protocol" ""
  WriteRegStr HKCU "Software\\Classes\\magnet\\DefaultIcon" "" "$INSTDIR\\${PRODUCT_EXECUTABLE},0"
  WriteRegStr HKCU "Software\\Classes\\magnet\\shell\\open\\command" "" "$\"$INSTDIR\\${PRODUCT_EXECUTABLE}$\" $\"%1$\""
  CreateShortcut "$SMPROGRAMS\\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\\${PRODUCT_EXECUTABLE}"
  CreateShortcut "$DESKTOP\\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\\${PRODUCT_EXECUTABLE}"
  !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
  !insertmacro wails.setShellContext
  RMDir /r "$AppData\\${PRODUCT_EXECUTABLE}"
  RMDir /r $INSTDIR
  Delete "$SMPROGRAMS\\${INFO_PRODUCTNAME}.lnk"
  Delete "$DESKTOP\\${INFO_PRODUCTNAME}.lnk"
  DeleteRegValue HKCU "Software\\Classes\\.torrent\\OpenWithProgids" "Torrade.Torrent"
  DeleteRegKey HKCU "Software\\Classes\\Torrade.Torrent"
  !insertmacro wails.deleteUninstaller
SectionEnd

