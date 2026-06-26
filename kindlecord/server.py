import threading
import socket
import time

try:
    import urlparse
    PY2 = True
except ImportError:
    import urllib.parse as urlparse
    PY2 = False

PAGE = (
    '<!DOCTYPE html><html><head>'
    '<meta charset="UTF-8"><meta name="viewport" '
    'content="width=device-width,initial-scale=1.0">'
    '<title>KindleCord Login</title>'
    '<style>'
    '*{box-sizing:border-box;margin:0;padding:0}'
    'body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;'
    'background:#1e1e2e;color:#cdd6f4;display:flex;'
    'justify-content:center;align-items:center;min-height:100vh;padding:16px}'
    '.card{background:#313244;padding:32px;border-radius:12px;'
    'width:100%;max-width:400px}'
    'h1{font-size:24px;margin-bottom:8px;color:#cba6f7}'
    'p{margin-bottom:16px;color:#a6adc8;font-size:14px}'
    'input{width:100%;padding:12px;font-size:14px;'
    'border:2px solid #45475a;border-radius:8px;background:#1e1e2e;'
    'color:#cdd6f4;margin-bottom:12px;outline:none}'
    'input:focus{border-color:#cba6f7}'
    'button{width:100%;padding:12px;font-size:16px;font-weight:600;'
    'background:#cba6f7;color:#1e1e2e;border:none;border-radius:8px;'
    'cursor:pointer}'
    'button:hover{background:#b4befe}'
    '.hint{font-size:12px;color:#585b70;margin-top:12px}'
    'code{background:#1e1e2e;padding:2px 6px;border-radius:4px;'
    'font-size:12px;color:#a6e3a1}'
    '</style></head><body>'
    '<div class="card"><h1>KindleCord</h1>'
    '<p>Paste your Discord token to log in on your Kindle.</p>'
    '<form method="POST" action="/">'
    '<input type="text" name="token" placeholder="Token do Discord" '
    'required autocomplete="off">'
    '<button type="submit">Log in</button></form>'
    '<div class="hint">'
    'How to get your token:<br>'
    '1. Open Discord in a browser<br>'
    '2. Press Ctrl+Shift+I<br>'
    '3. Go to Console<br>'
    '4. Type: localStorage.getItem(\'token\')</div></div></body></html>'
)

SUCCESS_PAGE = (
    '<!DOCTYPE html><html><head>'
    '<meta charset="UTF-8"><meta name="viewport" '
    'content="width=device-width,initial-scale=1.0">'
    '<title>KindleCord</title></head>'
    '<body style="background:#1e1e2e;color:#cdd6f4;display:flex;'
    'justify-content:center;align-items:center;min-height:100vh;'
    'font-family:sans-serif">'
    '<div style="text-align:center">'
    '<h1 style="color:#a6e3a1">Token saved!</h1>'
    '<p>You can close this page.</p></div></body></html>'
)


def _recv_until(sock, delim):
    """Read from socket until delimiter is found, return data + rest."""
    data = b""
    while True:
        chunk = sock.recv(4096)
        if not chunk:
            break
        data += chunk
        if delim in data:
            break
    return data


def _parse_headers(data):
    """Parse HTTP headers from bytes, return (method, path, headers_dict, body_start)."""
    try:
        if PY2:
            decoded = data.decode("utf-8")
        else:
            decoded = data.decode("utf-8")
    except Exception:
        return None, None, None, None

    # Split into lines
    lines = decoded.split("\r\n")
    if not lines:
        return None, None, None, None

    # Request line: METHOD /path HTTP/1.1
    req_line = lines[0]
    parts = req_line.split(" ")
    if len(parts) < 2:
        return None, None, None, None
    method = parts[0].upper()
    path = parts[1]

    # Headers
    headers = {}
    i = 1
    while i < len(lines):
        line = lines[i]
        if line == "":
            i += 1
            break
        if ":" in line:
            key, val = line.split(":", 1)
            headers[key.strip().lower()] = val.strip()
        i += 1

    # Body starts after blank line
    body_start = data.find(b"\r\n\r\n")
    if body_start == -1:
        body = b""
    else:
        body_start += 4  # skip \r\n\r\n
        body = data[body_start:]

    return method, path, headers, body


