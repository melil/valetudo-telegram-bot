# ==============================================================================
#  🤖 Valetudo Telegram Bot — Онлайн-установщик для Windows (PowerShell)
#  Использование:
#    irm https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.ps1 | iex
#    $Version="1.0.13"; irm https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.ps1 | iex
# ==============================================================================

$ErrorActionPreference = "Stop"

$Repo = "melil/valetudo-telegram-bot"
$TargetVer = if ($Version) { $Version } elseif ($args.Count -gt 0) { $args[0] } else { "latest" }

Clear-Host

# ------------------------------------------------------------------------------
# Шаг 0: Выбор языка мастера
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
        $title = "   🤖 Valetudo Telegram Bot — Quick Installer (Windows) "
        $intro = "This installer will download, configure and deploy the bot to your robot.`nPress [Enter] to accept default values.`n"
        $step1 = "[Step 1/4] Robot connection settings:"
        $promptIP = "  👉 Robot IP address [192.168.1.91]"
        $promptUser = "  👉 SSH user [root]"
        $targetLabel = "Target:"
        $step2 = "[Step 2/4] Telegram parameters:"
        $promptToken = "  👉 Bot Token (@BotFather)"
        $errToken = "     Token cannot be empty! Obtain it from @BotFather."
        $promptChat = "  👉 Your numeric Telegram ID (@userinfobot)"
        $errChat = "     ID cannot be empty! Obtain it from @userinfobot."
        $step3 = "[Step 3/4] Additional options:"
        $promptLang = "  👉 Bot interface language (ru/en/de/zh) [en]"
        $promptAuto = "  👉 Add bot to robot autostart on boot? [Y/n]"
        $step4 = "[Step 4/4] Downloading and deploying:"
        $msgDownBin = "  Downloading prebuilt tgbot binary ($TargetVer)..."
        $msgDownRun = "  Downloading supervisor script run.sh..."
        $msgSSH = "  Checking robot SSH accessibility..."
        $errSSH = "  Error: Cannot connect to robot via SSH!"
        $msgSaving = "  Creating /data/tgbot and saving .env on robot..."
        $msgUpload = "  Uploading files to robot via SCP..."
        $msgAuto = "  Configuring autostart in /data/_root.sh..."
        $msgStarting = "  Starting supervisor run.sh..."
        $finishTitle = "   🎉 Congratulations! Bot deployed and running!   "
        $finish1 = "1. Open your bot in Telegram and send the command: /start"
        $finish2 = "2. To view live logs in real time, run:"
        $defaultBotLang = "en"
    }
    "3" {
        $title = "   🤖 Valetudo Telegram Bot — Schnell-Installer (Windows) "
        $intro = "Dieser Installer lädt den Bot herunter, konfiguriert und installiert ihn.`nDrücken Sie [Enter], um Standardwerte zu übernehmen.`n"
        $step1 = "[Schritt 1/4] Roboter-Verbindungsparameter:"
        $promptIP = "  👉 Roboter-IP-Adresse [192.168.1.91]"
        $promptUser = "  👉 SSH-Benutzer [root]"
        $targetLabel = "Ziel:"
        $step2 = "[Schritt 2/4] Telegram-Einstellungen:"
        $promptToken = "  👉 Bot-Token (@BotFather)"
        $errToken = "     Token darf nicht leer sein! Holen Sie es sich bei @BotFather."
        $promptChat = "  👉 Ihre numerische Telegram-ID (@userinfobot)"
        $errChat = "     ID darf nicht leer sein! Finden Sie sie über @userinfobot."
        $step3 = "[Schritt 3/4] Zusätzliche Optionen:"
        $promptLang = "  👉 Sprache der Bot-Oberfläche (ru/en/de/zh) [de]"
        $promptAuto = "  👉 Bot zum Roboter-Autostart hinzufügen? [Y/n]"
        $step4 = "[Schritt 4/4] Herunterladen und Bereitstellen:"
        $msgDownBin = "  Lade tgbot-Binärdatei herunter ($TargetVer)..."
        $msgDownRun = "  Lade Supervisor run.sh herunter..."
        $msgSSH = "  Prüfe Roboter-SSH-Verbindung..."
        $errSSH = "  Fehler: Verbindung zum Roboter über SSH fehlgeschlagen!"
        $msgSaving = "  Erstelle /data/tgbot und speichere .env..."
        $msgUpload = "  Lade Dateien via SCP auf den Roboter..."
        $msgAuto = "  Richte Autostart in /data/_root.sh ein..."
        $msgStarting = "  Starte Supervisor run.sh..."
        $finishTitle = "   🎉 Glückwunsch! Bot erfolgreich bereitgestellt! "
        $finish1 = "1. Öffnen Sie Ihren Bot in Telegram und senden Sie: /start"
        $finish2 = "2. Um Live-Logs in Echtzeit anzuzeigen, führen Sie aus:"
        $defaultBotLang = "de"
    }
    "4" {
        $title = "   🤖 Valetudo Telegram Bot — Windows 在线安装向导   "
        $intro = "此安装向导将为您下载、配置并将机器人部署到扫地机。`n直接按 [Enter] 即可使用默认值。`n"
        $step1 = "[步骤 1/4] 扫地机连接参数："
        $promptIP = "  👉 扫地机 IP 地址 [192.168.1.91]"
        $promptUser = "  👉 SSH 用户名 [root]"
        $targetLabel = "目标主机："
        $step2 = "[步骤 2/4] Telegram 参数设置："
        $promptToken = "  👉 机器人 Token (@BotFather)"
        $errToken = "     Token 不能为空！请通过 @BotFather 获取。"
        $promptChat = "  👉 您的 Telegram 数字 ID (@userinfobot)"
        $errChat = "     ID 不能为空！可通过 @userinfobot 查询。"
        $step3 = "[步骤 3/4] 附加配置："
        $promptLang = "  👉 机器人界面语言 (ru/en/de/zh) [zh]"
        $promptAuto = "  👉 是否将机器人加入扫地机开机自启？[Y/n]"
        $step4 = "[步骤 4/4] 下载与部署："
        $msgDownBin = "  正在下载 tgbot 二进制文件 ($TargetVer)..."
        $msgDownRun = "  正在下载守护脚本 run.sh..."
        $msgSSH = "  正在检查扫地机 SSH 连接..."
        $errSSH = "  错误：无法通过 SSH 连接到扫地机！"
        $msgSaving = "  正在创建 /data/tgbot 并保存 .env..."
        $msgUpload = "  正在通过 SCP 上传文件到扫地机..."
        $msgAuto = "  正在配置 /data/_root.sh 开机自启..."
        $msgStarting = "  正在启动守护脚本 run.sh..."
        $finishTitle = "   🎉 恭喜！机器人已成功部署并运行！             "
        $finish1 = "1. 在 Telegram 中打开您的机器人并发送命令：/start"
        $finish2 = "2. 实时查看运行日志命令："
        $defaultBotLang = "zh"
    }
    default {
        $title = "   🤖 Valetudo Telegram Bot — Мастер установки (Windows) "
        $intro = "Этот мастер скачает, настроит и установит бота прямо на вашего робота.`nДля выбора значений по умолчанию в скобках просто нажмите [Enter].`n"
        $step1 = "[Шаг 1/4] Параметры подключения к роботу:"
        $promptIP = "  👉 IP-адрес робота [192.168.1.91]"
        $promptUser = "  👉 SSH-пользователь [root]"
        $targetLabel = "Цель:"
        $step2 = "[Шаг 2/4] Telegram параметры:"
        $promptToken = "  👉 Токен бота (@BotFather)"
        $errToken = "     Токен не может быть пустым! Получите его у @BotFather."
        $promptChat = "  👉 Ваш числовой Telegram ID (@userinfobot)"
        $errChat = "     ID не может быть пустым! Узнайте его у @userinfobot."
        $step3 = "[Шаг 3/4] Дополнительные настройки:"
        $promptLang = "  👉 Язык интерфейса бота (ru/en/de/zh) [ru]"
        $promptAuto = "  👉 Добавить бота в автозагрузку робота при старте? [Y/n]"
        $step4 = "[Шаг 4/4] Загрузка и развертывание:"
        $msgDownBin = "  Загрузка бинарника tgbot ($TargetVer)..."
        $msgDownRun = "  Загрузка супервизора run.sh..."
        $msgSSH = "  Проверка доступности робота по SSH..."
        $errSSH = "  Ошибка: Не удалось подключиться к роботу по SSH!"
        $msgSaving = "  Создание каталога /data/tgbot и запись .env на роботе..."
        $msgUpload = "  Загрузка файлов на робота через SCP..."
        $msgAuto = "  Настройка автозагрузки в /data/_root.sh..."
        $msgStarting = "  Запуск супервизора run.sh..."
        $finishTitle = "   🎉 Поздравляем! Бот успешно установлен и запущен!       "
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
# Шаг 4: Загрузка файлов и развертывание
# ------------------------------------------------------------------------------
Write-Host $step4 -ForegroundColor Yellow

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "tgbot_installer_$([System.Guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
$localBin = Join-Path $tempDir "tgbot"
$localRun = Join-Path $tempDir "run.sh"

try {
    Write-Host $msgDownBin -ForegroundColor Cyan
    if ($TargetVer -eq "latest") {
        $binUrl = "https://github.com/$Repo/releases/latest/download/tgbot"
    } else {
        $binUrl = "https://github.com/$Repo/releases/download/$TargetVer/tgbot"
    }

    try {
        Invoke-WebRequest -Uri $binUrl -OutFile $localBin -UseBasicParsing
    } catch {
        if ($TargetVer -ne "latest" -and -not $TargetVer.StartsWith("v")) {
            $binUrl = "https://github.com/$Repo/releases/download/v$TargetVer/tgbot"
            Invoke-WebRequest -Uri $binUrl -OutFile $localBin -UseBasicParsing
        } else {
            throw $_
        }
    }
    Write-Host "  ✓ tgbot загружен успешно." -ForegroundColor Green

    Write-Host $msgDownRun -ForegroundColor Cyan
    $runUrl = "https://raw.githubusercontent.com/$Repo/main/run.sh"
    Invoke-WebRequest -Uri $runUrl -OutFile $localRun -UseBasicParsing
    Write-Host "  ✓ run.sh загружен успешно." -ForegroundColor Green

    Write-Host $msgSSH -ForegroundColor Cyan
    ssh -o ConnectTimeout=5 $Target "echo ssh_ok" | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Host $errSSH -ForegroundColor Red
        exit 1
    }
    Write-Host "  ✓ Подключение по SSH успешно." -ForegroundColor Green

    Write-Host $msgSaving -ForegroundColor Cyan
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
    Write-Host "  ✓ Конфигурация сохранена в /data/tgbot/.env" -ForegroundColor Green

    # Остановка старых процессов
    ssh -o ConnectTimeout=5 $Target "killall run.sh tgbot 2>/dev/null || true; sleep 1; killall -9 run.sh tgbot 2>/dev/null || true; rm -f /var/run/tgbot_run.pid /tmp/tgbot.alive"

    Write-Host $msgUpload -ForegroundColor Cyan
    scp -o ConnectTimeout=10 "$localBin" "$localRun" "${Target}:/data/tgbot/"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  Ошибка при передаче файлов по SCP!" -ForegroundColor Red
        exit 1
    }
    Write-Host "  ✓ Файлы успешно переданы на робота." -ForegroundColor Green

    if ($Autostart -match "^[Yy]") {
        Write-Host $msgAuto -ForegroundColor Cyan
        ssh -o ConnectTimeout=5 $Target "grep -q '/data/tgbot/run.sh' /data/_root.sh 2>/dev/null || echo '/data/tgbot/run.sh &' >> /data/_root.sh; chmod +x /data/_root.sh 2>/dev/null || true"
        Write-Host "  ✓ Автозагрузка настроена." -ForegroundColor Green
    }

    Write-Host $msgStarting -ForegroundColor Cyan
    ssh -o ConnectTimeout=5 $Target "chmod +x /data/tgbot/tgbot /data/tgbot/run.sh; nohup /data/tgbot/run.sh >/dev/null 2>&1 &"

    Start-Sleep -Seconds 2
} finally {
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}

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
