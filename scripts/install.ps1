# Git Dashboard install script for Windows
# Run in PowerShell: iwr -useb https://raw.githubusercontent.com/sky-jiangcheng/CodeStat/master/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

$InstallDir = "$env:LOCALAPPDATA\GitDashboard"
$BinaryName = "git-dashboard.exe"
$Repo = "sky-jiangcheng/CodeStat"
$Target = "windows-amd64"

Write-Host "Downloading Git Dashboard for Windows..."
$DownloadUrl = "https://github.com/$Repo/releases/latest/download/git-dashboard-$Target.exe"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Invoke-WebRequest -Uri $DownloadUrl -OutFile "$InstallDir\$BinaryName"

# Add to PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to PATH"
}

Write-Host ""
Write-Host "Git Dashboard installed to $InstallDir"
Write-Host "Run 'git-dashboard' in a new terminal to start!"
Write-Host ""
Write-Host "You can also create a desktop shortcut to: $InstallDir\$BinaryName"
