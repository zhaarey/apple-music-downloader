; Inno Setup script for Apple Music Downloader
; Requires Inno Setup 6: https://jrsoftware.org/isinfo.php

#define MyAppName "Apple Music Downloader"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "Apple Music Downloader"
#define MyAppExeName "AppleMusicDownloader.exe"

[Setup]
AppId={{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\AppleMusicDownloader
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=AppleMusicDownloader-Setup
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
Source: "..\dist\AppleMusicDownloader.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\amd.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\tools\*"; DestDir: "{app}\tools"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\config.yaml.example"; DestDir: "{userappdata}\AppleMusicDownloader"; DestName: "config.yaml.example"; Flags: ignoreversion onlyifdoesntexist
Source: "..\README-WINDOWS.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\CLI (amd)"; Filename: "{app}\amd.exe"
Name: "{group}\Windows guide"; Filename: "{app}\README-WINDOWS.md"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    ForceDirectories(ExpandConstant('{userappdata}\AppleMusicDownloader'));
  end;
end;
