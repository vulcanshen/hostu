# hostu uninstaller for Windows.
# Usage: irm https://raw.githubusercontent.com/vulcanshen/hostu/main/uninstall.ps1 | iex
#
# hostu's own (tdp project: an app with a Windows build writes it): the same
# steps as uninstall.sh. The hosts file stays as hostu last wrote it.

$ErrorActionPreference = "Stop"

$app = "hostu"
$installDir = Join-Path $env:LOCALAPPDATA $app
$exe = Join-Path $installDir "$app.exe"

if (-not (Test-Path $exe)) {
    Write-Host "$app not found in $installDir" -ForegroundColor Red
    exit 1
}
Remove-Item $exe -Force
Write-Host "removed $exe"
if (-not (Get-ChildItem $installDir -Force)) {
    Remove-Item $installDir -Force
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ";") -contains $installDir) {
    $newPath = ($userPath -split ";" | Where-Object { $_ -ne $installDir }) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "removed $installDir from your PATH"
}

# Ask offers to remove one directory, kept unless the answer is yes.
function Ask($dir, $what) {
    if (-not (Test-Path $dir)) { return }
    $answer = Read-Host "Remove $what in $dir? [y/N]"
    if ($answer -match "^(y|yes)$") {
        Remove-Item $dir -Recurse -Force
        Write-Host "removed $dir"
    } else {
        Write-Host "kept $dir"
    }
}

# Where tdp E3 puts the settings and the data; HOSTU__CONFIG and HOSTU__DATA move them.
$configDir = if ($env:HOSTU__CONFIG) { $env:HOSTU__CONFIG } elseif ($env:XDG_CONFIG_HOME) { Join-Path $env:XDG_CONFIG_HOME $app } else { Join-Path $env:USERPROFILE ".config\$app" }
$dataDir = if ($env:HOSTU__DATA) { $env:HOSTU__DATA } else { Join-Path $env:USERPROFILE ".$app" }
Ask $configDir "$app settings"
Ask $dataDir "$app data (the profiles, and the hosts file as it was before $app first wrote it)"

Write-Host ""
Write-Host "$app uninstalled. The hosts file is as $app last wrote it."
