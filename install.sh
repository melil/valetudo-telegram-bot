#!/bin/sh
# ==============================================================================
#  🤖 Valetudo Telegram Bot — Интерактивный онлайн-установщик (Installer)
#  Использование:
#    sh -c "$(curl -fsSLk https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh 2>/dev/null || wget -q --no-check-certificate -O- https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh 2>/dev/null || curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh || wget -qO- https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh)"
#    curl -fsSLk https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh -o /tmp/install.sh && sh /tmp/install.sh 1.0.18
# ==============================================================================

set -e

REPO="melil/valetudo-telegram-bot"
TARGET_VER="${1:-latest}"
INSTALL_DIR="/data/tgbot"

# Цвета для терминала
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

# Очистка экрана если доступен tty
if [ -t 1 ]; then
    clear 2>/dev/null || true
fi

# ------------------------------------------------------------------------------
# Проверка запуска на ПК vs Робот
# ------------------------------------------------------------------------------
if [ ! -d "/data" ] && [ "$(uname -s)" != "Linux" ]; then
    echo -e "${YELLOW}Похоже, скрипт запущен на ПК/ноутбуке, а не прямо на роботе.${NC}"
    printf "%s" "Подключиться к роботу по SSH и запустить установку там? [Y/n]: "
    read -r RUN_SSH </dev/tty
    RUN_SSH=${RUN_SSH:-Y}
    case "$RUN_SSH" in
        [Nn]*) ;;
        *)
            printf "%s" "Введите IP-адрес робота [192.168.1.91]: "
            read -r R_IP </dev/tty
            R_IP=${R_IP:-192.168.1.91}
            printf "%s" "SSH-пользователь [root]: "
            read -r R_USER </dev/tty
            R_USER=${R_USER:-root}
            echo -e "${CYAN}Подключение к ${R_USER}@${R_IP} и запуск установки...${NC}"
            ssh -t "${R_USER}@${R_IP}" "curl -fsSLk https://raw.githubusercontent.com/${REPO}/main/install.sh -o /tmp/install.sh 2>/dev/null || wget -q --no-check-certificate -O /tmp/install.sh https://raw.githubusercontent.com/${REPO}/main/install.sh 2>/dev/null || curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh -o /tmp/install.sh 2>/dev/null || wget -qO /tmp/install.sh https://raw.githubusercontent.com/${REPO}/main/install.sh; sh /tmp/install.sh ${TARGET_VER}; rm -f /tmp/install.sh"
            exit 0
            ;;
    esac
fi

# Проверка системного времени (на роботах после перезагрузки без NTP время может быть сброшено на 1970 год, что ломает TLS/HTTPS)
if [ "$(date +%Y)" -lt 2024 ]; then
    echo -e "${YELLOW}Предупреждение: системное время на устройстве не синхронизировано ($(date)).${NC}"
    echo -e "${YELLOW}Попытка синхронизации времени по NTP...${NC}"
    ntpd -q -p pool.ntp.org 2>/dev/null || ntpd -q -p time.google.com 2>/dev/null || true
fi

# ------------------------------------------------------------------------------
# Шаг 0: Выбор языка мастера (1: ru, 2: en, 3: de, 4: zh)
# ------------------------------------------------------------------------------
echo -e "${CYAN}${BOLD}============================================================${NC}"
echo -e "${CYAN}${BOLD}   Select language / Выберите язык мастера / 语言选择       ${NC}"
echo -e "${CYAN}${BOLD}============================================================${NC}"
echo "  1) Русский (Russian) [По умолчанию]"
echo "  2) English"
echo "  3) Deutsch (German)"
echo "  4) 简体中文 (Chinese)"
echo ""
printf "%s" "👉 Выбор / Choice [1-4, Default: 1]: "
read -r LANG_CHOICE </dev/tty

case "$LANG_CHOICE" in
    2) LANG_CODE="en" ;;
    3) LANG_CODE="de" ;;
    4) LANG_CODE="zh" ;;
    *) LANG_CODE="ru" ;;
esac

