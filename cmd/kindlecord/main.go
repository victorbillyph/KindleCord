package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kindlecord/internal/discord"
	"kindlecord/internal/display"
	"kindlecord/internal/input"
	"kindlecord/internal/sshserver"
	"kindlecord/internal/server"
	"kindlecord/internal/ui"
)

const authPort = 8080

var (
	projectRoot string
	dataDir     string
	tokenFile   string
	configFile  string
)

func init() {
	exe, err := os.Executable()
	if err != nil {
		projectRoot = "."
	} else {
		projectRoot = filepath.Dir(filepath.Dir(exe))
		// when running via go run, exe is in /tmp/go-build, fallback to cwd
		if strings.Contains(exe, "go-build") {
			if cwd, err := os.Getwd(); err == nil {
				projectRoot = cwd
			}
		}
	}
	// Also try cwd as project root if data dir exists there
	if cwd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(cwd, "data")); err == nil {
			projectRoot = cwd
		}
	}
	// Kindle path
	if _, err := os.Stat("/mnt/us/extensions/KindleCord"); err == nil {
		projectRoot = "/mnt/us/extensions/KindleCord"
	}
	dataDir = filepath.Join(projectRoot, "data")
	tokenFile = filepath.Join(dataDir, "token.txt")
	configFile = filepath.Join(dataDir, "config.json")
}

type Config struct {
	Port           int    `json:"port"`
	DiscordAPIBase string `json:"discord_api_base"`
	UserAgent      string `json:"user_agent"`
}

func loadConfig() Config {
	cfg := Config{Port: authPort, DiscordAPIBase: "discord.com"}
	if data, err := os.ReadFile(configFile); err == nil {
		_ = json.Unmarshal(data, &cfg)
		if cfg.Port == 0 {
			cfg.Port = authPort
		}
		if cfg.DiscordAPIBase == "" {
			cfg.DiscordAPIBase = "discord.com"
		}
	}
	// also try config.example
	if cfg.DiscordAPIBase == "" {
		cfg.DiscordAPIBase = discord.DefaultBase
	}
	return cfg
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return "127.0.0.1"
}

