# ---------- Установщик nginx-lens (Windows) ----------
# Скачивает бинарники с GitHub Releases.

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\nginx-lens\bin",
    [string]$ConfigDir = "C:\ProgramData\nginx-lens"
)

$ErrorActionPreference = "Stop"
$Repo = "rainysundaynight/nginx-lens"

function Get-PlatformArch {
    $arch = switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { "amd64" }
        "ARM64" { "arm64" }
        default { throw "Неподдерживаемая архитектура: $($env:PROCESSOR_ARCHITECTURE)" }
    }
    return @{ Os = "Windows"; Arch = $arch }
}

function Resolve-Version {
    param([string]$Ver)
    if ($Ver -ne "latest") { return $Ver.TrimStart("v") }
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    return $release.tag_name.TrimStart("v")
}

$plat = Get-PlatformArch
$ver = Resolve-Version -Ver $Version
$archive = "nginx-lens_${ver}_$($plat.Os)_$($plat.Arch).zip"
$url = "https://github.com/$Repo/releases/download/v$ver/$archive"
$tmp = Join-Path $env:TEMP "nginx-lens-install"
New-Item -ItemType Directory -Force -Path $tmp, $InstallDir, $ConfigDir | Out-Null

Write-Host "Скачивание $url ..."
Invoke-WebRequest -Uri $url -OutFile (Join-Path $tmp $archive)
Expand-Archive -Path (Join-Path $tmp $archive) -DestinationPath $tmp -Force

foreach ($bin in @("nginx-lens.exe", "nginx-lens-agent.exe", "nginx-lens-hub.exe")) {
    $src = Join-Path $tmp $bin
    if (Test-Path $src) {
        Copy-Item $src (Join-Path $InstallDir $bin) -Force
        Write-Host "Установлен: $InstallDir\$bin"
    }
}

$example = Join-Path $tmp "example-config.yaml"
$configPath = Join-Path $ConfigDir "config.yaml"
if ((Test-Path $example) -and -not (Test-Path $configPath)) {
    Copy-Item $example $configPath
    Write-Host "Конфиг: $configPath"
}

$env:Path = "$InstallDir;$env:Path"
[Environment]::SetEnvironmentVariable("NGINX_LENS_CONFIG", $configPath, "User")

Write-Host ""
Write-Host "Готово. Добавьте в PATH: $InstallDir"
Write-Host "  nginx-lens init"
Write-Host "  nginx-lens config validate"
