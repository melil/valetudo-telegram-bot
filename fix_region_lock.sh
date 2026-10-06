#!/bin/sh
# ==============================================================================
# Dreame Vacuum Region Lock Fix for Valetudo (Permanent Bypass)
# ==============================================================================
# Fixes "Error code -1" on BasicControlCapability, LocateCapability, etc.
# caused by Dreame cloud region verification (LockCode 401).
# Compatible with: Dreame X30 Pro, L10s Ultra, L20 Ultra, X40, and other models.
# ==============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { printf "${CYAN}[*] %s${NC}\n" "$1"; }
log_ok()    { printf "${GREEN}[+] %s${NC}\n" "$1"; }
log_warn()  { printf "${YELLOW}[!] %s${NC}\n" "$1"; }
log_err()   { printf "${RED}[-] %s${NC}\n" "$1"; }

# 1. Root check
if [ "$(id -u 2>/dev/null || echo 1)" -ne 0 ]; then
    log_err "This script must be run as root on the robot!"
    exit 1
fi

log_info "Starting Dreame Region Lock permanent bypass..."

# 2. Check for Dreame ava subsystem
if [ ! -f /ava/script/curl_server.sh ]; then
    log_err "/ava/script/curl_server.sh not found. Is this a Dreame robot?"
    exit 1
fi

# 3. Immediately unlock hardware via avacmd
log_info "Unlocking robot hardware (code 200)..."
if command -v avacmd >/dev/null 2>&1; then
    avacmd msg_cvt '{"type":"msgCvt","cmd":"device_lock","code":200}' >/dev/null 2>&1 || true
    sleep 1
fi

# 4. Enforce LockCode: 200 in configuration
if [ -f /data/config/ava/iot_conf.json ]; then
    log_info "Patching /data/config/ava/iot_conf.json..."
    sed -i 's/"LockCode":[0-9]*/"LockCode":200/g' /data/config/ava/iot_conf.json 2>/dev/null || true
fi
rm -f /data/log/devlock_record.json 2>/dev/null || true
if command -v record_events.sh >/dev/null 2>&1; then
    record_events.sh lock_code 200 2>/dev/null || true
fi

# 5. Create persistent wrapper for curl_server.sh
log_info "Creating persistent curl_server.sh wrapper..."
mkdir -p /data/scripts

# Unmount existing wrapper if already active to grab stock script
umount /ava/script/curl_server.sh 2>/dev/null || true

if [ ! -f /data/scripts/curl_server.orig.sh ]; then
    cp -a /ava/script/curl_server.sh /data/scripts/curl_server.orig.sh
    log_ok "Backed up stock curl_server.sh to /data/scripts/curl_server.orig.sh"
fi

cat << 'EOF' > /data/scripts/curl_server.sh
#!/bin/sh
# Dreame region lock bypass wrapper
case "$1" in
    ip_limit|nation_limit)
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] curl_server.sh: [BYPASS] enforcing device unlock code 200" >> /tmp/log/script.log 2>/dev/null
        avacmd msg_cvt '{"type":"msgCvt","cmd":"device_lock","code":200}' &
        rm -f /data/log/devlock_record.json 2>/dev/null
        record_events.sh lock_code 200 2>/dev/null
        exit 0
        ;;
    *)
        if [ -x /data/scripts/curl_server.orig.sh ]; then
            exec /data/scripts/curl_server.orig.sh "$@"
        fi
        ;;
esac
EOF
chmod +x /data/scripts/curl_server.sh /data/scripts/curl_server.orig.sh 2>/dev/null || true

# Bind-mount wrapper over stock script
mount --bind /data/scripts/curl_server.sh /ava/script/curl_server.sh
log_ok "Bind-mounted bypass wrapper to /ava/script/curl_server.sh"

# 6. Block outbound Dreame region check ports (iptables firewall)
if command -v iptables >/dev/null 2>&1; then
    log_info "Configuring iptables firewall rules..."
    iptables -C OUTPUT -p tcp --dport 15540 -j DROP 2>/dev/null || iptables -I OUTPUT -p tcp --dport 15540 -j DROP 2>/dev/null || true
    iptables -C OUTPUT -p tcp --dport 15541 -j DROP 2>/dev/null || iptables -I OUTPUT -p tcp --dport 15541 -j DROP 2>/dev/null || true
    log_ok "Ports 15540 and 15541 blocked in iptables"
fi

# 7. Ensure boot persistence in /data/_root.sh
log_info "Ensuring reboot persistence in /data/_root.sh..."
mkdir -p /data
if [ ! -f /data/_root.sh ]; then
    cat << 'EOF' > /data/_root.sh
