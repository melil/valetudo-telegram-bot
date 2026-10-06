#!/usr/bin/env bash
# ==============================================================================
#  🤖 Valetudo Telegram Bot — Интерактивный мастер установки (Setup Wizard)
# ==============================================================================

set -e

# Цвета для терминала
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

clear 2>/dev/null || true

# ------------------------------------------------------------------------------
# Выбор локализации мастера (1: ru, 2: en, 3: de, 4: zh)
# ------------------------------------------------------------------------------
echo -e "${CYAN}${BOLD}============================================================${NC}"
echo -e "${CYAN}${BOLD}   Select language / Выберите язык мастера / 语言选择       ${NC}"
echo -e "${CYAN}${BOLD}============================================================${NC}"
echo "  1) Русский (Russian) [По умолчанию]"
echo "  2) English"
echo "  3) Deutsch (German)"
echo "  4) 简体中文 (Chinese)"
echo ""
read -rp "👉 Выбор / Choice [1-4, Default: 1]: " LANG_CHOICE

case "$LANG_CHOICE" in
    2) LANG_CODE="en" ;;
    3) LANG_CODE="de" ;;
    4) LANG_CODE="zh" ;;
    *) LANG_CODE="ru" ;;
esac

# ------------------------------------------------------------------------------
# Локализованные строки
# ------------------------------------------------------------------------------
case "$LANG_CODE" in
    en)
        T_TITLE="   🤖 Valetudo Telegram Bot — Quick Setup Wizard     "
        T_INTRO="This wizard will configure, build, and deploy the bot to your robot.\nPress [Enter] to accept default values shown in brackets.\n"
        T_STEP1="[Step 1/5] Robot connection parameters:"
        T_PROMPT_IP="  👉 Robot IP address [192.168.1.91]: "
        T_PROMPT_USER="  👉 SSH user [root]: "
        T_TARGET="Target:"
        T_STEP2="[Step 2/5] Telegram settings:"
        T_PROMPT_TOKEN="  👉 Bot Token (@BotFather): "
        T_ERR_TOKEN="Token cannot be empty! Obtain it from @BotFather."
        T_PROMPT_CHAT="  👉 Your numeric Telegram ID (@userinfobot): "
        T_ERR_CHAT="ID cannot be empty! Obtain it from @userinfobot."
        T_STEP3="[Step 3/5] Additional options:"
        T_PROMPT_LANG="  👉 Bot interface language (ru/en/de/zh) [en]: "
        T_PROMPT_AUTO="  👉 Add bot to robot autostart on boot? [Y/n]: "
        T_STEP4="[Step 4/5] Environment check & binary build:"
        T_ERR_NOGO="Error: Go compiler not found! Install Go 1.22+ from https://go.dev/dl/"
        T_FOUND_GO="Found Go version:"
        T_BUILDING="Compiling for Linux ARM64 (CGO_ENABLED=0)..."
        T_ERR_BUILD="Error: build failed, 'tgbot' binary not found!"
        T_BUILD_OK="✓ Binary built successfully"
        T_STEP5="[Step 5/5] Deploying to robot"
        T_CHECK_SSH="Checking robot SSH accessibility..."
        T_ERR_SSH="Error: Cannot connect to robot via SSH!"
        T_SSH_HINT="Make sure the robot is powered on, connected to Wi-Fi, and SSH is enabled."
        T_SSH_OK="✓ SSH connection successful."
        T_SAVING_ENV="Creating /data/tgbot directory and saving .env configuration..."
        T_ENV_OK="✓ Configuration saved to /data/tgbot/.env"
        T_STOP_OLD="Stopping old bot processes (if running)..."
        T_UPLOADING="Uploading files to robot via SCP..."
        T_UPLOAD_OK="✓ Files uploaded to /data/tgbot/"
        T_CONFIG_AUTO="Configuring autostart in /data/_root.sh..."
        T_AUTO_OK="✓ Autostart configured."
        T_STARTING="Starting supervisor run.sh..."
        T_FINISH_TITLE="   🎉 Congratulations! Bot deployed and running!   "
        T_FINISH_1="1. Open your bot in Telegram and send the command: /start"
        T_FINISH_2="2. To view live logs in real time, run:"
        DEFAULT_BOT_LANG="en"
        ;;
    de)
        T_TITLE="   🤖 Valetudo Telegram Bot — Schnelleinrichtungsassistent  "
        T_INTRO="Dieser Assistent konfiguriert, baut und startet den Bot auf Ihrem Roboter.\nDrücken Sie [Enter], um die Standardwerte in Klammern zu übernehmen.\n"
        T_STEP1="[Schritt 1/5] Roboter-Verbindungsparameter:"
        T_PROMPT_IP="  👉 Roboter-IP-Adresse [192.168.1.91]: "
        T_PROMPT_USER="  👉 SSH-Benutzer [root]: "
        T_TARGET="Ziel:"
        T_STEP2="[Schritt 2/5] Telegram-Einstellungen:"
        T_PROMPT_TOKEN="  👉 Bot-Token (@BotFather): "
        T_ERR_TOKEN="Token darf nicht leer sein! Holen Sie es sich bei @BotFather."
        T_PROMPT_CHAT="  👉 Ihre numerische Telegram-ID (@userinfobot): "
        T_ERR_CHAT="ID darf nicht leer sein! Finden Sie sie über @userinfobot."
        T_STEP3="[Schritt 3/5] Zusätzliche Optionen:"
        T_PROMPT_LANG="  👉 Sprache der Bot-Oberfläche (ru/en/de/zh) [de]: "
        T_PROMPT_AUTO="  👉 Bot zum Roboter-Autostart hinzufügen? [Y/n]: "
        T_STEP4="[Schritt 4/5] Umgebungsprüfung und Binärbau:"
        T_ERR_NOGO="Fehler: Go-Compiler nicht gefunden! Installieren Sie Go 1.22+ von https://go.dev/dl/"
        T_FOUND_GO="Gefundene Go-Version:"
        T_BUILDING="Kompilierung für Linux ARM64 (CGO_ENABLED=0)..."
        T_ERR_BUILD="Fehler: Build fehlgeschlagen, Binärdatei 'tgbot' nicht gefunden!"
        T_BUILD_OK="✓ Binärdatei erfolgreich gebaut"
        T_STEP5="[Schritt 5/5] Bereitstellung auf dem Roboter"
        T_CHECK_SSH="Prüfe Roboter-SSH-Verbindung..."
        T_ERR_SSH="Fehler: Verbindung zum Roboter über SSH fehlgeschlagen!"
        T_SSH_HINT="Stellen Sie sicher, dass der Roboter eingeschaltet und SSH aktiviert ist."
        T_SSH_OK="✓ SSH-Verbindung erfolgreich."
        T_SAVING_ENV="Erstelle Verzeichnis /data/tgbot und speichere .env..."
        T_ENV_OK="✓ Konfiguration in /data/tgbot/.env gespeichert"
        T_STOP_OLD="Beende alte Bot-Prozesse (falls vorhanden)..."
        T_UPLOADING="Lade Dateien via SCP auf den Roboter..."
        T_UPLOAD_OK="✓ Dateien nach /data/tgbot/ übertragen"
        T_CONFIG_AUTO="Richte Autostart in /data/_root.sh ein..."
        T_AUTO_OK="✓ Autostart eingerichtet."
        T_STARTING="Starte Supervisor run.sh..."
        T_FINISH_TITLE="   🎉 Glückwunsch! Bot erfolgreich bereitgestellt! "
        T_FINISH_1="1. Öffnen Sie Ihren Bot in Telegram und senden Sie: /start"
        T_FINISH_2="2. Um Live-Logs in Echtzeit anzuzeigen, führen Sie aus:"
        DEFAULT_BOT_LANG="de"
        ;;
    zh)
        T_TITLE="   🤖 Valetudo Telegram Bot — 快速安装向导          "
        T_INTRO="此向导将为您配置、编译并在扫地机上启动机器人。\n直接按 [Enter] 即可使用括号中的默认值。\n"
        T_STEP1="[步骤 1/5] 扫地机连接参数："
        T_PROMPT_IP="  👉 扫地机 IP 地址 [192.168.1.91]: "
        T_PROMPT_USER="  👉 SSH 用户名 [root]: "
        T_TARGET="目标主机："
        T_STEP2="[步骤 2/5] Telegram 参数设置："
        T_PROMPT_TOKEN="  👉 机器人 Token (@BotFather): "
        T_ERR_TOKEN="Token 不能为空！请通过 @BotFather 获取。"
        T_PROMPT_CHAT="  👉 您的 Telegram 数字 ID (@userinfobot): "
        T_ERR_CHAT="ID 不能为空！可通过 @userinfobot 查询。"
        T_STEP3="[步骤 3/5] 附加配置："
        T_PROMPT_LANG="  👉 机器人界面语言 (ru/en/de/zh) [zh]: "
        T_PROMPT_AUTO="  👉 是否将机器人加入扫地机开机自启？[Y/n]: "
        T_STEP4="[步骤 4/5] 环境检查与编译："
        T_ERR_NOGO="错误：系统中未找到 Go 编译器！请从 https://go.dev/dl/ 安装 Go 1.22+"
        T_FOUND_GO="检测到 Go 版本："
        T_BUILDING="正在编译 Linux ARM64 二进制文件 (CGO_ENABLED=0)..."
        T_ERR_BUILD="错误：编译失败，未找到 'tgbot' 文件！"
        T_BUILD_OK="✓ 二进制文件编译成功"
        T_STEP5="[步骤 5/5] 部署到扫地机"
        T_CHECK_SSH="正在检查扫地机 SSH 连接..."
        T_ERR_SSH="错误：无法通过 SSH 连接到扫地机！"
        T_SSH_HINT="请确保扫地机已开机、联网并且已启用 SSH。"
        T_SSH_OK="✓ SSH 连接成功。"
        T_SAVING_ENV="正在创建 /data/tgbot 目录并写入 .env 配置文件..."
        T_ENV_OK="✓ 配置已保存至 /data/tgbot/.env"
        T_STOP_OLD="正在停止旧的机器人进程（若有）..."
        T_UPLOADING="正在通过 SCP 上传文件到扫地机..."
        T_UPLOAD_OK="✓ 文件已上传至 /data/tgbot/"
        T_CONFIG_AUTO="正在配置 /data/_root.sh 开机自启..."
        T_AUTO_OK="✓ 开机自启已配置。"
        T_STARTING="正在启动守护脚本 run.sh..."
        T_FINISH_TITLE="   🎉 恭喜！机器人已成功部署并运行！             "
        T_FINISH_1="1. 在 Telegram 中打开您的机器人并发送命令：/start"
        T_FINISH_2="2. 实时查看运行日志命令："
        DEFAULT_BOT_LANG="zh"
        ;;
    *)
        T_TITLE="   🤖 Valetudo Telegram Bot — Мастер быстрой установки     "
        T_INTRO="Этот мастер настроит, скомпилирует и запустит бота на вашем роботе.\nДля выбора значений по умолчанию в скобках просто нажмите [Enter].\n"
        T_STEP1="[Шаг 1/5] Параметры подключения к роботу:"
        T_PROMPT_IP="  👉 IP-адрес робота [192.168.1.91]: "
        T_PROMPT_USER="  👉 SSH-пользователь [root]: "
        T_TARGET="Цель:"
        T_STEP2="[Шаг 2/5] Telegram параметры:"
        T_PROMPT_TOKEN="  👉 Токен бота (@BotFather): "
        T_ERR_TOKEN="Токен не может быть пустым! Получите его у @BotFather."
        T_PROMPT_CHAT="  👉 Ваш числовой Telegram ID (@userinfobot): "
        T_ERR_CHAT="ID не может быть пустым! Узнайте его у @userinfobot."
        T_STEP3="[Шаг 3/5] Дополнительные настройки:"
        T_PROMPT_LANG="  👉 Язык интерфейса бота (ru/en/de/zh) [ru]: "
        T_PROMPT_AUTO="  👉 Добавить бота в автозагрузку робота при старте? [Y/n]: "
        T_STEP4="[Шаг 4/5] Подготовка и сборка бинарника:"
        T_ERR_NOGO="Ошибка: компилятор Go не найден в системе! Установите Go 1.22+ с https://go.dev/dl/"
        T_FOUND_GO="Найдена версия Go:"
        T_BUILDING="Компиляция для Linux ARM64 (CGO_ENABLED=0)..."
        T_ERR_BUILD="Ошибка: сборка не удалась, файл 'tgbot' не найден!"
        T_BUILD_OK="✓ Бинарник собран успешно"
        T_STEP5="[Шаг 5/5] Развертывание на робота"
        T_CHECK_SSH="Проверка доступности робота по SSH..."
        T_ERR_SSH="Ошибка: Не удалось подключиться к роботу по SSH!"
        T_SSH_HINT="Убедитесь, что робот включён, доступен в локальной сети и SSH-сервер запущен."
        T_SSH_OK="✓ Подключение по SSH успешно."
        T_SAVING_ENV="Создание каталога /data/tgbot и запись конфигурации .env..."
        T_ENV_OK="✓ Конфигурация сохранена в /data/tgbot/.env"
        T_STOP_OLD="Остановка старых процессов бота (если запущены)..."
        T_UPLOADING="Загрузка файлов на робота через SCP..."
        T_UPLOAD_OK="✓ Файлы успешно загружены в /data/tgbot/"
        T_CONFIG_AUTO="Настройка автозагрузки в /data/_root.sh..."
        T_AUTO_OK="✓ Автозагрузка настроена."
        T_STARTING="Запуск супервизора run.sh..."
        T_FINISH_TITLE="   🎉 Поздравляем! Бот успешно развернут и запущен!        "
        T_FINISH_1="1. Откройте вашего бота в Telegram и отправьте команду: /start"
        T_FINISH_2="2. Для просмотра логов в реальном времени выполните:"
        DEFAULT_BOT_LANG="ru"
        ;;