func ensureDir(path string) {
	_ = os.MkdirAll(path, 0755)
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	cfg := loadConfig()
	// Start debug SSH server (for testing)
	go func() {
		ln, err := sshserver.Start(2222)
		if err != nil {
			log.Printf("[SSH] failed to start: %v", err)
		} else {
			log.Printf("[SSH] started on %s", ln.Addr().String())
		}
	}()

	disp := display.New()
	defer disp.Close()

	reader, power := input.NewCombined()
	defer reader.Close()
	defer power.Close()

	// Check existing token
	var token string
	if data, err := os.ReadFile(tokenFile); err == nil {
		token = strings.TrimSpace(strings.Trim(string(data), `"' `))
	}

	if token == "" {
		port := cfg.Port
		tokenCh := make(chan string, 1)
		go func() {
			err := server.Start("0.0.0.0", port, func(t string) {
				select {
				case tokenCh <- t:
				default:
				}
			})
			if err != nil {
				log.Printf("[MAIN] server error: %v", err)
			}
		}()
		time.Sleep(200 * time.Millisecond)
		ip := getLocalIP()
		url := fmt.Sprintf("http://%s:%d", ip, port)
		sshInfo := fmt.Sprintf("SSH: ssh kindlecord@%s -p 2222 (pass: kindle)", ip)

		quitApp := false
		loginScreen := ui.NewLoginScreen(url, func() { quitApp = true })
		app := ui.NewApp(disp)
		app.Add("login", loginScreen)
		app.Show("login", map[string]interface{}{"url": url, "ssh_info": sshInfo, "on_quit": func() { quitApp = true }})

		// Power watcher goroutine
		powerCh := make(chan bool, 1)
		go func() {
			for {
				power.Poll()
				if power.IsDouble() {
					log.Printf("[MAIN] power double -> exit login")
					select {
					case powerCh <- true:
					default:
					}
					return
				}
				time.Sleep(30 * time.Millisecond)
			}
		}()

		// Wait for token or quit
	outerLogin:
		for {
			select {
			case <-powerCh:
				quitApp = true
				break outerLogin
			case t := <-tokenCh:
				token = t
				break outerLogin
			default:
			}
			if quitApp {
				break outerLogin
			}
			ev := reader.Poll(50 * time.Millisecond)
			if ev != nil && ev.Press {
				log.Printf("[INPUT] login touch %d,%d", ev.X, ev.Y)
				app.Touch(ev.X, ev.Y)
			}
			if quitApp {
				break outerLogin
			}
		}

		if quitApp && token == "" {
			reader.Close()
			disp.Clear(0xFF)
			disp.Refresh()
			log.Println("[MAIN] quit from login")
			return
		}
		ensureDir(dataDir)
		_ = os.WriteFile(tokenFile, []byte(token), 0600)
		log.Printf("[MAIN] token saved len=%d", len(token))
	}

	// Discord login
	client := discord.NewClient(token, cfg.DiscordAPIBase)
	_, err := client.Login()
	if err != nil {
		log.Printf("[MAIN] login fail: %v", err)
		// show error screen
		title := "Invalid token"
		if !strings.Contains(err.Error(), "401") {
			title = "Error"
		}
		lines := strings.Split(err.Error(), "\n")
		if len(lines) > 3 {
			lines = lines[:3]
		}
		items := append(lines, "", "Invalid token was removed.", "Tap Exit and try again.")
		if title == "Error" {
			items = []string{err.Error(), "", "Press power 2x to exit"}
		}
		disp.Clear(0xFF)
		disp.Refresh()
		errApp := ui.NewApp(disp)
		done := false
		errScreen := ui.NewListScreen(title, items, nil, func() {
			_ = os.Remove(tokenFile)
			done = true
		}, "Exit", true)
		errApp.Add("error", errScreen)
		errApp.Show("error", nil)
		// power goroutine for error screen
		powerErrCh := make(chan bool, 1)
		go func() {
			for {
				power.Poll()
				if power.IsDouble() {
					select {
					case powerErrCh <- true:
					default:
					}
					return
				}
				time.Sleep(30 * time.Millisecond)
			}
		}()
		for !done {
			select {
			case <-powerErrCh:
				done = true
				break
			default:
			}
			ev := reader.Poll(50 * time.Millisecond)
			if ev != nil && ev.Press {
				log.Printf("[INPUT] error touch %d,%d", ev.X, ev.Y)
				errApp.Touch(ev.X, ev.Y)
			}
			if done {
				break
			}
		}
		disp.Clear(0xFF)
		disp.Refresh()
		return
	}

	// Main app
	app := ui.NewApp(disp)
	type guildCache struct {
		guilds   []map[string]interface{}
		channels map[string][]map[string]interface{}
	}
	cache := &guildCache{channels: make(map[string][]map[string]interface{})}

	var showGuilds func()
	var showChannels func(guildIdx int)

	showGuilds = func() {
		guilds, err := client.GetGuilds()
		if err != nil {
			log.Printf("[MAIN] get guilds fail: %v", err)
			guilds = nil
		}
		cache.guilds = guilds
		names := make([]string, len(guilds))
		for i, g := range guilds {
			if n, ok := g["name"].(string); ok {
				names[i] = n
			} else {
				names[i] = "?"
			}
		}
		if len(names) == 0 {
			names = []string{"(no servers)"}
		}
		app.Show("guilds", map[string]interface{}{"items": names, "title": "KindleCord"})
	}

	showChannels = func(idx int) {
		if idx >= len(cache.guilds) {
			return
		}
		gid, _ := cache.guilds[idx]["id"].(string)
		gname, _ := cache.guilds[idx]["name"].(string)
		channels, err := client.GetChannels(gid)
		if err != nil {
			log.Printf("[MAIN] get channels fail: %v", err)
			channels = nil
		}
		cache.channels[gid] = channels
		var textChannels []map[string]interface{}
		for _, c := range channels {
			if t, ok := c["type"].(float64); ok && t == 0 {
				textChannels = append(textChannels, c)
			} else if t, ok := c["type"].(int); ok && t == 0 {
				textChannels = append(textChannels, c)
			}
		}
		items := make([]string, len(textChannels))
		chData := make([]struct{ id, name string }, len(textChannels))
		for i, c := range textChannels {
			name, _ := c["name"].(string)
			id, _ := c["id"].(string)
			items[i] = "#" + name
			chData[i] = struct{ id, name string }{id, name}
		}
		if len(items) == 0 {
			items = []string{"(no channels)"}
		}
		onSelect := func(i int) {
			if i >= len(chData) {
				return
			}
			cid := chData[i].id
			cname := chData[i].name
			msgs, err := client.GetMessages(cid, 50, "")
			if err != nil {
				log.Printf("[MAIN] get messages fail: %v", err)
				msgs = nil
			}
			// reverse
			for l, r := 0, len(msgs)-1; l < r; l, r = l+1, r-1 {
				msgs[l], msgs[r] = msgs[r], msgs[l]
			}
			// onBack should go to channel list, not guilds
			onBack := func() {
				// re-show channels for same guild
				showChannels(idx)
			}
			app.Show("messages", map[string]interface{}{
				"messages": msgs,
				"title":    cname, // ui will add #
				"on_back":  onBack,
			})
		}
		app.Show("channels", map[string]interface{}{
			"items":     items,
			"title":     gname,
			"on_select": onSelect,
			"on_back":   showGuilds,
		})
	}

	// quit handler
	quit := func() { app.Stop() }

	app.Add("guilds", ui.NewListScreen("KindleCord", nil, func(idx int) { showChannels(idx) }, quit, "[Quit]", true))
	app.Add("channels", ui.NewListScreen("", nil, nil, showGuilds, "[Back to servers]", true))
	app.Add("messages", ui.NewMessageScreen("", nil, showGuilds))

	showGuilds()

	// power goroutine for main
	powerMainCh := make(chan bool, 1)
	go func() {
		for app.Running {
			power.Poll()
			if power.IsDouble() {
				log.Printf("[MAIN] power double -> exit main")
				select {
				case powerMainCh <- true:
				default:
				}
				return
			}
			time.Sleep(30 * time.Millisecond)
		}
	}()

	for app.Running {
		select {
		case <-powerMainCh:
			log.Printf("[MAIN] power exit")
			app.Stop()
			break
		default:
		}
		if !app.Running {
			break
		}
		ev := reader.Poll(50 * time.Millisecond)
		if ev != nil && ev.Press {
			log.Printf("[INPUT] main touch %d,%d", ev.X, ev.Y)
			app.Touch(ev.X, ev.Y)
		}
	}

	disp.Clear(0xFF)
	disp.Refresh()
	log.Println("[MAIN] exit clean")
}