def _handle_client(conn, addr, callback):
    """Handle a single HTTP client connection."""
    try:
        conn.settimeout(5.0)

        # Read request (headers + maybe partial body)
        data = _recv_until(conn, b"\r\n\r\n")
        if not data:
            conn.close()
            return

        method, path, headers, partial_body = _parse_headers(data)
        if method is None:
            conn.close()
            return

        # Read full body if Content-Length is specified
        body = partial_body
        cl = headers.get("content-length", "0")
        try:
            content_length = int(cl)
        except (ValueError, TypeError):
            content_length = 0

        while len(body) < content_length:
            chunk = conn.recv(4096)
            if not chunk:
                break
            body += chunk

        # Build response
        if method == "GET":
            if path in ("/", "/token", "/api"):
                resp_body = PAGE
                code = 200
                ctype = "text/html"
            elif path == "/favicon.ico":
                resp_body = ""
                code = 204
                ctype = "text/plain"
            else:
                resp_body = "Not found"
                code = 404
                ctype = "text/plain"

            if PY2:
                resp_bytes = resp_body
            else:
                resp_bytes = resp_body.encode("utf-8")
            _send_response(conn, code, ctype, resp_bytes)

        elif method == "POST":
            if PY2:
                body_str = body.decode("utf-8")
            else:
                body_str = body.decode("utf-8")
            params = urlparse.parse_qs(body_str)
            token = params.get("token", [""])[0].strip().strip('"').strip("'").strip()

            if token:
                print("[SERVER] token received, len=%d" % len(token))
                if callback:
                    callback(token)
                resp_body = SUCCESS_PAGE
                code = 200
                ctype = "text/html"
            else:
                print("[SERVER] missing token (body=%r)" % body_str)
                resp_body = "Missing token"
                code = 400
                ctype = "text/plain"

            if PY2:
                resp_bytes = resp_body
            else:
                resp_bytes = resp_body.encode("utf-8")
            _send_response(conn, code, ctype, resp_bytes)

        else:
            _send_response(conn, 405, "text/plain", b"Method not allowed")

    except socket.timeout:
        pass
    except Exception as e:
        print("[SERVER] error: %s" % e)
    finally:
        try:
            conn.close()
        except Exception:
            pass


def _send_response(conn, code, ctype, body):
    """Send HTTP response and close connection."""
    reason = {200: "OK", 204: "No Content", 400: "Bad Request",
              404: "Not Found", 405: "Method Not Allowed"}.get(code, "Unknown")
    status_line = "HTTP/1.1 %d %s\r\n" % (code, reason)
    headers = (
        "Content-Type: %s\r\n" % ctype
        + "Content-Length: %d\r\n" % len(body)
        + "Connection: close\r\n"
        + "\r\n"
    )
    if PY2:
        response = status_line + headers + body
    else:
        response = status_line.encode() + headers.encode() + body
    try:
        conn.sendall(response)
    except Exception as e:
        print("[SERVER] send error: %s" % e)


def _server_loop(host, port, callback, ready_event):
    """Accept connections in a loop."""
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    try:
        s.bind((host, port))
        s.listen(5)
        s.settimeout(1.0)
        print("[SERVER] listening on %s:%d" % (host, port))
        if ready_event is not None:
            ready_event.set()
        while True:
            try:
                conn, addr = s.accept()
                print("[SERVER] connection from %s" % (addr,))
                _handle_client(conn, addr, callback)
            except socket.timeout:
                continue
            except Exception as e:
                print("[SERVER] accept error: %s" % e)
    except Exception as e:
        print("[SERVER] bind error: %s" % e)
        if ready_event is not None:
            ready_event.set()
    finally:
        try:
            s.close()
        except Exception:
            pass


def start_server(host, port, callback, ready_event=None):
    t = threading.Thread(target=_server_loop,
                         args=(host, port, callback, ready_event))
    t.daemon = True
    t.start()
    return True