# Локализованные тексты
case "$LANG_CODE" in
    en)
        T_TITLE="   🤖 Valetudo Telegram Bot — Quick Installer        "
        T_INTRO="This installer will download, configure and launch the bot.\nPress [Enter] to accept default values.\n"
        T_STEP1="[Step 1/3] Telegram Settings:"
        T_PROMPT_TOKEN="  👉 Bot Token (@BotFather): "
        T_ERR_TOKEN="Token cannot be empty! Obtain it from @BotFather."
        T_PROMPT_CHAT="  👉 Your numeric Telegram ID (@userinfobot): "
        T_ERR_CHAT="ID cannot be empty! Obtain it from @userinfobot."
        T_STEP2="[Step 2/3] Additional Settings:"
        T_PROMPT_LANG="  👉 Bot interface language (ru/en/de/zh) [en]: "
        T_PROMPT_AUTO="  👉 Add bot to robot autostart on boot? [Y/n]: "
        T_STEP3="[Step 3/3] Downloading and installing:"
        T_DOWNLOADING_BIN="Downloading tgbot binary (${TARGET_VER})..."
        T_DOWNLOADING_RUN="Downloading supervisor script run.sh..."
        T_ERR_DOWNLOAD="Download failed! Please check your internet connection or tag name."
        T_SAVING_ENV="Creating ${INSTALL_DIR} and saving .env..."
        T_CONFIG_AUTO="Configuring autostart in /data/_root.sh..."
        T_STARTING="Starting supervisor run.sh..."
        T_FINISH_TITLE="   🎉 Congratulations! Bot is installed and running! "
        T_FINISH_1="1. Open your bot in Telegram and send: /start"
        T_FINISH_2="2. To view live logs in real time, run:"
        DEFAULT_BOT_LANG="en"
        ;;
    de)
        T_TITLE="   🤖 Valetudo Telegram Bot — Schnell-Installer     "
        T_INTRO="Dieser Installer lädt den Bot herunter, konfiguriert und startet ihn.\nDrücken Sie [Enter], um Standardwerte zu übernehmen.\n"
        T_STEP1="[Schritt 1/3] Telegram-Einstellungen:"
        T_PROMPT_TOKEN="  👉 Bot-Token (@BotFather): "
        T_ERR_TOKEN="Token darf nicht leer sein! Holen Sie es sich bei @BotFather."
        T_PROMPT_CHAT="  👉 Ihre numerische Telegram-ID (@userinfobot): "
        T_ERR_CHAT="ID darf nicht leer sein! Finden Sie sie über @userinfobot."
        T_STEP2="[Schritt 2/3] Zusätzliche Einstellungen:"
        T_PROMPT_LANG="  👉 Sprache der Bot-Oberfläche (ru/en/de/zh) [de]: "
        T_PROMPT_AUTO="  👉 Bot zum Roboter-Autostart hinzufügen? [Y/n]: "
        T_STEP3="[Schritt 3/3] Herunterladen und Installieren:"
        T_DOWNLOADING_BIN="Lade tgbot-Binärdatei herunter (${TARGET_VER})..."
        T_DOWNLOADING_RUN="Lade Supervisor run.sh herunter..."
        T_ERR_DOWNLOAD="Download fehlgeschlagen! Bitte Verbindung oder Tag prüfen."
        T_SAVING_ENV="Erstelle ${INSTALL_DIR} und speichere .env..."
        T_CONFIG_AUTO="Richte Autostart in /data/_root.sh ein..."
        T_STARTING="Starte Supervisor run.sh..."
        T_FINISH_TITLE="   🎉 Glückwunsch! Bot ist installiert und läuft!   "
        T_FINISH_1="1. Öffnen Sie Ihren Bot in Telegram und senden Sie: /start"
        T_FINISH_2="2. Um Live-Logs in Echtzeit anzuzeigen, führen Sie aus:"
        DEFAULT_BOT_LANG="de"
        ;;
    zh)
        T_TITLE="   🤖 Valetudo Telegram Bot — 一键在线安装向导      "
        T_INTRO="此安装向导将自动下载、配置并启动机器人。\n直接按 [Enter] 即可使用默认值。\n"
        T_STEP1="[步骤 1/3] Telegram 参数设置："
        T_PROMPT_TOKEN="  👉 机器人 Token (@BotFather): "
        T_ERR_TOKEN="Token 不能为空！请通过 @BotFather 获取。"
        T_PROMPT_CHAT="  👉 您的 Telegram 数字 ID (@userinfobot): "
        T_ERR_CHAT="ID 不能为空！可通过 @userinfobot 查询。"
        T_STEP2="[步骤 2/3] 附加配置："
        T_PROMPT_LANG="  👉 机器人界面语言 (ru/en/de/zh) [zh]: "
        T_PROMPT_AUTO="  👉 是否加入扫地机开机自启？[Y/n]: "
        T_STEP3="[步骤 3/3] 正在下载并安装："
        T_DOWNLOADING_BIN="正在下载 tgbot 二进制文件 (${TARGET_VER})..."
        T_DOWNLOADING_RUN="正在下载守护脚本 run.sh..."
        T_ERR_DOWNLOAD="下载失败！请检查网络连接或版本标签。"
        T_SAVING_ENV="创建 ${INSTALL_DIR} 目录并写入 .env..."
        T_CONFIG_AUTO="配置 /data/_root.sh 开机自启..."
        T_STARTING="启动守护脚本 run.sh..."
        T_FINISH_TITLE="   🎉 恭喜！机器人已成功安装并运行！             "
        T_FINISH_1="1. 在 Telegram 中打开您的机器人并发送命令：/start"
        T_FINISH_2="2. 实时查看运行日志命令："
        DEFAULT_BOT_LANG="zh"
        ;;
    *)
        T_TITLE="   🤖 Valetudo Telegram Bot — Мастер быстрой установки     "
        T_INTRO="Этот мастер скачает, настроит и запустит бота прямо на вашем роботе.\nДля выбора значений в скобках просто нажмите [Enter].\n"
        T_STEP1="[Шаг 1/3] Параметры Telegram:"
        T_PROMPT_TOKEN="  👉 Токен бота (@BotFather): "
        T_ERR_TOKEN="Токен не может быть пустым! Получите его у @BotFather."
        T_PROMPT_CHAT="  👉 Ваш числовой Telegram ID (@userinfobot): "
        T_ERR_CHAT="ID не может быть пустым! Узнайте его у @userinfobot."
        T_STEP2="[Шаг 2/3] Дополнительные настройки:"
        T_PROMPT_LANG="  👉 Язык интерфейса бота (ru/en/de/zh) [ru]: "
        T_PROMPT_AUTO="  👉 Добавить бота в автозагрузку робота при включении? [Y/n]: "
        T_STEP3="[Шаг 3/3] Загрузка и развертывание:"
        T_DOWNLOADING_BIN="Загрузка бинарника tgbot (${TARGET_VER})..."
        T_DOWNLOADING_RUN="Загрузка супервизора run.sh..."
        T_ERR_DOWNLOAD="Ошибка загрузки! Проверьте интернет или имя релиза."
        T_SAVING_ENV="Создание каталога ${INSTALL_DIR} и сохранение .env..."
        T_CONFIG_AUTO="Настройка автозагрузки в /data/_root.sh..."
        T_STARTING="Запуск супервизора run.sh..."
        T_FINISH_TITLE="   🎉 Поздравляем! Бот успешно установлен и запущен!       "
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
# Шаг 1: Telegram настройки
# ------------------------------------------------------------------------------
echo -e "${YELLOW}${BOLD}${T_STEP1}${NC}"
while true; do
    printf "%s" "$T_PROMPT_TOKEN"
    read -r BOT_TOKEN </dev/tty
    if [ -n "$BOT_TOKEN" ]; then
        break
    fi
    echo -e "     ${RED}${T_ERR_TOKEN}${NC}"
