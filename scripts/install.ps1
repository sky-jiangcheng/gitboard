# GitBoard install script for Windows
# Run in PowerShell: iwr -useb https://raw.githubusercontent.com/sky-jiangcheng/gitboard/master/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

$InstallDir = "$env:LOCALAPPDATA\GitBoard"
$BinaryName = "gitboard.exe"
$Repo = "sky-jiangcheng/gitboard"
$Target = "windows-amd64"

Write-Host "Downloading GitBoard for Windows..."
$DownloadUrl = "https://github.com/$Repo/releases/latest/download/gitboard-$Target.exe"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Invoke-WebRequest -Uri $DownloadUrl -OutFile "$InstallDir\$BinaryName"

# Add to PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to PATH"
}

Write-Host ""
Write-Host "GitBoard installed to $InstallDir"
Write-Host "Run 'gitboard' in a new terminal to start!"
Write-Host ""
Write-Host "You can also create a desktop shortcut to: $InstallDir\$BinaryName"