#!/bin/sh
mkdir -p /tmp/log/custom /tmp/log/local_control
EOF
fi

if ! grep -q "curl_server.sh" /data/_root.sh 2>/dev/null; then
    # Add bypass lines near the top of _root.sh
    cat << 'EOF' >> /data/_root.sh

# Dreame region lock bypass
if [ -f /data/scripts/curl_server.sh ]; then
    mount --bind /data/scripts/curl_server.sh /ava/script/curl_server.sh 2>/dev/null
fi
iptables -C OUTPUT -p tcp --dport 15540 -j DROP 2>/dev/null || iptables -I OUTPUT -p tcp --dport 15540 -j DROP 2>/dev/null
iptables -C OUTPUT -p tcp --dport 15541 -j DROP 2>/dev/null || iptables -I OUTPUT -p tcp --dport 15541 -j DROP 2>/dev/null

(
    for _i in $(seq 1 30); do
        if [ -S /var/run/ava_cmd.sock ] || [ -f /data/config/ava/iot_conf.json ]; then
            break
        fi
        sleep 2
    done
    sleep 3
    if [ -f /data/config/ava/iot_conf.json ]; then
        sed -i 's/"LockCode":[0-9]*/"LockCode":200/g' /data/config/ava/iot_conf.json 2>/dev/null
    fi
    avacmd msg_cvt '{"type":"msgCvt","cmd":"device_lock","code":200}' 2>/dev/null
    rm -f /data/log/devlock_record.json 2>/dev/null
) &
EOF
    log_ok "Added region lock bypass to /data/_root.sh"
else
    log_ok "/data/_root.sh already configured"
fi
chmod +x /data/_root.sh

# 8. Ensure boot persistence in /data/_root_postboot.sh (if present)
if [ -f /data/_root_postboot.sh ]; then
    if ! grep -q "curl_server.sh" /data/_root_postboot.sh 2>/dev/null; then
        log_info "Updating /data/_root_postboot.sh..."
        sed -i '/hosts blackhole mounted/a \    if [ -f /data/scripts/curl_server.sh ]; then\n        mount --bind /data/scripts/curl_server.sh /ava/script/curl_server.sh 2>/dev/null\n    fi\n    iptables -C OUTPUT -p tcp --dport 15540 -j DROP 2>/dev/null || iptables -I OUTPUT -p tcp --dport 15540 -j DROP 2>/dev/null\n    iptables -C OUTPUT -p tcp --dport 15541 -j DROP 2>/dev/null || iptables -I OUTPUT -p tcp --dport 15541 -j DROP 2>/dev/null' /data/_root_postboot.sh 2>/dev/null || true
        log_ok "/data/_root_postboot.sh updated"
    else
        log_ok "/data/_root_postboot.sh already configured"
    fi
fi

# 9. Verification test
log_info "Verifying fix..."
/ava/script/curl_server.sh ip_limit >/dev/null 2>&1 || true
sleep 1

LOCK_CODE=$(grep -o '"LockCode":[0-9]*' /data/config/ava/iot_conf.json 2>/dev/null | cut -d: -f2 || echo "unknown")

if [ "$LOCK_CODE" = "200" ]; then
    log_ok "SUCCESS! LockCode is 200 (Device Unlocked)."
else
    log_warn "LockCode is currently '$LOCK_CODE'. Attempting one more unlock signal..."
    avacmd msg_cvt '{"type":"msgCvt","cmd":"device_lock","code":200}' >/dev/null 2>&1 || true
fi

# 10. Test Valetudo if running
if pgrep -f valetudo >/dev/null 2>&1; then
    log_info "Testing Valetudo LocateCapability..."
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT -H 'Content-Type: application/json' -d '{"action":"locate"}' http://127.0.0.1/api/v2/robot/capabilities/LocateCapability 2>/dev/null || echo "failed")
    if [ "$HTTP_CODE" = "200" ]; then
        log_ok "Valetudo LocateCapability responded 200 OK! The robot is fully operational."
    else
        log_info "Valetudo returned HTTP $HTTP_CODE (robot will be ready once Valetudo reconnects)."
    fi
fi

printf "\n${GREEN}===================================================================${NC}\n"
printf "${GREEN}✔ Permanent Region Lock bypass installed successfully!${NC}\n"
printf "${GREEN}The robot will no longer lock after reboot or on network reconnect.${NC}\n"
printf "${GREEN}===================================================================${NC}\n\n"
