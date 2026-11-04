param([switch]$DesktopShortcut)
$ErrorActionPreference = 'Stop'
$exe = Join-Path $PSScriptRoot 'freeturn-desktop.exe'
if (!(Test-Path -LiteralPath $exe)) { throw 'freeturn-desktop.exe not found beside installer' }
$process = Start-Process -FilePath $exe -ArgumentList '--install-desktop' -Verb RunAs -Wait -PassThru
if ($process.ExitCode -ne 0) { throw "Installation failed ($($process.ExitCode))" }
if (!$DesktopShortcut) {
    $DesktopShortcut = (Read-Host 'Create a desktop shortcut? / Создать ярлык на рабочем столе? [y/N]') -match '^(y|yes|д|да)$'
}
if ($DesktopShortcut) {
    $process = Start-Process -FilePath $exe -ArgumentList '--create-desktop-shortcut' -Verb RunAs -Wait -PassThru
    if ($process.ExitCode -ne 0) { throw 'Shortcut creation failed' }
}
Write-Host 'FreeTurn installed. Profiles and server settings were not changed.'