done

while true; do
    printf "%s" "$T_PROMPT_CHAT"
    read -r CHAT_ID </dev/tty
    if [ -n "$CHAT_ID" ]; then
        break
    fi
    echo -e "     ${RED}${T_ERR_CHAT}${NC}"
done
echo ""

# ------------------------------------------------------------------------------
# Шаг 2: Дополнительные опции
# ------------------------------------------------------------------------------
echo -e "${YELLOW}${BOLD}${T_STEP2}${NC}"
printf "%s" "$T_PROMPT_LANG"
read -r BOT_LANG </dev/tty
BOT_LANG=${BOT_LANG:-$DEFAULT_BOT_LANG}

printf "%s" "$T_PROMPT_AUTO"
read -r AUTOSTART </dev/tty
AUTOSTART=${AUTOSTART:-Y}
echo ""

# ------------------------------------------------------------------------------
# Шаг 3: Скачивание файлов и установка
# ------------------------------------------------------------------------------
echo -e "${YELLOW}${BOLD}${T_STEP3}${NC}"

mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

download() {
    local url="$1"
    local dest="$2"

    # 1. Попытка curl
    if command -v curl >/dev/null 2>&1; then
        # Сначала пробуем со стандартной валидацией TLS
        if curl -fsSL "$url" -o "$dest" 2>/dev/null; then
            return 0
        fi
        # На прошивках роботов (OpenWrt/Tina Linux) часто отсутствуют CA-сертификаты (ошибка curl 60).
        # Пробуем curl с флагом -k (--insecure)
        if curl -fsSLk "$url" -o "$dest" 2>/dev/null; then
            return 0
        fi
    fi

    # 2. Попытка wget (если curl не установлен или не смог завершить загрузку)
    if command -v wget >/dev/null 2>&1; then
        if wget -qO "$dest" "$url" 2>/dev/null; then
            return 0
        fi
        if wget -q --no-check-certificate -O "$dest" "$url" 2>/dev/null; then
            return 0
        fi
    fi

    # 3. Финальная попытка с открытым выводом ошибок в консоль для диагностики
    if command -v curl >/dev/null 2>&1; then
        curl -fSLk "$url" -o "$dest"
    elif command -v wget >/dev/null 2>&1; then
        wget --no-check-certificate -O "$dest" "$url"
    else
        echo -e "${RED}Ошибка: в системе не найден ни curl, ни wget!${NC}" >&2
        return 1
    fi
}