esac

echo ""
echo -e "${CYAN}${BOLD}============================================================${NC}"
echo -e "${CYAN}${BOLD}${T_TITLE}${NC}"
echo -e "${CYAN}${BOLD}============================================================${NC}"
echo -e "${T_INTRO}"

# ------------------------------------------------------------------------------
# Шаг 1: Опрос параметров
# ------------------------------------------------------------------------------
echo -e "${YELLOW}${BOLD}${T_STEP1}${NC}"
read -rp "$T_PROMPT_IP" ROBOT_IP
ROBOT_IP=${ROBOT_IP:-192.168.1.91}

read -rp "$T_PROMPT_USER" ROBOT_USER
ROBOT_USER=${ROBOT_USER:-root}

TARGET="${ROBOT_USER}@${ROBOT_IP}"
echo -e "  ${T_TARGET} ${GREEN}${TARGET}${NC}\n"

# ------------------------------------------------------------------------------
# Шаг 2: Telegram параметры
# ------------------------------------------------------------------------------
echo -e "${YELLOW}${BOLD}${T_STEP2}${NC}"
while true; do
    read -rp "$T_PROMPT_TOKEN" BOT_TOKEN
    if [ -n "$BOT_TOKEN" ]; then
        break
    fi
    echo -e "     ${RED}${T_ERR_TOKEN}${NC}"
