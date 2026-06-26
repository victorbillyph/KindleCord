# -*- coding: utf-8 -*-
import json
import ssl

try:
    import urllib2
    import urllib
    PY2 = True
except ImportError:
    import urllib.request as urllib2
    import urllib.parse as urllib
    PY2 = False

# Disable SSL verification — Kindle's CA store is outdated
try:
    ssl._create_default_https_context = ssl._create_unverified_context
except AttributeError:
    pass

API_BASE = "https://discord.com/api/v10"


class DiscordError(Exception):
    pass


class DiscordClient:
    def __init__(self, token, api_base=API_BASE):
        self.token = token
        self.api_base = api_base
        self.user = None

    def _headers(self):
        return {
            "Authorization": self.token,
            "Content-Type": "application/json",
            "User-Agent": "KindleCord/0.1.0",
        }

    def _request(self, method, path, data=None):
        url = self.api_base + path
        body = json.dumps(data) if data is not None else None
        if not PY2 and body is not None:
            body = body.encode()
        req = urllib2.Request(url, data=body, headers=self._headers())
        if PY2:
            req.get_method = lambda: method
        else:
            req.method = method
        try:
            with urllib2.urlopen(req) as resp:
                return json.loads(resp.read())
        except urllib2.HTTPError as e:
            resp_body = e.read()
            raise DiscordError("HTTP %d: %s" % (e.code, resp_body))
        except urllib2.URLError as e:
            reason = e.reason if PY2 else e.reason
            raise DiscordError("Network error: %s" % reason)

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
