# ==============================================================================
#  🤖 Valetudo Telegram Bot — Интерактивный мастер установки (PowerShell Wizard)
# ==============================================================================

$ErrorActionPreference = "Stop"

Set-Location $PSScriptRoot

Clear-Host

# ------------------------------------------------------------------------------
# Выбор локализации мастера (1: ru, 2: en, 3: de, 4: zh)
# ------------------------------------------------------------------------------
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "   Select language / Выберите язык мастера / 语言选择       " -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "  1) Русский (Russian) [По умолчанию]"
Write-Host "  2) English"
Write-Host "  3) Deutsch (German)"
Write-Host "  4) 简体中文 (Chinese)"
Write-Host ""
$langChoice = Read-Host "👉 Выбор / Choice [1-4, Default: 1]"

switch ($langChoice) {
    "2" {
        $langCode = "en"
        $title = "   🤖 Valetudo Telegram Bot — Quick Setup Wizard     "
        $intro = "This wizard will configure, build, and deploy the bot to your robot.`nPress [Enter] to accept default values in brackets.`n"
        $step1 = "[Step 1/5] Robot connection parameters:"
        $promptIP = "  👉 Robot IP address [192.168.1.91]"
        $promptUser = "  👉 SSH user [root]"
        $targetLabel = "Target:"
        $step2 = "[Step 2/5] Telegram settings:"
        $promptToken = "  👉 Bot Token (@BotFather)"
        $errToken = "     Token cannot be empty! Obtain it from @BotFather."
        $promptChat = "  👉 Your numeric Telegram ID (@userinfobot)"
        $errChat = "     ID cannot be empty! Obtain it from @userinfobot."
        $step3 = "[Step 3/5] Additional options:"
        $promptLang = "  👉 Bot interface language (ru/en/de/zh) [en]"
        $promptAuto = "  👉 Add bot to robot autostart on boot? [Y/n]"
        $step4 = "[Step 4/5] Environment check & binary build:"
        $errNoGo = "  Error: Go compiler not found! Install Go 1.22+ from https://go.dev/dl/"
        $foundGo = "  Found Go version:"
        $building = "  Compiling for Linux ARM64 (CGO_ENABLED=0)..."
        $errBuild = "  Error: build failed!"
        $buildOk = "  ✓ Binary built successfully"
        $step5 = "[Step 5/5] Deploying to robot"
        $checkSSH = "  Checking robot SSH accessibility..."
        $errSSH = "  Error: Cannot connect to robot via SSH!"
        $sshOk = "  ✓ SSH connection successful."
        $savingEnv = "  Creating /data/tgbot directory and saving .env configuration..."
        $envOk = "  ✓ Configuration saved to /data/tgbot/.env"
        $stopOld = "  Stopping old bot processes (if running)..."
        $uploading = "  Uploading files to robot via SCP..."
        $uploadOk = "  ✓ Files uploaded to /data/tgbot/"
        $configAuto = "  Configuring autostart in /data/_root.sh..."
        $autoOk = "  ✓ Autostart configured."
        $starting = "  Starting supervisor run.sh..."
        $finishTitle = "   🎉 Congratulations! Bot deployed and running!   "
        $finish1 = "1. Open your bot in Telegram and send the command: /start"
        $finish2 = "2. To view live logs in real time, run:"
        $defaultBotLang = "en"
    }
    "3" {
        $langCode = "de"
        $title = "   🤖 Valetudo Telegram Bot — Schnelleinrichtungsassistent  "
        $intro = "Dieser Assistent konfiguriert, baut und startet den Bot auf Ihrem Roboter.`nDrücken Sie [Enter], um die Standardwerte in Klammern zu übernehmen.`n"
        $step1 = "[Schritt 1/5] Roboter-Verbindungsparameter:"
        $promptIP = "  👉 Roboter-IP-Adresse [192.168.1.91]"
        $promptUser = "  👉 SSH-Benutzer [root]"
        $targetLabel = "Ziel:"
        $step2 = "[Schritt 2/5] Telegram-Einstellungen:"
        $promptToken = "  👉 Bot-Token (@BotFather)"
        $errToken = "     Token darf nicht leer sein! Holen Sie es sich bei @BotFather."
        $promptChat = "  👉 Ihre numerische Telegram-ID (@userinfobot)"
        $errChat = "     ID darf nicht leer sein! Finden Sie sie über @userinfobot."
        $step3 = "[Schritt 3/5] Zusätzliche Optionen:"
        $promptLang = "  👉 Sprache der Bot-Oberfläche (ru/en/de/zh) [de]"
        $promptAuto = "  👉 Bot zum Roboter-Autostart hinzufügen? [Y/n]"
        $step4 = "[Schritt 4/5] Umgebungsprüfung und Binärbau:"
        $errNoGo = "  Fehler: Go-Compiler nicht gefunden! Installieren Sie Go 1.22+ von https://go.dev/dl/"
        $foundGo = "  Gefundene Go-Version:"
        $building = "  Kompilierung für Linux ARM64 (CGO_ENABLED=0)..."
        $errBuild = "  Fehler: Build fehlgeschlagen!"
        $buildOk = "  ✓ Binärdatei erfolgreich gebaut"
        $step5 = "[Schritt 5/5] Bereitstellung auf dem Roboter"
        $checkSSH = "  Prüfe Roboter-SSH-Verbindung..."
        $errSSH = "  Fehler: Verbindung zum Roboter über SSH fehlgeschlagen!"
        $sshOk = "  ✓ SSH-Verbindung erfolgreich."
        $savingEnv = "  Erstelle Verzeichnis /data/tgbot und speichere .env..."
        $envOk = "  ✓ Konfiguration in /data/tgbot/.env gespeichert"
        $stopOld = "  Beende alte Bot-Prozesse (falls vorhanden)..."
        $uploading = "  Lade Dateien via SCP auf den Roboter..."
        $uploadOk = "  ✓ Dateien nach /data/tgbot/ übertragen"
        $configAuto = "  Richte Autostart in /data/_root.sh ein..."
        $autoOk = "  ✓ Autostart eingerichtet."
        $starting = "  Starte Supervisor run.sh..."
        $finishTitle = "   🎉 Glückwunsch! Bot erfolgreich bereitgestellt! "
        $finish1 = "1. Öffnen Sie Ihren Bot in Telegram und senden Sie: /start"
        $finish2 = "2. Um Live-Logs in Echtzeit anzuzeigen, führen Sie aus:"
        $defaultBotLang = "de"
    }
    "4" {
        $langCode = "zh"
        $title = "   🤖 Valetudo Telegram Bot — 快速安装向导          "
        $intro = "此向导将为您配置、编译并在扫地机上启动机器人。`n直接按 [Enter] 即可使用括号中的默认值。`n"
        $step1 = "[步骤 1/5] 扫地机连接参数："
        $promptIP = "  👉 扫地机 IP 地址 [192.168.1.91]"
        $promptUser = "  👉 SSH 用户名 [root]"
        $targetLabel = "目标主机："
        $step2 = "[步骤 2/5] Telegram 参数设置："
        $promptToken = "  👉 机器人 Token (@BotFather)"
        $errToken = "     Token 不能为空！请通过 @BotFather 获取。"
        $promptChat = "  👉 您的 Telegram 数字 ID (@userinfobot)"
        $errChat = "     ID 不能为空！可通过 @userinfobot 查询。"
        $step3 = "[步骤 3/5] 附加配置："
        $promptLang = "  👉 机器人界面语言 (ru/en/de/zh) [zh]"
        $promptAuto = "  👉 是否将机器人加入扫地机开机自启？[Y/n]"
        $step4 = "[步骤 4/5] 环境检查与编译："
        $errNoGo = "  错误：系统中未找到 Go 编译器！请从 https://go.dev/dl/ 安装 Go 1.22+"
        $foundGo = "  检测到 Go 版本："
        $building = "  正在编译 Linux ARM64 二进制文件 (CGO_ENABLED=0)..."
        $errBuild = "  错误：编译失败！"
        $buildOk = "  ✓ 二进制文件编译成功"
        $step5 = "[步骤 5/5] 部署到扫地机"
        $checkSSH = "  正在检查扫地机 SSH 连接..."
        $errSSH = "  错误：无法通过 SSH 连接到扫地机！"
        $sshOk = "  ✓ SSH 连接成功。"
        $savingEnv = "  正在创建 /data/tgbot 目录并写入 .env 配置文件..."
        $envOk = "  ✓ 配置已保存至 /data/tgbot/.env"
        $stopOld = "  正在停止旧的机器人进程（若有）..."
        $uploading = "  正在通过 SCP 上传文件到扫地机..."
        $uploadOk = "  ✓ 文件已上传至 /data/tgbot/"
        $configAuto = "  正在配置 /data/_root.sh 开机自启..."
        $autoOk = "  ✓ 开机自启已配置。"
        $starting = "  正在启动守护脚本 run.sh..."
        $finishTitle = "   🎉 恭喜！机器人已成功部署并运行！             "
        $finish1 = "1. 在 Telegram 中打开您的机器人并发送命令：/start"
        $finish2 = "2. 实时查看运行日志命令："
        $defaultBotLang = "zh"
    }
    default {
        $langCode = "ru"
        $title = "   🤖 Valetudo Telegram Bot — Мастер быстрой установки     "
        $intro = "Этот мастер настроит, скомпилирует и запустит бота на вашем роботе.`nДля выбора значений по умолчанию в скобках просто нажмите [Enter].`n"
        $step1 = "[Шаг 1/5] Параметры подключения к роботу:"
        $promptIP = "  👉 IP-адрес робота [192.168.1.91]"
        $promptUser = "  👉 SSH-пользователь [root]"
        $targetLabel = "Цель:"
        $step2 = "[Шаг 2/5] Telegram параметры:"
        $promptToken = "  👉 Токен бота (@BotFather)"
        $errToken = "     Токен не может быть пустым! Получите его у @BotFather."
        $promptChat = "  👉 Ваш числовой Telegram ID (@userinfobot)"
        $errChat = "     ID не может быть пустым! Узнайте его у @userinfobot."
        $step3 = "[Шаг 3/5] Дополнительные настройки:"
        $promptLang = "  👉 Язык интерфейса бота (ru/en/de/zh) [ru]"
        $promptAuto = "  👉 Добавить бота в автозагрузку робота при старте? [Y/n]"
        $step4 = "[Шаг 4/5] Подготовка и сборка бинарника:"
        $errNoGo = "  Ошибка: компилятор Go не найден в системе PATH! Установите Go 1.22+ с https://go.dev/dl/"
        $foundGo = "  Найдена версия Go:"
        $building = "  Компиляция для Linux ARM64 (CGO_ENABLED=0)..."
        $errBuild = "  Ошибка: сборка не удалась!"
        $buildOk = "  ✓ Бинарник собран успешно"
        $step5 = "[Шаг 5/5] Развертывание на робота"
        $checkSSH = "  Проверка доступности робота по SSH..."
        $errSSH = "  Ошибка: Не удалось подключиться к роботу по SSH!"
        $sshOk = "  ✓ Подключение по SSH успешно."
        $savingEnv = "  Создание каталога /data/tgbot и запись конфигурации .env..."
        $envOk = "  ✓ Конфигурация сохранена в /data/tgbot/.env"
        $stopOld = "  Остановка старых процессов бота (если запущены)..."
        $uploading = "  Загрузка файлов на робота через SCP..."
        $uploadOk = "  ✓ Файлы успешно загружены в /data/tgbot/"
        $configAuto = "  Настройка автозагрузки в /data/_root.sh..."
        $autoOk = "  ✓ Автозагрузка настроена."
        $starting = "  Запуск супервизора run.sh..."
        $finishTitle = "   🎉 Поздравляем! Бот успешно развернут и запущен!        "
        $finish1 = "1. Откройте вашего бота в Telegram и отправьте команду: /start"
        $finish2 = "2. Для просмотра логов в реальном времени выполните:"
        $defaultBotLang = "ru"
    }
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host $title -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host $intro

# ------------------------------------------------------------------------------
# Шаг 1: Опрос параметров
# ------------------------------------------------------------------------------
Write-Host $step1 -ForegroundColor Yellow
$RobotIP = Read-Host $promptIP
if ([string]::IsNullOrWhiteSpace($RobotIP)) { $RobotIP = "192.168.1.91" }

$RobotUser = Read-Host $promptUser
if ([string]::IsNullOrWhiteSpace($RobotUser)) { $RobotUser = "root" }

$Target = "$RobotUser@$RobotIP"
Write-Host "  $targetLabel $Target`n" -ForegroundColor Green

# ------------------------------------------------------------------------------
# Шаг 2: Telegram параметры
# ------------------------------------------------------------------------------
Write-Host $step2 -ForegroundColor Yellow
do {
    $BotToken = Read-Host $promptToken
    if ([string]::IsNullOrWhiteSpace($BotToken)) {
        Write-Host $errToken -ForegroundColor Red
    }
} while ([string]::IsNullOrWhiteSpace($BotToken))

do {
    $ChatID = Read-Host $promptChat
    if ([string]::IsNullOrWhiteSpace($ChatID)) {
        Write-Host $errChat -ForegroundColor Red
    }
} while ([string]::IsNullOrWhiteSpace($ChatID))

Write-Host ""

# ------------------------------------------------------------------------------
# Шаг 3: Дополнительные настройки
# ------------------------------------------------------------------------------
Write-Host $step3 -ForegroundColor Yellow
$BotLang = Read-Host $promptLang
if ([string]::IsNullOrWhiteSpace($BotLang)) { $BotLang = $defaultBotLang }

$Autostart = Read-Host $promptAuto
if ([string]::IsNullOrWhiteSpace($Autostart)) { $Autostart = "Y" }

Write-Host ""

# ------------------------------------------------------------------------------
# Шаг 4: Сборка бинарника
# ------------------------------------------------------------------------------
Write-Host $step4 -ForegroundColor Yellow

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host $errNoGo -ForegroundColor Red
    exit 1
}

