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
    throw "Этот установщик предназначен только для Windows."
}

$architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
if ($architecture -eq "Arm64") {
    Write-Warning "Для Windows ARM64 устанавливается amd64-версия через системную x64-эмуляцию."
} elseif ($architecture -ne "X64") {
    throw "Неподдерживаемая архитектура Windows: $architecture"
}

if ($Version -eq "latest") {
    $releaseBase = "https://github.com/$repository/releases/latest/download"
} elseif ($Version -match '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$') {
    $releaseBase = "https://github.com/$repository/releases/download/$Version"
} else {
    throw "Version должна быть latest или семантическим тегом вида v1.0.0."
}

if ([Net.ServicePointManager]::SecurityProtocol -band [Net.SecurityProtocolType]::Tls12) {
    # TLS 1.2 уже включён.
} else {
    [Net.ServicePointManager]::SecurityProtocol =
        [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
}

$tempDir = Join-Path ([IO.Path]::GetTempPath()) ("go-socks2vpn-gui-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    $archivePath = Join-Path $tempDir $assetName
    $checksumsPath = Join-Path $tempDir "SHA256SUMS"

    Write-Host "Скачиваю ${assetName}…"
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseBase/$assetName" -OutFile $archivePath
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseBase/SHA256SUMS" -OutFile $checksumsPath

    $escapedAssetName = [regex]::Escape($assetName)
    $checksumLine = Get-Content $checksumsPath |
        Where-Object { $_ -match "^[0-9A-Fa-f]{64}\s+\*?$escapedAssetName$" } |
        Select-Object -First 1
    if (-not $checksumLine) {
        throw "$assetName отсутствует в SHA256SUMS."
    }

    $expectedHash = ($checksumLine -split '\s+')[0].ToLowerInvariant()
    $actualHash = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        throw "SHA-256 скачанного архива не совпадает."
    }
    Write-Host "SHA-256 подтверждён: $actualHash"

    $extractDir = Join-Path $tempDir "extracted"
    Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force
    $sourceExecutable = Join-Path $extractDir "socks2vpn-gui.exe"
    if (-not (Test-Path -LiteralPath $sourceExecutable -PathType Leaf)) {
        throw "Архив не содержит socks2vpn-gui.exe."
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
`$ErrorActionPreference = "Stop"
Start-Process -FilePath '$escapedExecutable' -Verb RunAs
"@

    $programsDir = [Environment]::GetFolderPath("Programs")
    $shortcutPath = Join-Path $programsDir "go-socks2vpn.lnk"
    $shell = New-Object -ComObject WScript.Shell
    $shortcut = $shell.CreateShortcut($shortcutPath)
    $shortcut.TargetPath = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
    $shortcut.Arguments = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$launcherPath`""
    $shortcut.WorkingDirectory = $InstallDir
    $shortcut.IconLocation = "$installedExecutable,0"
    $shortcut.Description = "go-socks2vpn GUI"
    $shortcut.Save()

    Write-Host "GUI установлен: $installedExecutable"
    Write-Host "Ярлык создан в меню Пуск: go-socks2vpn"
    Write-Host "При запуске Windows покажет стандартный запрос прав администратора."
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
