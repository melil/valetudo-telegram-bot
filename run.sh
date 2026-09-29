#!/bin/sh
# Valetudo Telegram Bot Supervisor for Dreame X30 Pro

DIR="/data/tgbot"
PIDFILE="/var/run/tgbot_run.pid"
LOGDIR="/tmp/log/custom"
LOGFILE="$LOGDIR/tgbot.log"

mkdir -p "$LOGDIR" "$DIR"
cd "$DIR" || exit 1

# Single-instance guard
if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] tgbot supervisor already running (pid $(cat "$PIDFILE")), exiting" >> "$LOGFILE"
    exit 0
fi
echo $$ > "$PIDFILE"

log_msg() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] supervisor[$$]: $*" >> "$LOGFILE"
}

log_msg "Supervisor started (PID $$)"

# Load environment
if [ -f "$DIR/.env" ]; then
    set -a
    . "$DIR/.env"
    set +a
else
    log_msg "ERROR: $DIR/.env not found!"
    rm -f "$PIDFILE"
    exit 1
fi

# Ensure DB is stored in persistent flash memory
export DB_PATH="${DB_PATH:-/data/tgbot/bot.db}"

BOT_PID=""

cleanup() {
    log_msg "Received stop signal, terminating tgbot..."
    if [ -n "$BOT_PID" ] && kill -0 "$BOT_PID" 2>/dev/null; then
        kill -15 "$BOT_PID" 2>/dev/null
        _w=0
        while [ $_w -lt 5 ] && kill -0 "$BOT_PID" 2>/dev/null; do
            sleep 1
            _w=$((_w + 1))
        done
        kill -9 "$BOT_PID" 2>/dev/null
    fi
    rm -f "$PIDFILE"
    exit 0
}
trap cleanup INT TERM HUP

# Wait for system clock sync (year >= 2024) to avoid TLS cert validation failure
_clk_wait=0
while [ "$(date +%Y)" -lt 2024 ] && [ $_clk_wait -lt 120 ]; do
    if [ $_clk_wait -eq 0 ]; then
        log_msg "Waiting for system time sync (NTP)..."
    fi
    sleep 2
    _clk_wait=$((_clk_wait + 2))
done
log_msg "System clock ready: $(date)"

# Wait for Valetudo API to be available (up to 90 seconds)
_val_wait=0
while [ $_val_wait -lt 90 ]; do
    if curl -s -m 2 http://127.0.0.1/api/v2/robot/capabilities >/dev/null 2>&1; then
        log_msg "Valetudo is ready after ${_val_wait}s"
        break
    fi
    sleep 2
    _val_wait=$((_val_wait + 2))
done
if [ $_val_wait -ge 90 ]; then
    log_msg "WARNING: Valetudo did not respond within 90s, starting bot anyway..."
fi

# Supervisor restart loop
while true; do
    # Rotate log if size exceeds 2 MB (2097152 bytes)
    if [ -f "$LOGFILE" ] && [ "$(wc -c < "$LOGFILE" 2>/dev/null || echo 0)" -gt 2097152 ]; then
        tail -n 2000 "$LOGFILE" > "$LOGFILE.tmp" && mv "$LOGFILE.tmp" "$LOGFILE"
    fi

    chmod +x "$DIR/tgbot" 2>/dev/null
    log_msg "Starting tgbot..."
    "$DIR/tgbot" >> "$LOGFILE" 2>&1 &
    BOT_PID=$!
    log_msg "tgbot running with PID $BOT_PID"

    wait $BOT_PID
    EXIT_CODE=$?
    log_msg "tgbot (PID $BOT_PID) exited with code $EXIT_CODE. Restarting in 3s..."
    BOT_PID=""
    sleep 3
done