done

while true; do
    read -rp "$T_PROMPT_CHAT" CHAT_ID
    if [ -n "$CHAT_ID" ]; then
        break
    fi
    echo -e "     ${RED}${T_ERR_CHAT}${NC}"
done
echo ""

# ------------------------------------------------------------------------------
# Шаг 3: Дополнительные настройки
# ------------------------------------------------------------------------------
echo -e "${YELLOW}${BOLD}${T_STEP3}${NC}"
read -rp "$T_PROMPT_LANG" BOT_LANG
BOT_LANG=${BOT_LANG:-$DEFAULT_BOT_LANG}

read -rp "$T_PROMPT_AUTO" AUTOSTART
AUTOSTART=${AUTOSTART:-Y}
echo ""

# ------------------------------------------------------------------------------
# Шаг 4: Сборка бинарника
# ------------------------------------------------------------------------------
echo -e "${YELLOW}${BOLD}${T_STEP4}${NC}"

if ! command -v go >/dev/null 2>&1; then
    echo -e "  ${RED}${T_ERR_NOGO}${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "  ${T_FOUND_GO} ${GREEN}${GO_VERSION}${NC}"
echo -e "  ${T_BUILDING}"

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o tgbot ./cmd/bot

if [ ! -f "tgbot" ]; then
    echo -e "  ${RED}${T_ERR_BUILD}${NC}"
    exit 1