$goVersion = (go version)
Write-Host "$foundGo $goVersion" -ForegroundColor Green
Write-Host $building -ForegroundColor Cyan

$env:GOOS = "linux"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "0"

go build -trimpath -ldflags="-s -w" -o tgbot ./cmd/bot

if ($LASTEXITCODE -ne 0 -or -not (Test-Path "tgbot")) {
    Write-Host $errBuild -ForegroundColor Red
    exit 1
}

$sizeMB = [math]::Round(((Get-Item tgbot).Length / 1MB), 2)
Write-Host "$buildOk (~$sizeMB MB)`n" -ForegroundColor Green

# ------------------------------------------------------------------------------
# Шаг 5: Развертывание
# ------------------------------------------------------------------------------
Write-Host "$step5 ($Target):" -ForegroundColor Yellow

Write-Host $checkSSH -ForegroundColor Cyan
ssh -o ConnectTimeout=5 $Target "echo ssh_ok" | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host $errSSH -ForegroundColor Red
    exit 1
}
Write-Host $sshOk -ForegroundColor Green

Write-Host $savingEnv -ForegroundColor Cyan
$envContent = @"
BOT_TOKEN=$BotToken
CHAT_ID=$ChatID
BOT_LANG=$BotLang
VALETUDO_BASE_URL=http://127.0.0.1/api/v2/robot
DB_PATH=/data/tgbot/bot.db
DND_ENABLED=true
"@
$envEscaped = $envContent.Replace("`r`n", "`n")
ssh -o ConnectTimeout=5 $Target "mkdir -p /data/tgbot && printf '%s' '$envEscaped' > /data/tgbot/.env"
Write-Host $envOk -ForegroundColor Green

