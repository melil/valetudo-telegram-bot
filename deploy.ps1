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
    go build -ldflags="-s -w" -o tgbot ./cmd/bot
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[-] Build failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    $sizeMB = [math]::Round(((Get-Item tgbot).Length / 1MB), 2)
    Write-Host "[+] Binary built successfully ($sizeMB MB)" -ForegroundColor Green
}

if (-not (Test-Path "tgbot")) {
    Write-Host "[-] Binary 'tgbot' not found!" -ForegroundColor Red
    exit 1
}

$Target = "$RobotUser@$RobotIP"

# 2. Stop running bot on robot
Write-Host "[*] Stopping tgbot on $Target..." -ForegroundColor Yellow
ssh -o ConnectTimeout=5 $Target "killall -9 tgbot 2>/dev/null || true"

# 3. Upload binary to robot
Write-Host "[*] Uploading tgbot to ${Target}:/data/tgbot/..." -ForegroundColor Cyan
scp -o ConnectTimeout=10 tgbot "${Target}:/data/tgbot/"
if ($LASTEXITCODE -ne 0) {
    Write-Host "[-] SCP upload failed!" -ForegroundColor Red
    exit $LASTEXITCODE
}

# 4. Start bot on robot
Write-Host "[*] Starting bot on robot..." -ForegroundColor Cyan
ssh -o ConnectTimeout=5 $Target "chmod +x /data/tgbot/tgbot; nohup /data/tgbot/run.sh > /tmp/log/custom/tgbot.log 2>&1 &"

Write-Host "[+] Deployment completed successfully!" -ForegroundColor Green
Write-Host "[i] View logs: ssh $Target tail -f /tmp/log/custom/tgbot.log" -ForegroundColor DarkGray
