package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const page = `<!DOCTYPE html><html><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
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
</style></head><body>
<div class="card"><h1>KindleCord</h1>
<p>Paste your Discord token to log in on your Kindle.</p>
<form method="POST" action="/">
<input type="text" name="token" placeholder="Token do Discord" required autocomplete="off">
<button type="submit">Log in</button></form>
<div class="hint">
How to get your token:<br>
1. Open Discord in a browser<br>
2. Press Ctrl+Shift+I<br>
3. Go to Console<br>
4. Type: localStorage.getItem('token')</div></div></body></html>`

const successPage = `<!DOCTYPE html><html><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>KindleCord</title></head>
<body style="background:#1e1e2e;color:#cdd6f4;display:flex;justify-content:center;align-items:center;min-height:100vh;font-family:sans-serif">
<div style="text-align:center">
<h1 style="color:#a6e3a1">Token saved!</h1>
<p>You can close this page.</p></div></body></html>`

// Start launches HTTP server and calls onToken when received. Returns error if bind fails.
func Start(host string, port int, onToken func(string)) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.URL.Path == "/favicon.ico" {
				w.WriteHeader(204)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(page))
			return
		}
		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Bad request", 400)
				return
			}
			token := r.FormValue("token")
			// also try raw body parse for compat
			if token == "" {
				// fallback manual parse
				// r.Form already handled
			}
			token = strings.TrimSpace(strings.Trim(token, `"' `))
			// url decode already done by ParseForm, but handle double encoded
			if decoded, err := url.QueryUnescape(token); err == nil {
				// only if it looks encoded
				if decoded != token && strings.Contains(token, "%") {
					token = decoded
				}
			}
			if token == "" {
				log.Printf("[SERVER] missing token")
				http.Error(w, "Missing token", 400)
				return
			}
			log.Printf("[SERVER] token received len=%d", len(token))
			if onToken != nil {
				onToken(token)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(successPage))
			return
		}
		http.Error(w, "Method not allowed", 405)
	})
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[SERVER] listening on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux}
	return srv.ListenAndServe()
}