Write-Host $stopOld -ForegroundColor Cyan
ssh -o ConnectTimeout=5 $Target "killall run.sh tgbot 2>/dev/null || true; sleep 1; killall -9 run.sh tgbot 2>/dev/null || true; rm -f /var/run/tgbot_run.pid /tmp/tgbot.alive"

Write-Host $uploading -ForegroundColor Cyan
scp -o ConnectTimeout=10 tgbot run.sh "${Target}:/data/tgbot/"
if ($LASTEXITCODE -ne 0) {
    Write-Host "  SCP error!" -ForegroundColor Red
    exit 1
}
Write-Host $uploadOk -ForegroundColor Green

if ($Autostart -match "^[Yy]") {
    Write-Host $configAuto -ForegroundColor Cyan
    ssh -o ConnectTimeout=5 $Target "grep -q '/data/tgbot/run.sh' /data/_root.sh 2>/dev/null || echo '/data/tgbot/run.sh &' >> /data/_root.sh; chmod +x /data/_root.sh 2>/dev/null || true"
    Write-Host $autoOk -ForegroundColor Green
}

Write-Host $starting -ForegroundColor Cyan
ssh -o ConnectTimeout=5 $Target "chmod +x /data/tgbot/tgbot /data/tgbot/run.sh; nohup /data/tgbot/run.sh >/dev/null 2>&1 &"

Start-Sleep -Seconds 2

# ------------------------------------------------------------------------------
# Финал
# ------------------------------------------------------------------------------
Write-Host ""
Write-Host "============================================================" -ForegroundColor Green
Write-Host $finishTitle -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Green
Write-Host ""
Write-Host $finish1 -ForegroundColor Yellow
Write-Host $finish2 -ForegroundColor DarkGray
Write-Host "   ssh $Target `"tail -f /tmp/log/custom/tgbot.log`"" -ForegroundColor Cyan
Write-Host ""
