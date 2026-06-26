#!/bin/sh
# KindleCord - Discord client for Kindle

if [ "$(dirname "${0}")" != "/var/tmp" ]; then
    cp -pf "${0}" /var/tmp/kindlecord.sh
    chmod 777 /var/tmp/kindlecord.sh
    exec /var/tmp/kindlecord.sh "$@"
fi

export LC_ALL="en_US.UTF-8"
export PYTHONUNBUFFERED=1
EXT_DIR="/mnt/us/extensions/KindleCord"
LOG="$EXT_DIR/kindlecord.log"
cd "$EXT_DIR" || exit 1

echo "=== KindleCord $(date) ===" > "$LOG"

cleanup() {
    echo "Cleanup..." >> "$LOG"
    killall -CONT awesome 2>> "$LOG"
    lipc-set-prop com.lab126.pillow disableEnablePillow enable 2>> "$LOG"
    eips -f
}
trap cleanup EXIT INT TERM

echo "Disabling pillow..." >> "$LOG"
lipc-set-prop com.lab126.pillow disableEnablePillow disable 2>> "$LOG"
usleep 250000

echo "Stopping awesome..." >> "$LOG"
killall -STOP awesome 2>> "$LOG"
usleep 250000

for cmd in python3 python2.7 "/mnt/us/python/bin/python2.7" python; do
    if command -v "$cmd" >/dev/null 2>&1; then
        echo "Using: $cmd" >> "$LOG"
        "$cmd" -m kindlecord >> "$LOG" 2>&1
        EXIT=$?
        echo "Exit code: $EXIT" >> "$LOG"
        break
    fi
done
