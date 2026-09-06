package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const page = `<!DOCTYPE html><html><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>KindleCord Setup</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;background:#1e1e2e;color:#cdd6f4;min-height:100vh;padding:24px}
.container{max-width:520px;margin:0 auto}
.card{background:#313244;padding:24px;border-radius:12px;margin-bottom:16px}
h1{font-size:22px;margin-bottom:8px;color:#cba6f7}
h2{font-size:16px;margin:16px 0 8px;color:#a6e3a1}
p{margin-bottom:12px;color:#a6adc8;font-size:14px;line-height:1.5}
input,textarea{width:100%;padding:12px;font-size:14px;border:2px solid #45475a;border-radius:8px;background:#1e1e2e;color:#cdd6f4;margin-bottom:12px;outline:none;font-family:inherit}
input:focus,textarea:focus{border-color:#cba6f7}
button{width:100%;padding:12px;font-size:15px;font-weight:600;background:#cba6f7;color:#1e1e2e;border:none;border-radius:8px;cursor:pointer}
button:hover{background:#b4befe}
button:disabled{background:#585b70;cursor:not-allowed}
.secondary{background:#45475a;color:#cdd6f4}
.secondary:hover{background:#585b70}
.hint{font-size:12px;color:#585b70;margin-top:8px}
.code{background:#1e1e2e;padding:12px;border-radius:8px;font-family:monospace;font-size:13px;color:#f9e2af;overflow-x:auto;margin:8px 0;white-space:pre-wrap}
.bookmarklet-link{display:inline-block;padding:10px 16px;background:#cba6f7;color:#1e1e2e;text-decoration:none;border-radius:8px;font-weight:600;font-size:14px;margin:8px 0}
.bookmarklet-link:hover{background:#b4befe}
.status{padding:12px;border-radius:8px;margin:12px 0;font-size:14px}
.status.success{background:#1e3a2e;color:#a6e3a1;border:1px solid #a6e3a1}
.status.error{background:#3a1e2e;color:#f38ba8;border:1px solid #f38ba8}
.status.waiting{background:#2e2e3e;color:#f9e2af;border:1px solid #f9e2af}
.step{margin-bottom:20px;padding-bottom:20px;border-bottom:1px solid #45475a}
.step:last-child{border-bottom:none;margin-bottom:0;padding-bottom:0}
.badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:11px;font-weight:600;background:#cba6f7;color:#1e1e2e;margin-right:8px}
</style></head><body>
<div class="container">
<div class="card">
<h1>KindleCord Setup</h1>
<p>Open this page on your computer or phone (same Wi-Fi as your Kindle) to send your Discord token.</p>
<div class="status waiting" id="status">Waiting for token...</div>
</div>

<div class="card step">
<span class="badge">1</span>
<h2>Automatic: Use the bookmarklet (easiest)</h2>
<p>Drag this button to your browser's bookmarks bar:</p>
<a class="bookmarklet-link" id="bookmarklet" href="#" draggable="true" title="Drag to bookmarks bar">KindleCord: Send Token</a>
<p class="hint">Then go to <strong>discord.com</strong>, click the bookmarklet — it grabs your token and sends it here automatically.</p>
</div>

<div class="card step">
<span class="badge">2</span>
<h2>Manual: Paste your token</h2>
<form id="tokenForm" method="POST" action="/">
<textarea name="token" placeholder="Paste your Discord token here" rows="3" required autocomplete="off"></textarea>
<button type="submit">Send to Kindle</button>
</form>
</div>

<div class="card step">
<span class="badge">3</span>
<h2>How to get the token manually (if needed)</h2>
<div class="code">localStorage.getItem('token')</div>
<p class="hint">1. Open Discord in browser → press <kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>I</kbd> (or F12)<br>
2. Click <strong>Console</strong> tab → paste the command above → press Enter<br>
3. Copy the result (long string in quotes) → paste in the box above</p>
</div>

<div class="card">
<p class="hint"><strong>Security:</strong> Your token never leaves your local network. This page talks directly to your Kindle.</p>
</div>
</div>

<script>
const km = location.origin;
const bm = document.getElementById('bookmarklet');
const form = document.getElementById('tokenForm');
const statusEl = document.getElementById('status');

// Bookmarklet code - extracts token and POSTs to Kindle
const bookmarkletCode = (function() {
    var t = localStorage.getItem('token');
    if (!t) { alert('Token not found. Open discord.com first.'); return; }
    fetch(km + '/api/token', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({token: t}),
        mode: 'cors'
    }).then(function(r) {
        if (r.ok) {
            document.body.innerHTML = '<div style="padding:40px;text-align:center;color:#a6e3a1;font-family:sans-serif"><h1>Token sent!</h1><p>Check your Kindle.</p></div>';
        } else {
            alert('Failed: ' + r.status);
        }
    }).catch(function(e) { alert('Error: ' + e); });
}).toString().replace(/^function\s*\(\)\s*\{|\}$/g, '').trim();

bm.href = 'javascript:' + encodeURIComponent('(' + bookmarkletCode + ')();');
bm.ondragstart = function(e) { e.dataTransfer.setData('text/plain', bm.href); };

// Manual form submit via fetch for better UX
form.onsubmit = function(e) {
    e.preventDefault();
    var token = form.token.value.trim().replace(/^["']|["']$/g, '');
    if (!token) { setStatus('Enter a token first', 'error'); return; }
    sendToken(token);
};

function sendToken(token) {
    setStatus('Sending token to Kindle...', 'waiting');
    fetch(km + '/api/token', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({token: token}),
        mode: 'cors'
    }).then(function(r) {
        if (r.ok) {
            setStatus('Token sent! Check your Kindle.', 'success');
            form.reset();
        } else {
            r.text().then(function(t) { setStatus('Failed: ' + t, 'error'); });
        }
    }).catch(function(e) { setStatus('Error: ' + e, 'error'); });
}

function setStatus(msg, type) {
    statusEl.textContent = msg;
    statusEl.className = 'status ' + type;
}

// Auto-check: if token already sent, show success
window.addEventListener('load', function() {
    // Could poll for status but simple is fine
});
</script>
</body></html>`

