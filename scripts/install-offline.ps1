# Install the NetoKeep client (nk.exe) on Windows from a binary placed next to this script.
#   powershell -ExecutionPolicy Bypass -File install-offline.ps1
# Put netokeep-client-windows-amd64.exe in the same folder as this script.

function Install-NetoKeep {
    $ErrorActionPreference = 'Stop'

    $BinDir = if ($env:NK_BIN_DIR) { $env:NK_BIN_DIR } else { Join-Path $HOME '.local\bin' }
    $Dest = Join-Path $BinDir 'nk.exe'
    $Src = Join-Path $PSScriptRoot 'netokeep-client-windows-amd64.exe'

    if (-not (Test-Path $Src)) {
        Write-Host "Error: netokeep-client-windows-amd64.exe was not found in`n`t$PSScriptRoot" -ForegroundColor Red
        return
    }

    # Refuse to install while any nk.exe is running, wherever it lives (users may move it): stop it first
    $running = @(Get-Process -Name nk -ErrorAction SilentlyContinue)
    if ($running.Count -gt 0) {
        Write-Host "Error: nk is running (PID: $($running.Id -join ', ')). Stop it first with nk stop <name>, then rerun the installer." -ForegroundColor Red
        return
    }

    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
    Write-Host '==> Installing Client...' -ForegroundColor Cyan
    # Copy next to the target first, so the user's file is left in place
    $tmp = "$Dest.download"
    Copy-Item $Src $tmp -Force

    Move-Item $tmp $Dest -Force

    $version = (& $Dest --version) -replace '^.*version\s+', ''
    Write-Host "`n$([char]0x2714) NetoKeep successfully installed!`n" -ForegroundColor Green
    Write-Host "    Version: $version`n"
    Write-Host "    Location: $BinDir`n"
    Write-Host 'Next: Run nk --help to get started.'

    # Detect whether the bin directory is in the user's PATH
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (($userPath -split ';') -notcontains $BinDir) {
        Write-Host "`nWarning: $BinDir is not in your PATH." -ForegroundColor Yellow
        Write-Host '    Run this once, then open a new terminal:'
        Write-Host "    [Environment]::SetEnvironmentVariable('Path', `"`$([Environment]::GetEnvironmentVariable('Path', 'User'));$BinDir`", 'User')" -ForegroundColor Cyan
    }
}

Install-NetoKeep
