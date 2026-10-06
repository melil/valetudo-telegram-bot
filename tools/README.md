# 🛠 Dreame Vacuum Local Control & Offline Isolation Fix

English | [Русский](#-описание-на-русском)

Fix for Dreame vacuum robots running [Valetudo](https://valetudo.cloud/) where control commands (`BasicControlCapability`, `LocateCapability`, `ZoneCleaningCapability`, etc.) fail with **`Error code -1`** approximately 1–2 minutes after boot.

---

## 🇬🇧 English Guide (Ready to forward)

### The Issue
On rooted Dreame vacuums running Valetudo, the stock firmware executes background telemetry verification checks (`/ava/script/msg_cvt.sh chk_ip_limit` &rarr; `/ava/script/curl_server.sh ip_limit`) around 1–2 minutes after boot.

Because `/etc/nsswitch.conf` is omitted in the squashfs rootfs of the robot, system glibc defaults to querying upstream DNS first (`dns [!UNAVAIL=return] files`), completely bypassing the local `/etc/hosts` blackholes installed by rooting scripts. When the robot connects to external cloud verification endpoints without an active official cloud subscription/region match, the daemon enters a locked state (`LockCode: 401`). 

As a result, the underlying `ava` robot controller process rejects all Valetudo control commands with **`Error code -1`**, while passive telemetry (map, sensors, battery) continues to display normally.

---

### ⚡️ Quick Automated Fix (Run via SSH on the robot)

Connect via SSH to your robot as `root` and run:

```sh
curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/tools/dreame_offline_fix.sh -o /tmp/fix.sh && sh /tmp/fix.sh && rm -f /tmp/fix.sh
```

*(Alternatively, copy [`dreame_offline_fix.sh`](dreame_offline_fix.sh) to the robot via SCP and run `sh dreame_offline_fix.sh`)*

---

### What the Fix Does
1. **Instant Unlock:** Sends `device_lock 200` to the hardware controller via `avacmd` to unlock the robot immediately without rebooting.
2. **Telemetry Wrapper:** Backs up `/ava/script/curl_server.sh` and bind-mounts a lightweight wrapper (`/data/scripts/curl_server.sh`) over it. Whenever the firmware attempts verification checks (`ip_limit` or `nation_limit`), the wrapper instantly confirms local-control ready state (`code 200`) without initiating outbound connections.
3. **Firewall Isolation:** Adds `iptables` DROP rules for external verification ports `15540` and `15541` to prevent cloud lock inquiry packets from leaving the robot.
4. **Boot Persistence:** Automatically configures `/data/_root.sh` and `/data/_root_postboot.sh` so the bind-mount, firewall rules, and unlock watchdog survive subsequent system reboots.

---

### Verification
In the Valetudo Web UI:
- Click **Locate Robot** or **Start Cleaning**.
- The robot will respond with voice/sound and start cleaning immediately without errors.
- Verification command via SSH:
  ```sh
  grep "LockCode" /data/config/ava/iot_conf.json
  # Output should show: "LockCode":200
  ```

---

## 🇷🇺 Описание на русском

### Причина сбоя
Через 1–2 минуты после загрузки робота встроенный демон прошивки Dreame опрашивает китайские облачные эндпоинты проверки статуса устройства. Поскольку в корневой squashfs-системе робота отсутствует файл `/etc/nsswitch.conf`, резолвер glibc по умолчанию отправляет запросы внешнему DNS роутера, игнорируя локальные заглушки `/etc/hosts`.

Облако отклоняет запрос, скрипт выставляет `LockCode: 401`, и системный демон `ava` блокирует управление железом робота — Valetudo выдаёт ошибку `Error code -1` на любые управляющие команды (старт уборки, локация, отправка на базу).

### Установка исправления
Выполните команду на роботе через SSH:

```sh
curl -fsSL https://raw.githubusercontent.com/melil/valetudo-telegram-bot/main/tools/dreame_offline_fix.sh -o /tmp/fix.sh && sh /tmp/fix.sh && rm -f /tmp/fix.sh
```

Скрипт полностью автономен, безопасен, восстанавливает статус `LockCode: 200` и закрепляет настройки в автозагрузке `/data/_root.sh`.
