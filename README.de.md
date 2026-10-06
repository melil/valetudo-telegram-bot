# 🤖 Valetudo Telegram Bot

[English](README.md) | [Русский](README.ru.md) | **Deutsch** | [简体中文](README.zh.md)

[![Vibe](https://img.shields.io/badge/vibecoded-with%20love-ff69b4.svg)](https://github.com/melil/valetudo-telegram-bot)
[![Go Version](https://img.shields.io/github/go-mod/go-version/melil/valetudo-telegram-bot?color=00ADD8&logo=go&logoColor=white)](https://github.com/melil/valetudo-telegram-bot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Ein leichtgewichtiger, autonomer Telegram-Bot in Go zur Steuerung von Saug- und Wischrobotern mit [Valetudo v2](https://valetudo.cloud/)-Firmware über die lokale REST-API. Getestet auf dem Dreame X30 Pro (unterstützt auch alle anderen Valetudo-kompatiblen Dreame-, Roborock- und ähnlichen Modelle).

Ausschließlich mit der Go-Standardbibliothek ohne externe Laufzeitabhängigkeiten entwickelt und zu einer **einzelnen kompakten statischen Binärdatei (~6 MB)** kompiliert. Extrem ressourceneffizient mit durchschnittlich nur **~12 MB RAM-Verbrauch**, ideal für den dauerhaften Betrieb direkt auf dem Embedded-Linux-System des Roboters (`/data/tgbot`), einem Heimrouter (OpenWrt) oder einem Server.

---

<p align="center">
  <img src="demo/de.gif" width="400px" alt="Demo">
</p>

---

## 🚀 Funktionen

- 🪄 **Reinigungsassistent (Wizard) & Schnellreinigung**:
  - Ein-Klick-Schnellstart für die standardmäßige Komplettreinigung direkt vom Haupt-Dashboard (`/clean`).
  - Interaktiver Schritt-für-Schritt-Reinigungsassistent für Räume (`/wizard`): Modusauswahl (Saugen, Wischen, Kombi, erst Saugen dann Wischen).
  - Dynamische Raumabfrage von der aktuellen Karte mit Unterstützung für benutzerdefinierte Raumnamen (`ROOM_ALIASES`).
  - Raumauswahl über Umschalt-Buttons (Alle auswählen / abwählen) und Konfiguration der Durchgänge (1x, 2x, 3x, 4x).
- 📖 **Befehlsübersicht (`/help`)**:
  - Strukturierte Liste aller Bot-Textbefehle, unterteilt in Benutzer- und Administratorfunktionen.
- 🏠 **Absaugstation- & Dock-Steuerung**:
  - Staubbehälterentleerung in den Stationsbeutel auslösen.
  - Moppwäsche an der Station starten.
  - Heißlufttrocknung der Wischpads starten und stoppen.
  - Wassertemperatur beim Waschen und Trocknungsdauer konfigurieren.
  - Roboter zur Basis zurücksenden (`/home`).
- 👥 **Mehrbenutzerzugriff & Rollenverwaltung (RBAC)**:
  - Schutz vor unbefugtem Zugriff. Neue Benutzer müssen vom Administrator genehmigt werden.
  - Inline-Buttons `Genehmigen` (`Approve`) und `Ablehnen` (`Reject`) für Administratoren.
  - Benutzerverwaltungsmenü (`/users`) mit Möglichkeit zum Widerruf von Rechten.
  - Vollständiges Audit-Protokoll aller Benutzeraktionen (`/audit`).
  - Individuelle Spracheinstellungen und Benachrichtigungsabonnements (Fehler, Berichte, Station).
- 🧹 **Verschleißüberwachung**:
  - Präzise Berechnung der Restlebensdauer für Haupt- und Seitenbürsten, HEPA-Filter, Sensoren und Wischpads.
  - Textbasierte grafische Fortschrittsbalken.
  - Sofortige Reset-Buttons nach Austausch oder Reinigung.
- 📊 **Systemressourcen & Hardware-Monitoring (`/resources`)**:
  - **Bot-Prozess**: Heap-Allokation (Heap Alloc/Sys), Goroutine-Anzahl, GC-Pausendauer & Zyklen, Bot-Uptime, Go-Version und Prozessorarchitektur.
  - **Roboter-Host (Linux)**: RAM-Auslastung (`/proc/meminfo`), CPU-Load-Average (`/proc/loadavg`), Speicherplatz (`/data` und `/`), CPU/SoC-Temperatursensoren (`/sys/class/thermal`) und OS-Uptime (`/proc/uptime`).
- 🏎 **Telemetrie & Fahrtenbuch (`/stats`)**:
  - Detaillierter Antriebsstatus, Akkuladestand.
  - Füllstand von Frisch- und Schmutzwassertank, Reinigungsmittelkartusche und Staubbeutel.
  - Statistiken der letzten Reinigung und Gesamthistorie (Stunden, Quadratmeter, Reinigungszyklen).
- 📡 **Autonomer Hintergrund-Watcher (`WatcherService`)**:
  - Sofortige Push-Benachrichtigungen bei Fehlern mit verständlicher Beschreibung.
  - Warnungen bei niedrigem Frischwasser- oder vollem Schmutzwasserstand.
  - Automatische Erstellung detaillierter Abschlussberichte (Dauer, gereinigte Fläche, Akkuverbrauch, gereinigte Räume) inklusive gerenderter PNG-Kartenansicht.
- 📱 **Smartes Ein-Nachrichten-Dashboard**:
  - Einzelne interaktive Dashboard-Nachricht mit kontextuellen Buttons passend zum aktuellen Roboterstatus.
  - Automatisches Bereinigen von Servicebefehlen und Wiederherstellung des Dashboards beim Leeren des Chatverlaufs.
- 🌙 **DND-Modus (Bitte nicht stören)**: Automatische Stummschaltung von Benachrichtigungen in der Nacht.
- 🛡 **Zuverlässigkeit**: Graceful Shutdown (`SIGINT`, `SIGTERM`), integrierter Supervisor mit Schutz vor Zeitabweichungen beim Kaltstart und PID-Sperre gegen Mehrfachinstanzen.

---

## ⚡️ Schnellstart

### 🤖 Direkt auf dem Roboter (via SSH) oder Linux / macOS

> [!NOTE]
> Auf Valetudo-Robotern (Dreame, Roborock etc.) läuft ein leichtgewichtiges BusyBox-Linux, auf dem **`/bin/sh`** statt `bash` bereitgestellt wird. Das Online-Installationsskript `install.sh` ist in reinem POSIX `sh` verfasst und unterstützt automatisch sowohl `curl` als auch `wget`.

**Installation der neuesten Version (Einzeiler):**
```sh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh 2>/dev/null || wget -qO- https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh)"
```

**Oder Skript herunterladen und ausführen:**
```sh
wget -qO /tmp/install.sh https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh || curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh -o /tmp/install.sh
sh /tmp/install.sh
```

**Installation einer bestimmten Version (z. B. `1.0.13`):**
```sh
wget -qO /tmp/install.sh https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh || curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh -o /tmp/install.sh
sh /tmp/install.sh 1.0.13
```

---

### 🪟 Windows (PowerShell)

Installation der neuesten Version direkt von Ihrem PC aus:
```powershell
irm https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.ps1 | iex
```

Installation einer bestimmten Version (z. B. `1.0.13`):
```powershell
$Version="1.0.13"; irm https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.ps1 | iex
```

---

### Schritte des Installationsassistenten:
1. 🌍 **Sprachauswahl** (`1` — Русский, `2` — English, `3` — Deutsch, `4` — 简体中文).
2. 🌐 **IP-Adresse des Roboters** (bei Ausführung vom PC) für die SSH-Verbindung.
3. 🤖 **Bot-Token** von [@BotFather](https://t.me/BotFather).
4. 🆔 **Ihre Telegram-ID** von [@userinfobot](https://t.me/userinfobot).
5. 🚀 **Autostart** beim Booten des Roboters (`[Y/n]`, Standard: `Y`).

> [!TIP]
> Der Assistent lädt automatisch das vorkompilierte ARM64-Binary aus den GitHub Releases herunter, richtet den `run.sh`-Supervisor ein, generiert `/data/tgbot/.env`, setzt Dateiberechtigungen, aktiviert den Autostart über `/data/_root.sh` und startet den Bot im Hintergrund.

---

### Installation überprüfen:
- Öffnen Sie den Bot in Telegram und senden Sie den Befehl `/start` — das interaktive Dashboard erscheint.
- Live-Logs auf dem Roboter einsehen:
  ```bash
  ssh root@<ROBOT_IP> "tail -f /tmp/log/custom/tgbot.log"
  ```

<details>
<summary>🛠 Alternative Installationsmethoden (vom PC oder lokaler Build)</summary>

**Installation vom lokalen Rechner (interaktiver Build- & Deploy-Assistent):**
- **macOS / Linux / Git Bash:** `./start.sh`
- **Windows (PowerShell):** `.\start.ps1`

**Manuelle Cross-Kompilierung und Bereitstellung auf dem Roboter:**
1. Cross-Kompilierung auf dem PC:
   ```bash
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o tgbot ./cmd/bot
   ```
2. Verzeichnis und `.env`-Datei auf dem Roboter anlegen:
   ```bash
   ssh root@<ROBOT_IP> "mkdir -p /data/tgbot && printf 'BOT_TOKEN=%s\nCHAT_ID=%s\n' '<YOUR_BOT_TOKEN>' '<YOUR_TELEGRAM_ID>' > /data/tgbot/.env"
   ```
3. Dateien übertragen und starten:
   ```bash
   scp tgbot run.sh root@<ROBOT_IP>:/data/tgbot/
   ssh root@<ROBOT_IP> "chmod +x /data/tgbot/tgbot /data/tgbot/run.sh && nohup /data/tgbot/run.sh >/dev/null 2>&1 &"
   ```
</details>

---

## 📁 Projektstruktur

Die Codebasis ist klar in Schichten und dedizierte Services unterteilt:

```text
.
├── cmd/
│   └── bot/
│       └── main.go                  # Einstiegspunkt, Initialisierung & Graceful Shutdown
│
├── internal/
│   ├── bot/
│   │   ├── domain/                  # Entitäten & Schnittstellen (Entities & Ports)
│   │   │   ├── models.go            # Domänenmodelle (CleaningSession, Report, RoomInfo, HostStats)
│   │   │   └── ports.go             # Abhängigkeitsschnittstellen (RobotClient, Messenger, UserRepository)
│   │   │
│   │   ├── service/                 # Anwendungsfall- & Geschäftslogikschicht (Single Responsibility)
│   │   │   ├── auth/                # Berechtigungen, Zugriffsprüfung, Benutzerrollen
│   │   │   ├── cleaning/            # Zustandsmaschine des Reinigungsassistenten
│   │   │   ├── consumables/         # Verschleißberechnung & Zähler-Reset
│   │   │   ├── session/             # Sitzungsverfolgung & Berichtserstellung
│   │   │   ├── system/              # Linux-OS-Metriken & Go-Runtime-Telemetrie
│   │   │   ├── update/              # GitHub-Releaseprüfung, Download & Selbstaktualisierung
│   │   │   └── watcher/             # Hintergrund-Überwachung & Alarmverteiler
│   │   │
│   │   ├── delivery/                # Präsentationsschicht (Presentation / Delivery)
│   │   │   └── telegram/
│   │   │       ├── formatters.go    # Nachrichtenvorlagen, Fortschrittsbalken & Formatierung
│   │   │       ├── menus.go         # Inline- & Reply-Tastaturgeneratoren
│   │   │       └── handler.go       # Telegram-Befehlsrouter & Callback-Handler
│   │   │
│   └── bot.go                       # Dependency-Injection-Container (Composition Root & Fassade)
│   │
│   ├── database/                    # Datenspeicherung (SQLite, Migrationen, Metadaten, Audit-Log)
│   ├── config/                      # Laden & Validieren von Umgebungsvariablen
│   ├── i18n/                        # Integrierte Mehrsprachigkeit (ru, en, de, zh) via go:embed
│   ├── telegram/                    # Schlanker HTTP-Client für die Telegram-Bot-API
│   ├── valetudo/                    # HTTP-Client für die lokale Valetudo-REST-API
│   └── version/                     # Bot-Versionierung & SemVer-Parser
│
├── .github/workflows/               # Automatisierte CI-Release-Builds bei Git-Tags
│   └── release.yml
├── install.sh                       # Einzeiler-Webinstaller (Linux/macOS/Roboter SSH: sh)
├── install.ps1                      # Einzeiler-Webinstaller (Windows: irm | iex)
├── start.sh                         # Interaktiver lokaler Build-Assistent (macOS/Linux/Bash)
├── start.ps1                        # Interaktiver lokaler Build-Assistent (Windows PowerShell)
├── deploy.ps1                       # Direktes SSH/SCP-Deployment-Skript
├── run.sh                           # Prozess-Supervisor für den Roboter
├── Dockerfile                       # Multi-Stage-Docker-Build
└── go.mod
```

---

## ⚙️ Umgebungsvariablen

| Variable | Erforderlich | Standardwert | Beschreibung |
|---|---|---|---|
| `BOT_TOKEN` | **Ja** | — | Telegram-Bot-Token von `@BotFather` |
| `CHAT_ID` | **Ja** | — | Telegram-Chat-ID des Hauptadministrators |
| `DB_PATH` | Nein | `/data/tgbot/bot.db` oder `bot.db` | Pfad zur SQLite-Datenbankdatei (Benutzer, Audit, Einstellungen) |
| `BOT_LANG` | Nein | `ru` | Standard-Oberflächensprache (`ru`, `en`, `de`, `zh`) |
| `VALETUDO_BASE_URL` | Nein | `http://127.0.0.1/api/v2/robot` | Basis-URL der lokalen Valetudo-REST-API |
| `TG_API_BASE` | Nein | `https://api.telegram.org` | Basis-URL der Telegram-Bot-API (oder lokaler Bot-API-Server) |
| `DND_ENABLED` | Nein | `true` | Nächtlichen Ruhemodus (Nicht stören) aktivieren |
| `DND_START_HOUR` | Nein | `23` | Startstunde des Ruhemodus (0–23) |
| `DND_END_HOUR` | Nein | `8` | Endstunde des Ruhemodus (0–23) |
| `ROOM_ALIASES` | Nein | `{}` | JSON-Objekt mit benutzerdefinierten Raumnamen (z. B. `{"10":"Wohnzimmer","2":"Küche"}`) |
| `GITHUB_REPO` | Nein | `melil/valetudo-telegram-bot` | GitHub-Repository für Versionsprüfungen und Downloads |
| `GITHUB_TOKEN` | Nein | — | GitHub Personal Access Token (optional für private Repos oder Rate-Limits) |
| `UPDATE_CHECK_INTERVAL` | Nein | `6h` | Intervall für die automatische Suche nach Updates |
| `AUTO_UPDATE_NOTIFY` | Nein | `true` | Administratoren automatisch benachrichtigen, wenn ein Update verfügbar ist |

---

## 🛠 Entwicklung & Tests

### 1. Lokaler Build für die Entwicklung

Kompilieren der Binärdatei für lokale Tests:
```bash
go build -o tgbot ./cmd/bot
```

Alle Unit- und Integrationstests ausführen:
```bash
go test -v ./...
```

### 2. Docker-Container erstellen

```bash
docker build -t valetudo-tgbot .
```

---

## 📦 Betrieb auf dem Roboter via Supervisor (`run.sh`)

Erstellen Sie auf dem Roboter die Datei `/data/tgbot/.env`:
```env
BOT_TOKEN=123456789:ABCdef...
CHAT_ID=123456789
VALETUDO_BASE_URL=http://127.0.0.1/api/v2/robot
DB_PATH=/data/tgbot/bot.db
BOT_LANG=de
DND_ENABLED=true
```

Der Supervisor [`run.sh`](run.sh) gewährleistet:
- Schutz vor Doppelstarts (PID-Lockdatei in `/var/run/tgbot_run.pid`).
- Warten auf Zeitsynchronisation (`year >= 2024`), um TLS-Zertifikatsfehler beim Kaltstart zu vermeiden.
- Automatischen Neustart des Bots bei unerwarteten Abstürzen.
- Log-Rotation in `/tmp/log/custom/tgbot.log`.

Live-Logs auf dem Roboter einsehen:
```bash
ssh root@<ROBOT_IP> "tail -f /tmp/log/custom/tgbot.log"
```

---

## 🔄 Versionierung & Over-The-Air (OTA) Updates

Der Bot unterstützt einen automatisierten Release- und OTA-Update-Zyklus direkt über Telegram:

### 1. Neue Version veröffentlichen
Git-Tag erstellen und pushen:
```bash
git tag 1.0.1
git push origin 1.0.1
```
Der GitHub Actions Workflow ([`.github/workflows/release.yml`](.github/workflows/release.yml)) wird:
1. Die Binärdatei `tgbot` für `linux/arm64` mit Versionsinformationen über `-ldflags` kompilieren.
2. Automatisch ein GitHub Release mit angehängter Binärdatei und Changelog anlegen.

### 2. Updates im Bot empfangen
- **Automatisch**: Der Bot prüft die GitHub-API alle 6 Stunden. Bei einer neuen Version erhalten Administratoren eine Nachricht mit Aktions-Buttons:
  - `[ 🚀 Aktualisieren ]` — Lädt die Binärdatei herunter, ersetzt `/data/tgbot/tgbot` atomar und startet den Bot über den Supervisor neu.
  - `[ ⏳ Später ]` — Verschiebt die Erinnerung für diese Version.
- **Manuell über das Menü**: Navigieren Sie in Telegram zu **«🤖 Roboter»** → **«🤖 Bot-Einstellungen»** → **«🔄 Updates»** (oder senden Sie `/update`), um nach Updates zu suchen und diese per Klick zu installieren.

---

## 📄 Lizenz

MIT License.
