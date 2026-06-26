import struct
import select
import os
import sys
import time

EV_ABS = 3
EV_KEY = 1
ABS_MT_POSITION_X = 53
ABS_MT_POSITION_Y = 54
BTN_TOUCH = 330
KEY_POWER = 116
KEY_SLEEP = 142

INPUT_EVENT_FORMAT = "llHHI"
INPUT_EVENT_SIZE = struct.calcsize(INPUT_EVENT_FORMAT)


class TouchEvent:
    PRESS = 1
    RELEASE = 0

    def __init__(self, x, y, action):
        self.x = x
        self.y = y
        self.action = action


class InputReader:
    def __init__(self, device_paths=None):
        if device_paths is None:
            device_paths = ["/dev/input/event1", "/dev/input/event0"]
        self.device = None
        self._simulate = True
        for path in device_paths:
            if os.path.exists(path):
                try:
                    self.device = open(path, "rb")
                    self._simulate = False
                    break
                except (IOError, OSError):
                    pass
        self._x = 0
        self._y = 0
        if self._simulate:
            print("[Input SIM] No input device found, touch disabled")

    def poll(self, timeout=0.1):
        if self._simulate:
            return None
        r, _, _ = select.select([self.device], [], [], timeout)
        if not r:
            return None
        data = self.device.read(INPUT_EVENT_SIZE)
        if len(data) < INPUT_EVENT_SIZE:
            return None
        _, _, ev_type, code, value = struct.unpack(INPUT_EVENT_FORMAT, data)

        if ev_type == EV_ABS:
            if code == ABS_MT_POSITION_X:
                self._x = value
            elif code == ABS_MT_POSITION_Y:
                self._y = value
        elif ev_type == EV_KEY and code == BTN_TOUCH:
            return TouchEvent(self._x, self._y,
                              TouchEvent.PRESS if value else TouchEvent.RELEASE)
        return None

    def close(self):
        if self.device:
            self.device.close()


class PowerWatcher:
    """Watches /dev/input/event0 for double-press of the power button."""
    DOUBLE_WINDOW = 0.5  # seconds

    def __init__(self, path="/dev/input/event0"):
        self._fd = None
        self._last = 0
        self._double = False
        if os.path.exists(path):
            try:
                self._fd = open(path, "rb")
            except (IOError, OSError):
                pass

    def poll(self, timeout=0):
        if not self._fd:
            return
        r, _, _ = select.select([self._fd], [], [], timeout)
        if not r:
            return
        try:
            while True:
                data = self._fd.read(INPUT_EVENT_SIZE)
                if len(data) < INPUT_EVENT_SIZE:
                    break
                _, _, ev_type, code, value = struct.unpack(INPUT_EVENT_FORMAT, data)
                if ev_type == EV_KEY and value == 1 and code in (KEY_POWER, KEY_SLEEP):
                    now = time.time()
                    if now - self._last < self.DOUBLE_WINDOW:
                        self._double = True
                    self._last = now
        except Exception:
            pass

    def is_double(self):
        if self._double:
            self._double = False
            return True
        return False

    def close(self):
        if self._fd:
            self._fd.close()
