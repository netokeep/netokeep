# Install the NetoKeep client (nk.exe) on Windows from GitHub releases.
#   irm https://raw.githubusercontent.com/netokeep/netokeep/main/scripts/install.ps1 | iex
# The server (nks) is Linux only, so only the client is installed here.

function Install-NetoKeep {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'  # Invoke-WebRequest is very slow with a progress bar
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

    $BinDir = if ($env:NK_BIN_DIR) { $env:NK_BIN_DIR } else { Join-Path $HOME '.local\bin' }
    $Repo = 'https://github.com/netokeep/netokeep'
    $Dest = Join-Path $BinDir 'nk.exe'

    # PROCESSOR_ARCHITEW6432 is set when 32-bit PowerShell runs on 64-bit Windows
    $arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    if ($arch -ne 'AMD64') {
        Write-Host "Error: Unsupported architecture: $arch" -ForegroundColor Red
        return
    }

    # Compare version if an old version exists
    if (Test-Path $Dest) {
        $installed = (& $Dest --version) -replace '^.*version\s+', ''
        try {
            # /releases/latest redirects to /releases/tag/v<version>
            $r = Invoke-WebRequest "$Repo/releases/latest" -Method Head -UseBasicParsing
            # Windows PowerShell 5.1 exposes ResponseUri; PowerShell 7 exposes RequestMessage.RequestUri
            $final = $r.BaseResponse.ResponseUri
            if (-not $final) { $final = $r.BaseResponse.RequestMessage.RequestUri }
            $latest = "$final" -replace '^.*/v', ''
        } catch { $latest = '' }

        if ($latest -and $latest -eq $installed) {
            Write-Host "$([char]0x2714) NetoKeep is already up-to-date (version $installed)." -ForegroundColor Green
            return
        } elseif ($latest) {
            Write-Host "`nA newer version of NetoKeep is available (installed: $installed, latest: $latest).`n" -ForegroundColor Cyan
        } else {
            Write-Host 'Could not determine the latest version; continuing with the download.' -ForegroundColor Yellow
        }
    }

    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
    $tmp = "$Dest.download"
    Write-Host '==> Downloading Client...' -ForegroundColor Cyan
    try {
        Invoke-WebRequest "$Repo/releases/latest/download/netokeep-client-windows-amd64.exe" -OutFile $tmp -UseBasicParsing
    } catch {
        Remove-Item $tmp -Force -ErrorAction SilentlyContinue
        Write-Host "Error: download failed: $($_.Exception.Message)" -ForegroundColor Red
        return
    }

    # A running exe cannot be overwritten, but it can be renamed: move it aside, then put the new one in place.
    # Old copies that are no longer running are cleaned up here or on the next install.
    Get-ChildItem "$Dest.*.old" -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
    if (Test-Path $Dest) {
        Rename-Item $Dest ("nk.exe.{0}.old" -f [guid]::NewGuid().ToString('N').Substring(0, 8))
    }
    Move-Item $tmp $Dest

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

# A function + return keeps `irm | iex` from closing the user's window on errors (exit would).
Install-NetoKeep
