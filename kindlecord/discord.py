# -*- coding: utf-8 -*-
import json
import ssl

try:
    import httplib
    import urllib
    PY2 = True
except ImportError:
    import http.client as httplib
    import urllib.parse as urllib
    PY2 = False

# Disable SSL verification — Kindle's CA store is outdated
try:
    ssl._create_default_https_context = ssl._create_unverified_context
except AttributeError:
    pass

API_BASE = "discord.com"
API_VERSION = "/api/v10"


class DiscordError(Exception):
    pass


class DiscordClient:
    def __init__(self, token, api_base=API_BASE):
        self.token = str(token).strip()
        self.api_base = api_base
        self.user = None
        print("[DISCORD] token len=%d, first=%.10s" % (len(self.token), self.token[:10]))

    def _request(self, method, path, data=None):
        url = API_VERSION + path
        body = json.dumps(data) if data is not None else None
        if PY2 and body is not None:
            body = body.encode("utf-8")
        if not PY2 and body is not None:
            body = body.encode("utf-8")

        # Bot tokens need "Bot " prefix; user tokens don't
        prefix = "Bot " if not self.token.startswith(("Bot ", "Bearer ")) else ""
        headers = {
            "Authorization": prefix + self.token,
            "Content-Type": "application/json",
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
            "Host": self.api_base,
        }

        print("[DISCORD] %s %s" % (method, url))
        try:
            if PY2:
                ctx = ssl._create_unverified_context()
                conn = httplib.HTTPSConnection(self.api_base, context=ctx, timeout=30)
            else:
                conn = httplib.HTTPSConnection(self.api_base, timeout=30)
            conn.request(method, url, body=body, headers=headers)
            resp = conn.getresponse()
            resp_body = resp.read()
            print("[DISCORD] response: %d" % resp.status)
            if PY2:
                resp_body_str = resp_body
            else:
                resp_body_str = resp_body.decode("utf-8")
            if resp.status >= 400:
                raise DiscordError("HTTP %d: %s" % (resp.status, resp_body_str))
            return json.loads(resp_body_str)
        except DiscordError:
            raise
        except Exception as e:
            raise DiscordError("Network error: %s" % e)

    def login(self):
        self.user = self._request("GET", "/users/@me")
        return self.user

    def get_user(self, user_id):
        return self._request("GET", "/users/{0}".format(user_id))

    def get_guilds(self):
        return self._request("GET", "/users/@me/guilds")

    def get_guild(self, guild_id):
        return self._request("GET", "/guilds/{0}".format(guild_id))

    def get_channels(self, guild_id):
        return self._request("GET", "/guilds/{0}/channels".format(guild_id))

    def get_messages(self, channel_id, limit=50, before=None):
        path = "/channels/{0}/messages?limit={1}".format(channel_id, limit)
        if before:
            path += "&before={0}".format(before)
        return self._request("GET", path)

    def send_message(self, channel_id, content):
        return self._request("POST",
            "/channels/{0}/messages".format(channel_id),
            {"content": content})

    def edit_message(self, channel_id, message_id, content):
        return self._request("PATCH",
            "/channels/{0}/messages/{1}".format(channel_id, message_id),
            {"content": content})

    def delete_message(self, channel_id, message_id):
        self._request("DELETE",
            "/channels/{0}/messages/{1}".format(channel_id, message_id))

    def add_reaction(self, channel_id, message_id, emoji):
        enc = urllib.quote(emoji, safe="")
        self._request("PUT",
            "/channels/{0}/messages/{1}/reactions/{2}/@me".format(
                channel_id, message_id, enc))

    def remove_reaction(self, channel_id, message_id, emoji):
        enc = urllib.quote(emoji, safe="")
        self._request("DELETE",
            "/channels/{0}/messages/{1}/reactions/{2}/@me".format(
                channel_id, message_id, enc))

    def create_dm(self, user_id):
        return self._request("POST", "/users/@me/channels",
                             {"recipient_id": user_id})

    def get_dms(self):
        return self._request("GET", "/users/@me/channels")