const successPage = `<!DOCTYPE html><html><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>KindleCord</title></head>
<body style="background:#1e1e2e;color:#cdd6f4;display:flex;justify-content:center;align-items:center;min-height:100vh;font-family:sans-serif">
<div style="text-align:center">
<h1 style="color:#a6e3a1">Token saved!</h1>
<p>You can close this page.</p></div></body></html>`

// tokenResponse is JSON response for /api/token
type tokenResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func corsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	corsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Start launches HTTP server and calls onToken when received. Returns error if bind fails.
func Start(host string, port int, onToken func(string)) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w)

		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}

		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(204)
			return
		}

		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(page))
			return
		}

		if r.Method == "POST" {
			// Accept both form and JSON
			var token string
			ct := r.Header.Get("Content-Type")
			if strings.HasPrefix(ct, "application/json") {
				var req struct {
					Token string `json:"token"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					token = req.Token
				}
			} else {
				if err := r.ParseForm(); err == nil {
					token = r.FormValue("token")
				}
			}

			token = strings.TrimSpace(strings.Trim(token, `"' `))
			if decoded, err := url.QueryUnescape(token); err == nil {
				if decoded != token && strings.Contains(token, "%") {
					token = decoded
				}
			}

			if token == "" {
				log.Printf("[SERVER] missing token")
				writeJSON(w, 400, tokenResponse{false, "Missing token"})
				return
			}

			log.Printf("[SERVER] token received len=%d", len(token))
			if onToken != nil {
				onToken(token)
			}

			// Return JSON for API calls, HTML for form submit
			if strings.HasPrefix(ct, "application/json") {
				writeJSON(w, 200, tokenResponse{true, "Token saved"})
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(successPage))
			}
			return
		}

		http.Error(w, "Method not allowed", 405)
	})

	// Dedicated API endpoint (same handler but explicit)
	mux.HandleFunc("/api/token", func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		if r.Method != "POST" {
			writeJSON(w, 405, tokenResponse{false, "Method not allowed"})
			return
		}

		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, tokenResponse{false, "Invalid JSON"})
			return
		}

		token := strings.TrimSpace(strings.Trim(req.Token, `"' `))
		if token == "" {
			writeJSON(w, 400, tokenResponse{false, "Missing token"})
			return
		}

		log.Printf("[SERVER] token received via /api/token len=%d", len(token))
		if onToken != nil {
			onToken(token)
		}
		writeJSON(w, 200, tokenResponse{true, "Token saved"})
	})

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[SERVER] listening on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux}
	return srv.ListenAndServe()
}
