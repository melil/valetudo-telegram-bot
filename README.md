# 🤖 Valetudo Telegram Bot (Dreame X30 Pro)

Легковесный, автономный Telegram-бот на Go для управления роботом-пылесосом Dreame X30 Pro (и другими моделями) через локальный REST API [Valetudo v2](https://valetudo.cloud/).

Бот написан **исключительно на стандартной библиотеке Go** (без сторонних Telegram SDK и тяжелых зависимостей) и компилируется в **один компактный статический бинарник (~6 МБ)**, идеально подходящий для запуска прямо на встроенной ОС робота или локальном роутере/сервере.

---

## 🚀 Возможности

- 🪄 **Мастер уборки (Wizard)**: Интерактивный 3-шаговый выбор режима (сухая, влажная, комбо, сначала сухая затем влажная), выбор комнат по чекбоксам и количества проходов (1x, 2x).
- 🏠 **Управление док-станцией**:
  - Выгрузка пыли в мешок станции.
  - Стирка швабр.
  - Старт и принудительная остановка сушки горячим воздухом.
  - Возврат на базу.
- 🧹 **Мониторинг расходников**: Текстовые прогресс-бары остаточного ресурса щеток, HEPA-фильтра и сенсоров с кнопками быстрого сброса после замены.
- 🏎 **Бортовой журнал и телеметрия**: Детальный статус силовой установки, заряд АКБ, аптайм Linux-системы робота, статус баков чистой/грязной воды, картриджа моющего средства и статистика уборок.
- 📡 **Автономный фоновый вотчер (`statusWatcher`)**:
  - Моментальные алерты при ошибках робота с кодами ошибок.
  - Пуш-уведомления, если закончилась чистая вода или переполнен бак грязной воды.
  - Автоматическая отправка PNG-карты и статистики завершённой сессии (время, площадь) по возвращении на станцию.
- 🌙 **Режим DND (Не беспокоить)**: Автоматическое отключение звуковых уведомлений Telegram в ночные часы.
- 🛡 **Надёжность**: Защита от зависаний HTTP (таймауты), отсутствие утечек сокетов и памяти, поддержка Graceful Shutdown (`SIGINT`, `SIGTERM`).

---

## 📁 Архитектура проекта

Проект организован по стандартному Go-лейауту:

```text
.
├── cmd/
│   └── bot/
│       └── main.go          # Точка входа, инициализация и Graceful Shutdown
├── internal/
│   ├── config/
│   │   └── config.go        # Парсинг ENV, DND-логика, маппинг комнат
│   ├── telegram/
│   │   ├── client.go        # HTTP-клиент Telegram API (сообщения, фото, инлайн-кнопки)
│   │   └── models.go        # DTO модели Telegram API
│   ├── valetudo/
│   │   ├── client.go        # HTTP-клиент Valetudo REST API (действия, пресеты, карта)
│   │   └── models.go        # DTO модели Valetudo API
│   └── bot/
│       ├── bot.go           # Контроллер Bot, жизненный цикл и запуск
│       ├── handler.go       # Роутер текстовых команд и обработка CallbackQuery
│       ├── menus.go         # Инлайн-клавиатуры меню (Робот, Станция, Настройки)
│       ├── wizard.go        # Сессии и шаги интерактивного визарда уборки
│       ├── telemetry.go     # Сборка отчёта бортового журнала, прогресс-бары, аптайм
│       └── watcher.go       # Фоновый воркер телеметрии и пуш-алертов
├── go.mod
├── Dockerfile               # Многоэтапная сборка контейнера
└── README.md
```

---

## ⚙️ Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|---|---|---|---|
| `BOT_TOKEN` | **Да** | — | Токен Telegram-бота от `@BotFather` |
| `CHAT_ID` | **Да** | — | ID чата владельца (доступ разрешён только этому ID) |
| `VALETUDO_BASE_URL` | Нет | `http://127.0.0.1/api/v2/robot` | Базовый URL API Valetudo |
| `TG_API_BASE` | Нет | `https://api.telegram.org` | Базовый URL Telegram API (или локальный bot api сервер) |
| `DND_ENABLED` | Нет | `true` | Включен ли ночной тихий режим |
| `DND_START_HOUR` | Нет | `23` | Час начала тихого режима (0–23) |
| `DND_END_HOUR` | Нет | `8` | Час окончания тихого режима (0–23) |

---

## 🛠 Сборка

### 1. Кросс-компиляция в один бинарник для робота (Linux ARM64)

На машине разработчика (Windows PowerShell):
```powershell
$env:GOOS="linux"; $env:GOARCH="arm64"; $env:CGO_ENABLED="0"
go build -ldflags="-s -w" -o tgbot ./cmd/bot
```

Или в один клик с помощью скрипта деплоя:
```powershell
.\deploy.ps1
```
*(Скрипт сам соберёт свежий бинарник, остановит старый процесс по SSH, загрузит по SCP и запустит `nohup /data/tgbot/run.sh`)*

На Linux / macOS (Bash):
```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o tgbot ./cmd/bot
```

> **Флаги `-ldflags="-s -w"`** вырезают таблицу символов и DWARF-отладку, сокращая размер файла до **~6 МБ**.

### 2. Сборка для локального тестирования (Windows / macOS / Linux)

```bash
go build -o tgbot ./cmd/bot
```

### 3. Сборка через Docker

```bash
docker build -t valetudo-tgbot .
```

---

## 📦 Запуск на роботе

1. Скопируйте собранный бинарник `tgbot` на робота по SSH/SCP (например, в `/data/tgbot`):
   ```bash
   scp tgbot root@<ip-робота>:/data/tgbot
   ```
2. Сделайте файл исполняемым:
   ```bash
   ssh root@<ip-робота> "chmod +x /data/tgbot"
   ```
3. Создайте service-юнит (для `systemd` или `procd` OpenWrt). Пример для `systemd` (`/etc/systemd/system/tgbot.service`):
   ```ini
   [Unit]
   Description=Valetudo Telegram Bot
   After=network.target valetudo.service

   [Service]
   Type=simple
   Environment=BOT_TOKEN=123456789:AA...
   Environment=CHAT_ID=123456789
   Environment=VALETUDO_BASE_URL=http://127.0.0.1/api/v2/robot
   ExecStart=/data/tgbot
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```
4. Запустите сервис:
   ```bash
   systemctl daemon-reload
   systemctl enable --now tgbot
   ```
