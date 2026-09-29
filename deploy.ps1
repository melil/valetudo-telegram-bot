[CmdletBinding()]
param (
    [string]$RobotIP = "192.168.1.91",
    [string]$RobotUser = "root",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

# 1. Enter project directory
Set-Location $PSScriptRoot
Write-Host "[*] Working directory: $PSScriptRoot" -ForegroundColor Cyan

# Build for Linux ARM64
if (-not $SkipBuild) {
    Write-Host "[*] Building tgbot for Linux ARM64..." -ForegroundColor Cyan
    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "0"
    $gitTag = (git describe --tags --always 2>$null)
    if (-not $gitTag) { $gitTag = "1.0.0" }
    $gitCommit = (git rev-parse --short HEAD 2>$null)
    $buildDate = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
    $ldFlags = "-s -w -X 'tgbot/internal/version.Version=$gitTag' -X 'tgbot/internal/version.Commit=$gitCommit' -X 'tgbot/internal/version.BuildDate=$buildDate'"
    go build -trimpath -ldflags="$ldFlags" -o tgbot ./cmd/bot
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[-] Build failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    $sizeMB = [math]::Round(((Get-Item tgbot).Length / 1MB), 2)
    Write-Host "[+] Binary built successfully ($sizeMB MB, version: $gitTag)" -ForegroundColor Green
}

if (-not (Test-Path "tgbot")) {
    Write-Host "[-] Binary 'tgbot' not found!" -ForegroundColor Red
    exit 1
}

$Target = "$RobotUser@$RobotIP"

# 2. Stop running bot and supervisor on robot
Write-Host "[*] Stopping tgbot on $Target..." -ForegroundColor Yellow
ssh -o ConnectTimeout=5 $Target "killall run.sh tgbot 2>/dev/null || true; rm -f /var/run/tgbot_run.pid"

# 3. Upload binary and supervisor script to robot
Write-Host "[*] Uploading tgbot and run.sh to ${Target}:/data/tgbot/..." -ForegroundColor Cyan
scp -o ConnectTimeout=10 tgbot run.sh "${Target}:/data/tgbot/"
if ($LASTEXITCODE -ne 0) {
    Write-Host "[-] SCP upload failed!" -ForegroundColor Red
    exit $LASTEXITCODE
}

# 4. Start supervisor on robot
Write-Host "[*] Starting bot supervisor on robot..." -ForegroundColor Cyan
ssh -o ConnectTimeout=5 $Target "chmod +x /data/tgbot/tgbot /data/tgbot/run.sh; nohup /data/tgbot/run.sh >/dev/null 2>&1 &"

Write-Host "[+] Deployment completed successfully!" -ForegroundColor Green
Write-Host "[i] View logs: ssh $Target tail -f /tmp/log/custom/tgbot.log" -ForegroundColor DarkGray
