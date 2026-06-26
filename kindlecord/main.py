import os
import socket
import sys
import time
import json
import threading

from kindlecord.display import Display
from kindlecord.input import InputReader, PowerWatcher
from kindlecord.server import start_server
from kindlecord.discord import DiscordClient, DiscordError
from kindlecord.ui import App95, LoginScreen95, ListScreen95, MessageScreen95, Dialog95

try:
    input = raw_input
except NameError:
    pass

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DATA_DIR = os.path.join(PROJECT_ROOT, "data")
TOKEN_FILE = os.path.join(DATA_DIR, "token.txt")
CONFIG_FILE = os.path.join(DATA_DIR, "config.json")
AUTH_PORT = 8080


def _load_config():
    config = {
        "port": AUTH_PORT,
        "discord_api_base": "https://discord.com/api/v10",
    }
    if os.path.exists(CONFIG_FILE):
        with open(CONFIG_FILE) as f:
            config.update(json.load(f))
    return config


def _get_local_ip():
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try:
        s.connect(("8.8.8.8", 80))
        return s.getsockname()[0]
    except OSError:
        return "127.0.0.1"
    finally:
        s.close()


def _ensure_dir(path):
    try:
        os.makedirs(path)
    except OSError:
        if not os.path.isdir(path):
            raise


def run():
    config = _load_config()
    display = Display()
    inp = InputReader()
    pw = PowerWatcher()

    # Check for existing token
    token = None
    if os.path.exists(TOKEN_FILE):
        with open(TOKEN_FILE) as f:
            token = f.read().strip()

    if not token:
        # Start web server for token entry
        port = config.get("port", AUTH_PORT)
        token_value = [None]
        token_event = threading.Event()
        server_ready = threading.Event()

        def _on_token(t):
            token_value[0] = t
            token_event.set()

        ok = start_server("0.0.0.0", port, _on_token, ready_event=server_ready)
        ip = _get_local_ip()
        url = "http://{0}:{1}".format(ip, port)

        if not ok:
            url = "http://{0}:{1} (FAILED)".format(ip, port)

        # Show login screen with quit button
        login_screen = LoginScreen95(url)
        quit_app = [False]

        def _quit():
            quit_app[0] = True

        login_screen._on_quit = _quit

        app = App95(display, None)
        app.add("login", login_screen)
        app.show("login")

        # Wait for token or quit
        while token_value[0] is None and not quit_app[0]:
            pw.poll()
            if pw.is_double():
                quit_app[0] = True
                break
            ev = inp.poll(timeout=0.2)
            if ev:
                login_screen.on_touch(ev.x, ev.y)

        if quit_app[0]:
            inp.close()
            display.clear()
            return

        token = token_value[0]
        _ensure_dir(DATA_DIR)
        with open(TOKEN_FILE, "w") as f:
            f.write(token)

    # Log in to Discord
    discord = DiscordClient(
        token,
        config.get("discord_api_base", "https://discord.com/api/v10"),
    )

    try:
        user = discord.login()
        items = None
    except DiscordError as e:
        title = "Login Error"
        err_lines = str(e).split("\n")[:5]
        items = err_lines + ["", "Delete token.txt", "Press power 2x to exit"]
    except Exception as e:
        title = "Error"
        items = [str(e), "", "Press power 2x to exit"]

    if items is not None:
        display.clear()
        err_app = App95(display, discord)
        err_app.add("error", ListScreen95(title, items,
                     show_title_bar=True, back_label="Exit"))
        done = [False]
        def _done():
            done[0] = True
        err_app.screens["error"]._on_back = _done
        err_app.show("error")
        while not done[0]:
            pw.poll()
            if pw.is_double():
                break
            ev = inp.poll(timeout=0.5)
            if ev:
                err_app.touch(ev.x, ev.y)
        inp.close()
        return

    # Main app with Discord
    app = App95(display, discord)
    guilds_cache = {"guilds": [], "channels": {}}

    def quit_app():
        app.stop()

    def show_guilds():
        try:
            guilds_cache["guilds"] = discord.get_guilds()
        except DiscordError:
            guilds_cache["guilds"] = []
        gnames = [g.get("name", "?") for g in guilds_cache["guilds"]]
        app.show("guilds", items=gnames)

    def _on_guild_select(idx):
        guilds = guilds_cache["guilds"]
        if idx >= len(guilds):
            return
        gid = guilds[idx]["id"]
        try:
            channels = discord.get_channels(gid)
        except DiscordError:
            channels = []
        guilds_cache["channels"][gid] = channels
        text_channels = [c for c in channels if c.get("type") == 0]
        items = ["#{0}".format(c.get("name", "?")) for c in text_channels]
        ch_data = [(c["id"], c.get("name", "?")) for c in text_channels]

        def sel(i):
            cid, cname = ch_data[i]
            _on_channel(cid, cname)

        gname = guilds[idx].get("name", "?")
        app.show("channels", items=items, title=gname,
                 on_select=sel, on_back=show_guilds)

    def _on_channel(cid, cname):
        try:
            msgs = discord.get_messages(cid, limit=50)
        except DiscordError:
            msgs = []
        msgs.reverse()
        app.show("messages", messages=msgs,
                 title="#{0}".format(cname),
                 on_back=show_guilds)

    def _on_back_to_main():
        show_guilds()

    app.add("guilds", ListScreen95("KindleCord", [],
             on_select=_on_guild_select, on_back=quit_app,
             back_label="[Quit]"))
    app.add("channels", ListScreen95("", [],
             back_label="[Back to servers]"))
    app.add("messages", MessageScreen95(""))

    show_guilds()

    try:
        while app.running:
            pw.poll()
            if pw.is_double():
                break
            ev = inp.poll(timeout=0.5)
            if ev:
                app.touch(ev.x, ev.y)
    except KeyboardInterrupt:
        pass
    finally:
        pw.close()
        inp.close()
        display.clear()