fi

SIZE_MB=$(du -m tgbot | cut -f1)
echo -e "  ${GREEN}${T_BUILD_OK} (~${SIZE_MB} MB)${NC}\n"

# ------------------------------------------------------------------------------
# Шаг 5: Развертывание
# ------------------------------------------------------------------------------
echo -e "${YELLOW}${BOLD}${T_STEP5} (${TARGET}):${NC}"

echo -e "  ${T_CHECK_SSH}"
if ! ssh -o ConnectTimeout=5 -o BatchMode=no "$TARGET" "echo 'ssh_ok'" >/dev/null 2>&1; then
    echo -e "  ${RED}${T_ERR_SSH}${NC}"
    echo -e "  ${T_SSH_HINT}"
    exit 1
fi
echo -e "  ${GREEN}${T_SSH_OK}${NC}"

echo -e "  ${T_SAVING_ENV}"
ssh -o ConnectTimeout=5 "$TARGET" "mkdir -p /data/tgbot && cat << 'EOF' > /data/tgbot/.env
BOT_TOKEN=${BOT_TOKEN}
CHAT_ID=${CHAT_ID}
BOT_LANG=${BOT_LANG}
VALETUDO_BASE_URL=http://127.0.0.1/api/v2/robot
DB_PATH=/data/tgbot/bot.db
DND_ENABLED=true
EOF"
echo -e "  ${GREEN}${T_ENV_OK}${NC}"