echo -e "  ${T_DOWNLOADING_BIN}"

# Определение URL релиза
if [ "$TARGET_VER" = "latest" ]; then
    BIN_URL="https://github.com/${REPO}/releases/latest/download/tgbot"
else
    # Пробуем указанную версию напрямую или с префиксом v
    BIN_URL="https://github.com/${REPO}/releases/download/${TARGET_VER}/tgbot"
fi

if ! download "$BIN_URL" "tgbot"; then
    # Попытка с префиксом v если без него не вышло
    case "$TARGET_VER" in
        latest|v*) ;;
        *)
            BIN_URL="https://github.com/${REPO}/releases/download/v${TARGET_VER}/tgbot"
            download "$BIN_URL" "tgbot" || true
            ;;
    esac
fi

if [ ! -s "tgbot" ]; then
    echo -e "  ${RED}${T_ERR_DOWNLOAD}${NC}"
    echo -e "  URL: $BIN_URL"
    exit 1
fi
echo -e "  ${GREEN}✓ tgbot загружен успешно.${NC}"

echo -e "  ${T_DOWNLOADING_RUN}"
RUN_URL="https://raw.githubusercontent.com/${REPO}/main/run.sh"
download "$RUN_URL" "run.sh"
if [ ! -s "run.sh" ]; then
    echo -e "  ${RED}Ошибка загрузки run.sh!${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓ run.sh загружен успешно.${NC}"

echo -e "  ${T_SAVING_ENV}"
cat << EOF > "$INSTALL_DIR/.env"
BOT_TOKEN=${BOT_TOKEN}
CHAT_ID=${CHAT_ID}
BOT_LANG=${BOT_LANG}
VALETUDO_BASE_URL=http://127.0.0.1/api/v2/robot
DB_PATH=/data/tgbot/bot.db
DND_ENABLED=true
EOF
echo -e "  ${GREEN}✓ .env сохранён.${NC}"

# Остановка старых процессов
echo -e "  Перезапуск службы бота..."
killall run.sh tgbot 2>/dev/null || true
sleep 1
killall -9 run.sh tgbot 2>/dev/null || true
rm -f /var/run/tgbot_run.pid /tmp/tgbot.alive

# Права
chmod +x "$INSTALL_DIR/tgbot" "$INSTALL_DIR/run.sh"

# Автозагрузка
case "$AUTOSTART" in
    [Nn]*) ;;
    *)
        echo -e "  ${T_CONFIG_AUTO}"
        grep -q "$INSTALL_DIR/run.sh" /data/_root.sh 2>/dev/null || echo "$INSTALL_DIR/run.sh &" >> /data/_root.sh
        chmod +x /data/_root.sh 2>/dev/null || true
        echo -e "  ${GREEN}✓ Автозагрузка настроена.${NC}"
        ;;
esac

# Запуск
echo -e "  ${T_STARTING}"
nohup "$INSTALL_DIR/run.sh" >/dev/null 2>&1 &
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
echo -e "   ${CYAN}tail -f /tmp/log/custom/tgbot.log${NC}"
echo ""
