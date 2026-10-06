# 🤖 Valetudo Telegram Bot

**English** | [Русский](README.ru.md)

[![Vibe](https://img.shields.io/badge/vibecoded-with%20love-ff69b4.svg)](https://github.com/melil/valetudo-telegram-bot)
[![Go Version](https://img.shields.io/github/go-mod/v/melil/valetudo-telegram-bot)](https://github.com/melil/valetudo-telegram-bot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A lightweight, standalone Go Telegram bot for controlling vacuum robots running [Valetudo v2](https://valetudo.cloud/) via its local REST API. Tested on Dreame X30 Pro (also supports other Valetudo-compatible Dreame, Roborock, and similar models).

Built entirely with the Go standard library without external runtime dependencies, compiling into a **single compact static binary (~6 MB)**. Highly resource-efficient, averaging just **~12 MB of RAM usage**, making it ideal for running directly on the robot's embedded Linux system (`/data/tgbot`), a home router (OpenWrt), or a server.

---

## 🚀 Features

- 🪄 **Cleaning Wizard & Quick Clean**:
  - One-tap quick start for standard full cleaning directly from the main dashboard (`/clean`).
  - Interactive step-by-step room cleaning wizard (`/wizard`): mode selection (vacuum, mop, combo, vacuum then mop).
  - Dynamic room discovery from the current map with custom aliases support (`ROOM_ALIASES`).
  - Room selection with toggle buttons (select all / deselect all) and pass count configuration (1x, 2x, 3x, 4x).
- 📖 **Command Help (`/help`)**:
  - Comprehensive, structured list of all bot text commands, categorized into user and administrator functions.
- 🏠 **Auto-Empty Dock & Station Control**:
  - Trigger dustbin emptying into the station dust bag.
  - Start mop washing at the dock.
  - Start and stop hot-air mop drying.
  - Configure washing water temperature and drying duration.
  - Return robot to dock (`/home`).
- 👥 **Multi-User Access & Role-Based Access Control (RBAC)**:
  - Protection against unauthorized access. First-time users require admin approval.
  - Inline `Approve` and `Reject` buttons for administrators.
  - User management menu (`/users`) with access revocation.
  - Comprehensive audit log of user actions (`/audit`).
  - Per-user preferences for language and notification subscriptions (errors, reports, station).
- 🧹 **Consumables Monitoring**:
  - Accurate remaining lifetime tracking for main/side brushes, HEPA filter, sensors, and mops.
  - Text-based graphical progress bars.
  - Instant consumable reset buttons after replacement or cleaning.
- 📊 **System Resources & Hardware Monitoring (`/resources`)**:
  - **Bot Process**: heap allocation (Heap Alloc/Sys), goroutine count, GC pause durations & cycle counts, bot uptime, Go version, and architecture.
  - **Robot Host (Linux)**: RAM usage (`/proc/meminfo`), CPU load averages (`/proc/loadavg`), disk usage (`/data` and `/`), CPU/SoC thermal sensors (`/sys/class/thermal`), and OS uptime (`/proc/uptime`).
- 🏎 **Telemetry & Logbook (`/stats`)**:
  - Detailed powertrain state, battery percentage.
  - Fresh and dirty water tank statuses, detergent cartridge, and dust bag state.
  - Statistics for the last session and lifetime cleaning history (hours, area, total runs).
- 📡 **Autonomous Background Watcher (`WatcherService`)**:
  - Instant push notifications on errors with localized description.
  - Warnings for low clean water or full dirty water tank.
  - Automatic generation of detailed post-cleaning session reports (elapsed time, cleaned area, battery consumption, visited rooms) accompanied by a rendered PNG floor map.
- 📱 **Single-Message Smart Dashboard**:
  - Single interactive dashboard message with contextual buttons reflecting current robot state.
  - Auto-cleanup of user command messages and automatic dashboard recreation on chat clear.
- 🌙 **DND (Do Not Disturb) Mode**: Automatic notification muting during night hours.
- 🛡 **Reliability**: Graceful Shutdown (`SIGINT`, `SIGTERM`), built-in process supervisor with time synchronization guard against cold-boot clock skew, and PID locking preventing duplicate instances.

---

## ⚡️ Quick Start

### 🤖 Directly on the Robot (via SSH) or Linux / macOS

> [!NOTE]
> Valetudo robots (Dreame, Roborock, etc.) run a lightweight BusyBox-based Linux environment providing **`/bin/sh`** rather than `bash`. The `install.sh` web installer is written in pure POSIX `sh` and automatically supports both `curl` and `wget`.

**Install latest version (one-liner):**
```sh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh 2>/dev/null || wget -qO- https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh)"
```

**Or download the script first:**
```sh
wget -qO /tmp/install.sh https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh || curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh -o /tmp/install.sh
sh /tmp/install.sh
```

**Install a specific version (e.g. `1.0.13`):**
```sh
wget -qO /tmp/install.sh https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh || curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh -o /tmp/install.sh
sh /tmp/install.sh 1.0.13
```

---

### 🪟 Windows (PowerShell)

Install the latest version directly from your computer:
```powershell
irm https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.ps1 | iex
```

Install a specific version (e.g. `1.0.13`):
```powershell
$Version="1.0.13"; irm https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.ps1 | iex
```

---

### Terminal Wizard Steps:
1. 🌍 **Wizard language** (`1` — Русский, `2` — English, `3` — Deutsch, `4` — 简体中文).
2. 🌐 **Robot IP address** (when running from PC) for SSH connection.
3. 🤖 **Bot token** from [@BotFather](https://t.me/BotFather).
4. 🆔 **Your Telegram ID** from [@userinfobot](https://t.me/userinfobot).
5. 🚀 **Autostart** on robot boot (`[Y/n]`, default: `Y`).

> [!TIP]
> The installer automatically fetches the precompiled ARM64 binary from GitHub Releases, sets up the `run.sh` supervisor, creates `/data/tgbot/.env`, sets file permissions, enables autostart via `/data/_root.sh`, and launches the bot service in the background.

---

### Verifying Installation:
- Open the bot in Telegram and send `/start` — the interactive control dashboard will appear.
- View live logs on the robot in real time:
  ```bash
  ssh root@<ROBOT_IP> "tail -f /tmp/log/custom/tgbot.log"
  ```

<details>
<summary>🛠 Alternative installation methods (from PC or local build)</summary>

**Installation from local computer (interactive build & deploy wizard):**
- **macOS / Linux / Git Bash:** `./start.sh`
- **Windows (PowerShell):** `.\start.ps1`

**Manual cross-compilation and deployment to the robot:**
1. Cross-compile on PC:
   ```bash
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o tgbot ./cmd/bot
   ```
2. Create directory and `.env` file on the robot:
   ```bash
   ssh root@<ROBOT_IP> "mkdir -p /data/tgbot && printf 'BOT_TOKEN=%s\nCHAT_ID=%s\n' '<YOUR_BOT_TOKEN>' '<YOUR_TELEGRAM_ID>' > /data/tgbot/.env"
   ```
3. Copy files and launch:
   ```bash
   scp tgbot run.sh root@<ROBOT_IP>:/data/tgbot/
   ssh root@<ROBOT_IP> "chmod +x /data/tgbot/tgbot /data/tgbot/run.sh && nohup /data/tgbot/run.sh >/dev/null 2>&1 &"
   ```
</details>

---

## 📁 Project Structure

The codebase is logically organized into decoupled layers and dedicated services:

```text
.
├── cmd/
│   └── bot/
│       └── main.go                  # Application entry point, initialization & Graceful Shutdown
│
├── internal/
│   ├── bot/
│   │   ├── domain/                  # Entities & Ports layer
│   │   │   ├── models.go            # Domain models (CleaningSession, Report, RoomInfo, HostStats)
│   │   │   └── ports.go             # Dependency interfaces (RobotClient, Messenger, UserRepository)
│   │   │
│   │   ├── service/                 # Use Cases & Business logic layer (Single Responsibility)
│   │   │   ├── auth/                # Permissions, access approval, user roles
│   │   │   ├── cleaning/            # State machine for segmented room cleaning wizard
│   │   │   ├── consumables/         # Consumable wear calculation & reset handling
│   │   │   ├── session/             # Cleaning session tracking & report generation
│   │   │   ├── system/              # Linux OS metrics & Go runtime telemetry
│   │   │   ├── update/              # GitHub release checks, download & self-updating
│   │   │   └── watcher/             # Background polling loop & alert dispatcher
│   │   │
│   │   ├── delivery/                # Presentation / Delivery adapters layer
│   │   │   └── telegram/
│   │   │       ├── formatters.go    # Message templates, progress bars & data formatting
│   │   │       ├── menus.go         # Inline & reply keyboard generators
│   │   │       └── handler.go       # Telegram command router & callback query handler
│   │   │
│   └── bot.go                       # Dependency injection container (Composition Root & Facade)
│   │
│   ├── database/                    # Data storage (SQLite, migrations, metadata, audit log)
│   ├── config/                      # Environment configuration loading & validation
│   ├── i18n/                        # Embedded multi-language support (ru, en, de, zh) via go:embed
│   ├── telegram/                    # Lightweight Telegram Bot API HTTP client
│   ├── valetudo/                    # Valetudo local REST API HTTP client
│   └── version/                     # Bot versioning & SemVer parser
│
├── .github/workflows/               # Automated CI release builds on tag push
│   └── release.yml
├── install.sh                       # One-liner web installer (Linux/macOS/Robot SSH: sh)
├── install.ps1                      # One-liner web installer (Windows: irm | iex)
├── start.sh                         # Interactive local build & deploy wizard (macOS/Linux/Bash)
├── start.ps1                        # Interactive local build & deploy wizard (Windows PowerShell)
├── deploy.ps1                       # Direct SSH/SCP deployment script
├── run.sh                           # Process supervisor script for the robot
├── Dockerfile                       # Multi-stage Docker build
└── go.mod
```

---

## ⚙️ Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `BOT_TOKEN` | **Yes** | — | Telegram Bot token from `@BotFather` |
| `CHAT_ID` | **Yes** | — | Telegram chat ID of the primary administrator |
| `DB_PATH` | No | `/data/tgbot/bot.db` or `bot.db` | Path to SQLite database file (users, audit, settings, metadata) |
| `BOT_LANG` | No | `ru` | Default interface language (`ru`, `en`, `de`, `zh`) |
| `VALETUDO_BASE_URL` | No | `http://127.0.0.1/api/v2/robot` | Base URL of Valetudo REST API |
| `TG_API_BASE` | No | `https://api.telegram.org` | Telegram Bot API base URL (or custom local Bot API server) |
| `DND_ENABLED` | No | `true` | Enable nocturnal Do Not Disturb (silent) mode |
| `DND_START_HOUR` | No | `23` | DND start hour (0–23) |
| `DND_END_HOUR` | No | `8` | DND end hour (0–23) |
| `ROOM_ALIASES` | No | `{}` | JSON dictionary of custom room names (e.g. `{"10":"Living Room","2":"Kitchen"}`) |
| `GITHUB_REPO` | No | `melil/valetudo-telegram-bot` | Target GitHub repository for release checks and updates |
| `GITHUB_TOKEN` | No | — | GitHub Personal Access Token (optional, for private repos or rate limit relief) |
| `UPDATE_CHECK_INTERVAL` | No | `6h` | Background check interval for new releases |
| `AUTO_UPDATE_NOTIFY` | No | `true` | Automatically notify admins when a new update is available |

---

## 🛠 Development & Testing

### 1. Local Development Build

Build the binary for running locally on your development machine:
```bash
go build -o tgbot ./cmd/bot
```

Run all unit and integration tests:
```bash
go test -v ./...
```

### 2. Docker Container Build

```bash
docker build -t valetudo-tgbot .
```

---

## 📦 Running on the Robot via Supervisor (`run.sh`)

On the robot, create the environment file `/data/tgbot/.env`:
```env
BOT_TOKEN=123456789:ABCdef...
CHAT_ID=123456789
VALETUDO_BASE_URL=http://127.0.0.1/api/v2/robot
DB_PATH=/data/tgbot/bot.db
BOT_LANG=ru
DND_ENABLED=true
```

The [`run.sh`](run.sh) supervisor ensures:
- Duplicate process prevention (PID lockfile in `/var/run/tgbot_run.pid`).
- Clock synchronization guard (`year >= 2024`), preventing Telegram TLS handshake failures on cold robot reboots.
- Automatic bot process restart on crashes or unexpected exits.
- Log rotation in `/tmp/log/custom/tgbot.log`.

Viewing bot logs on the robot:
```bash
ssh root@<ROBOT_IP> "tail -f /tmp/log/custom/tgbot.log"
```

---

## 🔄 Versioning & Over-The-Air (OTA) Updates

The bot supports an automated versioning, release, and OTA update cycle directly through Telegram:

### 1. Publishing a New Release
Create and push a git tag to the repository:
```bash
git tag 1.0.1
git push origin 1.0.1
```
The GitHub Actions workflow ([`.github/workflows/release.yml`](.github/workflows/release.yml)) will:
1. Compile the `tgbot` binary for `linux/arm64`, embedding the version string via `-ldflags`.
2. Automatically create a GitHub Release with the attached binary and release notes.

### 2. Receiving Updates in the Bot
- **Automatically**: The bot checks the GitHub API every 6 hours. When a new version is released, admins receive a message with interactive buttons:
  - `[ 🚀 Update ]` — Downloads the binary, atomically replaces `/data/tgbot/tgbot`, and restarts via the `run.sh` supervisor. Once started, the bot sends a confirmation of successful launch.
  - `[ ⏳ Later ]` — Postpones the notification for this release.
- **Manually via Menu**: In Telegram, navigate to **«🤖 Robot»** → **«🤖 Bot Settings»** → **«🔄 Updates»** (or send `/update`), where the bot checks for new releases and offers one-click installation.

---

## 📄 License

MIT License.