echo -e "  ${T_STOP_OLD}"
ssh -o ConnectTimeout=5 "$TARGET" "killall run.sh tgbot 2>/dev/null || true; sleep 1; killall -9 run.sh tgbot 2>/dev/null || true; rm -f /var/run/tgbot_run.pid /tmp/tgbot.alive"

echo -e "  ${T_UPLOADING}"
scp -o ConnectTimeout=10 tgbot run.sh "$TARGET:/data/tgbot/"
echo -e "  ${GREEN}${T_UPLOAD_OK}${NC}"

if [[ "$AUTOSTART" =~ ^[Yy]$ ]]; then
    echo -e "  ${T_CONFIG_AUTO}"
    ssh -o ConnectTimeout=5 "$TARGET" "grep -q '/data/tgbot/run.sh' /data/_root.sh 2>/dev/null || echo '/data/tgbot/run.sh &' >> /data/_root.sh; chmod +x /data/_root.sh 2>/dev/null || true"
    echo -e "  ${GREEN}${T_AUTO_OK}${NC}"
fi

echo -e "  ${T_STARTING}"
ssh -o ConnectTimeout=5 "$TARGET" "chmod +x /data/tgbot/tgbot /data/tgbot/run.sh && nohup /data/tgbot/run.sh >/dev/null 2>&1 &"

sleep 2

# ------------------------------------------------------------------------------
# Финал
# ------------------------------------------------------------------------------
echo ""
echo -e "${GREEN}${BOLD}============================================================${NC}"
echo -e "${GREEN}${BOLD}${T_FINISH_TITLE}${NC}"
echo -e "${GREEN}${BOLD}============================================================${NC}"
echo ""
echo -e "${T_FINISH_1}"
echo -e "${T_FINISH_2}"
echo -e "   ${CYAN}ssh ${TARGET} \"tail -f /tmp/log/custom/tgbot.log\"${NC}"
echo ""
