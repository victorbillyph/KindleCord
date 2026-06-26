import threading
import socket

try:
    import BaseHTTPServer
    import urlparse
    PY2 = True
except ImportError:
    import http.server as BaseHTTPServer
    import urllib.parse as urlparse
    PY2 = False

PAGE = """<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>KindleCord Login</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;background:#1e1e2e;color:#cdd6f4;display:flex;justify-content:center;align-items:center;min-height:100vh;padding:16px}
.card{background:#313244;padding:32px;border-radius:12px;width:100%;max-width:400px}
h1{font-size:24px;margin-bottom:8px;color:#cba6f7}
p{margin-bottom:16px;color:#a6adc8;font-size:14px}
input{width:100%;padding:12px;font-size:14px;border:2px solid #45475a;border-radius:8px;background:#1e1e2e;color:#cdd6f4;margin-bottom:12px;outline:none}
input:focus{border-color:#cba6f7}
button{width:100%;padding:12px;font-size:16px;font-weight:600;background:#cba6f7;color:#1e1e2e;border:none;border-radius:8px;cursor:pointer}
button:hover{background:#b4befe}
.hint{font-size:12px;color:#585b70;margin-top:12px}
code{background:#1e1e2e;padding:2px 6px;border-radius:4px;font-size:12px;color:#a6e3a1}
</style>
</head>
<body>
<div class="card">
<h1>KindleCord</h1>
<p>Paste your Discord token to log in on your Kindle.</p>
<form method="POST" action="/token">
<input type="text" name="token" placeholder="Discord token" required autocomplete="off">
<button type="submit">Log in</button>
</form>
<div class="hint">
How to get your token:<br>
1. Open Discord in a browser<br>
2. Press <code>Ctrl+Shift+I</code><br>
3. Go to Console<br>
4. Type: <code>localStorage.getItem('token')</code>
</div>
</div>
</body>
</html>"""

SUCCESS_PAGE = """<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>KindleCord</title></head>
<body style="background:#1e1e2e;color:#cdd6f4;display:flex;justify-content:center;align-items:center;min-height:100vh;font-family:sans-serif">
<div style="text-align:center"><h1 style="color:#a6e3a1">Token saved!</h1><p>You can close this page.</p></div>
</body>
</html>"""


class _Handler(BaseHTTPServer.BaseHTTPRequestHandler):
    cb = None

    def do_GET(self):
        if self.path == "/":
            self._send(200, "text/html", PAGE.encode())
        elif self.path == "/favicon.ico":
            self._send(204, "text/plain", b"")
        else:
            self._send(404, "text/plain", b"Not found")

    def do_POST(self):
        if self.path == "/token":
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length)
            if PY2:
                body = body.decode("utf-8")
            params = urlparse.parse_qs(body)
            token = params.get("token", [""])[0].strip()
            if token:
                if self.cb:
                    self.cb(token)
                self._send(200, "text/html", SUCCESS_PAGE.encode())
            else:
                self._send(400, "text/plain", b"Missing token")
        else:
            self._send(404, "text/plain", b"Not found")

    def _send(self, code, ctype, data):
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Connection", "close")
        self.end_headers()
        try:
            self.wfile.write(data)
            self.wfile.flush()
        except (IOError, OSError):
            pass

    def log_message(self, *args):
        pass


def start_server(host, port, callback):
    _Handler.cb = callback
    try:
        server = BaseHTTPServer.HTTPServer((host, port), _Handler)
        server.timeout = 1
        t = threading.Thread(target=server.serve_forever)
        t.daemon = True
        t.start()
        return server
    except (socket.error, IOError, OSError) as e:
        print("[SERVER] Failed to start: %s" % e)
        return None
