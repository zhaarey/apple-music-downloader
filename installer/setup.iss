; Inno Setup script for Aura Audio Downloader
; Requires Inno Setup 6: https://jrsoftware.org/isinfo.php

#define MyAppName "Aura Audio Downloader"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "Aura Audio Downloader"
#define MyAppExeName "AuraAudioDownloader.exe"

[Setup]
AppId={{B2C3D4E5-F6A7-8901-BCDE-F12345678901}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\AuraAudioDownloader
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=AuraAudioDownloader-Setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "..\dist\AuraAudioDownloader.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\aura.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\amd.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\tools\*"; DestDir: "{app}\tools"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\dist\scripts\*"; DestDir: "{app}\scripts"; Flags: ignoreversion
Source: "..\config.yaml.example"; DestDir: "{userappdata}\AuraAudioDownloader"; DestName: "config.yaml.example"; Flags: ignoreversion onlyifdoesntexist
Source: "..\README-WINDOWS.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\CLI (aura)"; Filename: "{app}\aura.exe"
Name: "{group}\CLI (amd, legacy)"; Filename: "{app}\amd.exe"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
    ForceDirectories(ExpandConstant('{userappdata}\AuraAudioDownloader'));
end;
