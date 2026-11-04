$ErrorActionPreference = 'Stop'
$answer = Read-Host 'Remove FreeTurn from this PC? Profiles and VPS are kept. / Удалить клиент с ПК? Профили и VPS сохранятся. [y/N]'
if ($answer -notmatch '^(y|yes|д|да)$') { exit 0 }
$exe = Join-Path $env:LOCALAPPDATA 'Programs/FreeTurn/freeturn-desktop.exe'
if (!(Test-Path -LiteralPath $exe)) { throw 'Managed FreeTurn installation not found' }
$process = Start-Process -FilePath $exe -ArgumentList '--uninstall-desktop' -Verb RunAs -Wait -PassThru
if ($process.ExitCode -ne 0) { throw "Uninstallation failed ($($process.ExitCode))" }
Write-Host 'Client removal requested. Profiles, keys and VPS are kept.'
