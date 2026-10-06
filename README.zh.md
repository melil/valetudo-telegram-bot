# 🤖 Valetudo Telegram Bot

[English](README.md) | [Русский](README.ru.md) | [Deutsch](README.de.md) | **简体中文**

[![Vibe](https://img.shields.io/badge/vibecoded-with%20love-ff69b4.svg)](https://github.com/melil/valetudo-telegram-bot)
[![Go Version](https://img.shields.io/github/go-mod/go-version/melil/valetudo-telegram-bot?color=00ADD8&logo=go&logoColor=white)](https://github.com/melil/valetudo-telegram-bot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

一款用 Go 语言编写的高性能、轻量级、完全自主运行的 Telegram 机器人，专为搭载 [Valetudo v2](https://valetudo.cloud/) 固件的扫地机器人设计，通过本地 REST API 进行全功能控制。在 Dreame X30 Pro 上经过充分测试，同时支持其他搭载 Valetudo 的 Dreame、Roborock 等兼容机型。

完全基于 Go 标准库开发，无外部运行时依赖，编译后为**单一紧凑的静态二进制文件（~6 MB）**。资源占用极低，平均仅消耗 **~12 MB 内存**，非常适合直接常驻运行在扫地机器人嵌入式 Linux 系统（`/data/tgbot`）、家用软路由（OpenWrt）或微型服务器中。

---

## 🚀 功能特性

- 🪄 **清扫向导 (Wizard) 与快速清扫**:
  - 主面板一键快捷启动全屋标准清扫（`/clean`）。
  - 交互式分步选区清扫向导（`/wizard`）：支持模式选择（仅扫地、仅拖地、扫拖一体、先扫后拖）。
  - 从当前地图动态加载房间分区列表，支持用户自定义房间别名（`ROOM_ALIASES`）。
  - 房间复选框按钮切换（全选 / 取消全选）及清扫遍数设置（1x, 2x, 3x, 4x）。
- 📖 **命令帮助列表 (`/help`)**:
  - 结构化展示所有 Bot 文本命令，清晰区分用户功能与管理员权限命令。
- 🏠 **自清洁基站控制**:
  - 触发尘盒集尘至基站尘袋。
  - 启动基站清洗拖布。
  - 启动 / 手动停止热风烘干拖布。
  - 调节洗拖布水温及热风烘干时长。
  - 召回机器人回基站充电（`/home`）。
- 👥 **多用户访问控制与权限管理 (RBAC)**:
  - 防范未授权访问，新用户首次进入需管理员审核批准。
  - 管理员专属交互式内联按钮：`Approve`（批准）与 `Reject`（拒绝）。
  - 用户管理菜单（`/users`），支持撤销权限。
  - 用户操作行为全量审计日志（`/audit`）。
  - 支持按用户独立设置语言偏好及事件通知订阅（故障报错、清扫报告、基站状态）。
- 🧹 **耗材健康寿命监控**:
  - 精确计算主刷、边刷、HEPA 滤网、传感器及拖布的剩余使用寿命与磨损度。
  - 纯文本图形化进度条直观呈现。
  - 更换或清理耗材后支持一键重置计数。
- 📊 **系统硬件与运行资源监控 (`/resources`)**:
  - **Bot 进程**：堆内存分配（Heap Alloc/Sys）、Goroutine 协程数、GC 垃圾回收暂停耗时与频次、Bot 运行时间、Go 版本与架构。
  - **机器人宿主（Linux）**：内存负载（`/proc/meminfo`）、CPU 平均负载（`/proc/loadavg`）、磁盘空间占用（`/data` 与 `/`）、CPU/SoC 核心温度传感器（`/sys/class/thermal`）及系统开机时间（`/proc/uptime`）。
- 🏎 **运行日志与传感器遥测 (`/stats`)**:
  - 详细的动力系统状态、当前电量百分比。
  - 清水箱、污水箱水位状态，清洁剂储液盒与尘袋状态。
  - 最近一次清扫指标及历史总清扫统计（时长、面积、清扫次数）。
- 📡 **自主后台监控服务 (`WatcherService`)**:
  - 故障报错毫秒级推送，附带清晰代码释义。
  - 清水不足或污水箱满即时预警。
  - 清扫完成后自动生成详尽的清扫报告（耗时、面积、耗电量、房间列表），并附带渲染生成的 PNG 房型地图。
- 📱 **单消息智能交互面板 (Single-Message Dashboard)**:
  - 交互式单消息面板，根据机器人当前状态动态变换控制按钮。
  - 自动清理用户发送的指令消息，聊天记录清空时自动重新生成主面板。
- 🌙 **夜间免打扰模式 (DND)**: 夜间时段自动静音所有通知。
- 🛡 **高可靠性保障**: 优雅退出（`SIGINT`, `SIGTERM`）、内置进程 Supervisor、冷启动时间同步校验防 TLS 报错、PID 锁防止进程重复运行。

---

## ⚡️ 快速开始

### 🤖 直接在扫地机器人上安装（通过 SSH）或 Linux / macOS

> [!NOTE]
> 在搭载 Valetudo 的机器人（Dreame、Roborock 等）上运行的是基于 BusyBox 的轻量级 Linux 系统，其提供的命令行环境为 **`/bin/sh`**，而非 `bash`。在线安装脚本 `install.sh` 采用纯 POSIX `sh` 编写，能够自动兼容 `curl` 与 `wget`。

**一键安装最新版本：**
```sh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh 2>/dev/null || wget -qO- https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh)"
```

**或先下载脚本再执行：**
```sh
wget -qO /tmp/install.sh https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh || curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh -o /tmp/install.sh
sh /tmp/install.sh
```

**安装指定版本（例如 `1.0.13`）：**
```sh
wget -qO /tmp/install.sh https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh || curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.sh -o /tmp/install.sh
sh /tmp/install.sh 1.0.13
```

---

### 🪟 Windows (PowerShell)

直接从您的电脑一键安装到机器人：
```powershell
irm https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.ps1 | iex
```

安装指定版本（例如 `1.0.13`）：
```powershell
$Version="1.0.13"; irm https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/install.ps1 | iex
```

---

### 终端安装向导步骤：
1. 🌍 **选择向导语言** (`1` — Русский, `2` — English, `3` — Deutsch, `4` — 简体中文)。
2. 🌐 **机器人 IP 地址**（从电脑运行时需要）用于 SSH 连接。
3. 🤖 **Bot Token**，来自 [@BotFather](https://t.me/BotFather)。
4. 🆔 **您的 Telegram ID**，来自 [@userinfobot](https://t.me/userinfobot)。
5. 🚀 **开机自启** 开机时随系统启动 (`[Y/n]`, 默认: `Y`)。

> [!TIP]
> 向导会自动从 GitHub Releases 下载预编译的 ARM64 二进制文件，配置 `run.sh` 守护进程，生成 `/data/tgbot/.env` 文件，赋予执行权限，并在 `/data/_root.sh` 中配置开机启动，最后在后台启动机器人服务。

---

### 验证运行状态：
- 在 Telegram 中打开机器人并发送 `/start` 命令，即可看到交互式控制面板。
- 在机器人上实时查看运行日志：
  ```bash
  ssh root@<ROBOT_IP> "tail -f /tmp/log/custom/tgbot.log"
  ```

<details>
<summary>🛠 其他安装方式（从电脑编译或本地运行）</summary>

**从本地电脑运行交互式编译部署脚本：**
- **macOS / Linux / Git Bash:** `./start.sh`
- **Windows (PowerShell):** `.\start.ps1`

**手动交叉编译并部署至机器人：**
1. 在 PC 上交叉编译：
   ```bash
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o tgbot ./cmd/bot
   ```
2. 在机器人上创建目录并写入 `.env`：
   ```bash
   ssh root@<ROBOT_IP> "mkdir -p /data/tgbot && printf 'BOT_TOKEN=%s\nCHAT_ID=%s\n' '<YOUR_BOT_TOKEN>' '<YOUR_TELEGRAM_ID>' > /data/tgbot/.env"
   ```
3. 复制文件并启动服务：
   ```bash
   scp tgbot run.sh root@<ROBOT_IP>:/data/tgbot/
   ssh root@<ROBOT_IP> "chmod +x /data/tgbot/tgbot /data/tgbot/run.sh && nohup /data/tgbot/run.sh >/dev/null 2>&1 &"
   ```
</details>

---

## 📁 项目结构

代码库结构清晰，划分为相互独立的逻辑分层与专用服务：

```text
.
├── cmd/
│   └── bot/
│       └── main.go                  # 应用程序入口、初始化与优雅退出 (Graceful Shutdown)
│
├── internal/
│   ├── bot/
│   │   ├── domain/                  # 领域实体与端口层 (Entities & Ports)
│   │   │   ├── models.go            # 领域业务模型 (CleaningSession, Report, RoomInfo, HostStats)
│   │   │   └── ports.go             # 外部依赖接口 (RobotClient, Messenger, UserRepository)
│   │   │
│   │   ├── service/                 # 业务逻辑服务层 (Use Cases / Single Responsibility)
│   │   │   ├── auth/                # 权限鉴权、用户准入审核与角色管理
│   │   │   ├── cleaning/            # 分区选区清扫向导状态机
│   │   │   ├── consumables/         # 耗材寿命磨损计算与重置
│   │   │   ├── session/             # 清扫会话生命周期跟踪与报告生成
│   │   │   ├── system/              # Linux 系统指标与 Go 运行时遥测采集
│   │   │   ├── update/              # GitHub 版本检查、下载与热更新
│   │   │   └── watcher/             # 后台轮询监控与报警推送服务
│   │   │
│   │   ├── delivery/                # 展现与交互层 (Presentation / Delivery)
│   │   │   └── telegram/
│   │   │       ├── formatters.go    # 消息模板、文本进度条与数据格式化
│   │   │       ├── menus.go         # 内联键盘与快捷回复键盘生成器
│   │   │       └── handler.go       # Telegram 指令路由与回调请求处理器
│   │   │
│   └── bot.go                       # 依赖注入容器 (Composition Root & Facade)
│   │
│   ├── database/                    # 数据持久化存储 (SQLite, 迁移, 元数据, 审计日志)
│   ├── config/                      # 环境变量读取与参数校验
│   ├── i18n/                        # 嵌入式多语言支持 (ru, en, de, zh) 通过 go:embed
│   ├── telegram/                    # 轻量级 Telegram Bot API HTTP 客户端
│   ├── valetudo/                    # Valetudo 本地 REST API HTTP 客户端
│   └── version/                     # 版本控制与 SemVer 解析器
│
├── .github/workflows/               # 推送 Tag 时自动触发 CI 发版
│   └── release.yml
├── install.sh                       # 一键在线安装脚本 (Linux/macOS/机器人 SSH: sh)
├── install.ps1                      # 一键在线安装脚本 (Windows: irm | iex)
├── start.sh                         # 本地交互式构建部署脚本 (macOS/Linux/Bash)
├── start.ps1                        # 本地交互式构建部署脚本 (Windows PowerShell)
├── deploy.ps1                       # SSH/SCP 直接部署脚本
├── run.sh                           # 机器人守护进程 Supervisor 脚本
├── Dockerfile                       # 多阶段构建 Dockerfile
└── go.mod
```

---

## ⚙️ 环境变量

| 变量名 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `BOT_TOKEN` | **是** | — | 来自 `@BotFather` 的 Telegram Bot Token |
| `CHAT_ID` | **是** | — | 主管理员的 Telegram Chat ID |
| `DB_PATH` | 否 | `/data/tgbot/bot.db` 或 `bot.db` | SQLite 数据库文件存储路径（用户、审计、设置） |
| `BOT_LANG` | 否 | `ru` | 默认界面语言（`ru`, `en`, `de`, `zh`） |
| `VALETUDO_BASE_URL` | 否 | `http://127.0.0.1/api/v2/robot` | Valetudo REST API 本地请求地址 |
| `TG_API_BASE` | 否 | `https://api.telegram.org` | Telegram Bot API 接口地址（或自定义本地反代/Bot API 服务器） |
| `DND_ENABLED` | 否 | `true` | 是否启用夜间免打扰静音模式 |
| `DND_START_HOUR` | 否 | `23` | 免打扰开始小时（0–23） |
| `DND_END_HOUR` | 否 | `8` | 免打扰结束小时（0–23） |
| `ROOM_ALIASES` | 否 | `{}` | 房间自定义别名 JSON 映射字典（例如：`{"10":"客厅","2":"厨房"}`） |
| `GITHUB_REPO` | 否 | `melil/valetudo-telegram-bot` | 用于检查版本与下载更新的 GitHub 仓库 |
| `GITHUB_TOKEN` | 否 | — | GitHub 个人访问令牌（可选，私有仓库或避免 API 频次受限） |
| `UPDATE_CHECK_INTERVAL` | 否 | `6h` | 自动检查新版本的时间间隔 |
| `AUTO_UPDATE_NOTIFY` | 否 | `true` | 有新版本可用时是否自动通知管理员 |

---

## 🛠 开发与测试

### 1. 本地开发编译

在本地开发机上编译二进制文件：
```bash
go build -o tgbot ./cmd/bot
```

运行全部单元测试与集成测试：
```bash
go test -v ./...
```

### 2. 构建 Docker 镜像

```bash
docker build -t valetudo-tgbot .
```

---

## 📦 通过 Supervisor 在机器人上运行 (`run.sh`)

在扫地机器人上创建环境配置文件 `/data/tgbot/.env`：
```env
BOT_TOKEN=123456789:ABCdef...
CHAT_ID=123456789
VALETUDO_BASE_URL=http://127.0.0.1/api/v2/robot
DB_PATH=/data/tgbot/bot.db
BOT_LANG=zh
DND_ENABLED=true
```

守护脚本 [`run.sh`](run.sh) 提供以下保障：
- 防重复启动机制（基于 `/var/run/tgbot_run.pid` 的 PID 文件锁）。
- 系统时间同步等待（`year >= 2024`），有效避免机器人冷重启时因时钟未对齐导致的 Telegram TLS 握手证书验证失败。
- 异常退出自动拉起重启。
- 日志文件滚动输出至 `/tmp/log/custom/tgbot.log`。

实时查看机器人端运行日志：
```bash
ssh root@<ROBOT_IP> "tail -f /tmp/log/custom/tgbot.log"
```

---

## 🔄 版本控制与在线更新 (OTA)

本机器人支持通过 Telegram 实现完全自动化的版本发布与 OTA 在线热更新闭环：

### 1. 发布新版本
在仓库中创建并推送 Git 标签：
```bash
git tag 1.0.1
git push origin 1.0.1
```
GitHub Actions 工作流（[`.github/workflows/release.yml`](.github/workflows/release.yml)）将自动：
1. 编译适配 `linux/arm64` 的 `tgbot` 独立二进制文件，并通过 `-ldflags` 写入版本号。
2. 自动生成 GitHub Release 并附带编译产物与更新日志。

### 2. 机器人在 Telegram 中接收更新
- **自动检测**：机器人每隔 6 小时自动查询 GitHub API。有新版本发布时，主管理员将收到包含操作按钮的消息：
  - `[ 🚀 立即更新 ]` — 下载新版本二进制文件，原子替换 `/data/tgbot/tgbot`，并通过 `run.sh` 守护进程平滑重启，启动成功后主动推送就绪通知。
  - `[ ⏳ 稍后提醒 ]` — 忽略本次提醒。
- **手动检查更新**：在 Telegram 中依次点击 **「🤖 机器人」** → **「🤖 机器人设置」** → **「🔄 检查更新」**（或直接发送 `/update`），机器人将即时查询最新版本并提供一键升级。

---

## 📄 开源许可证

MIT License.
