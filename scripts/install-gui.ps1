[CmdletBinding()]
param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\go-socks2vpn",
    [switch]$SkipPathUpdate
)

$ErrorActionPreference = "Stop"
$repository = "santaklouse/go-socks2vpn"
$assetName = "go-socks2vpn-gui-windows-amd64.zip"

if (-not $IsWindows -and $PSVersionTable.PSEdition -eq "Core") {
    throw "This installer supports Windows only."
}

$architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
if ($architecture -eq "Arm64") {
    Write-Warning "On Windows ARM64, the amd64 build is installed using system x64 emulation."
} elseif ($architecture -ne "X64") {
    throw "Unsupported Windows architecture: $architecture"
}

if ($Version -eq "latest") {
    $releaseBase = "https://github.com/$repository/releases/latest/download"
} elseif ($Version -match '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$') {
    $releaseBase = "https://github.com/$repository/releases/download/$Version"
} else {
    throw "Version must be latest or a semantic version tag such as v1.0.0."
}

if ([Net.ServicePointManager]::SecurityProtocol -band [Net.SecurityProtocolType]::Tls12) {
    # TLS 1.2 is already enabled.
} else {
    [Net.ServicePointManager]::SecurityProtocol =
        [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
}

$tempDir = Join-Path ([IO.Path]::GetTempPath()) ("go-socks2vpn-gui-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    $archivePath = Join-Path $tempDir $assetName
    $checksumsPath = Join-Path $tempDir "SHA256SUMS"

    Write-Host "Downloading ${assetName}…"
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseBase/$assetName" -OutFile $archivePath
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseBase/SHA256SUMS" -OutFile $checksumsPath

    $escapedAssetName = [regex]::Escape($assetName)
    $checksumLine = Get-Content $checksumsPath |
        Where-Object { $_ -match "^[0-9A-Fa-f]{64}\s+\*?$escapedAssetName$" } |
        Select-Object -First 1
    if (-not $checksumLine) {
        throw "$assetName is missing from SHA256SUMS."
    }

    $expectedHash = ($checksumLine -split '\s+')[0].ToLowerInvariant()
    $actualHash = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        throw "The downloaded archive has an unexpected SHA-256."
    }
    Write-Host "SHA-256 verified: $actualHash"

    $extractDir = Join-Path $tempDir "extracted"
    Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force
    $sourceExecutable = Join-Path $extractDir "socks2vpn-gui.exe"
    if (-not (Test-Path -LiteralPath $sourceExecutable -PathType Leaf)) {
        throw "The archive does not contain socks2vpn-gui.exe."
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $installedExecutable = Join-Path $InstallDir "socks2vpn-gui.exe"
    Copy-Item -LiteralPath $sourceExecutable -Destination $installedExecutable -Force

    if (-not $SkipPathUpdate) {
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        $pathParts = @($userPath -split ';' | Where-Object { $_ })
        if (-not ($pathParts | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') })) {
            $newUserPath = (@($pathParts) + $InstallDir) -join ';'
            [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        }
        if (-not (($env:Path -split ';') | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') })) {
            $env:Path = "$env:Path;$InstallDir"
        }
    }

    $launcherPath = Join-Path $InstallDir "launch-socks2vpn-gui.ps1"
    $escapedExecutable = $installedExecutable.Replace("'", "''")
    Set-Content -LiteralPath $launcherPath -Encoding UTF8 -Value @"
[CmdletBinding()]
param([string]`$DeepLink)

`$ErrorActionPreference = "Stop"
if (`$DeepLink) {
    Start-Process -FilePath '$escapedExecutable' -ArgumentList @("--deep-link", `$DeepLink) -Verb RunAs
} else {
    Start-Process -FilePath '$escapedExecutable' -Verb RunAs
}
"@

    $powerShellPath = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
    $handlerCommand = "`"$powerShellPath`" -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$launcherPath`" -DeepLink `"%1`""
    foreach ($scheme in @("socks2vpn", "socks2vps")) {
        $schemePath = "HKCU:\Software\Classes\$scheme"
        New-Item -Path $schemePath -Force | Out-Null
        Set-Item -Path $schemePath -Value "URL:go-socks2vpn configuration link"
        New-ItemProperty -Path $schemePath -Name "URL Protocol" -Value "" -PropertyType String -Force | Out-Null
        $commandPath = Join-Path $schemePath "shell\open\command"
        New-Item -Path $commandPath -Force | Out-Null
        Set-Item -Path $commandPath -Value $handlerCommand
    }

    $programsDir = [Environment]::GetFolderPath("Programs")
    $shortcutPath = Join-Path $programsDir "go-socks2vpn.lnk"
    $shell = New-Object -ComObject WScript.Shell
    $shortcut = $shell.CreateShortcut($shortcutPath)
    $shortcut.TargetPath = $powerShellPath
    $shortcut.Arguments = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$launcherPath`""
    $shortcut.WorkingDirectory = $InstallDir
    $shortcut.IconLocation = "$installedExecutable,0"
    $shortcut.Description = "go-socks2vpn GUI"
    $shortcut.Save()

    Write-Host "GUI installed: $installedExecutable"
    Write-Host "Start menu shortcut created: go-socks2vpn"
    Write-Host "URL handlers registered: socks2vpn:// and socks2vps://"
    Write-Host "Windows will show the standard administrator privilege prompt at startup."
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
