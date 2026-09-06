package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kindlecord/internal/discord"
	"kindlecord/internal/display"
	"kindlecord/internal/input"
	"kindlecord/internal/server"
	"kindlecord/internal/sshserver"
	"kindlecord/internal/ui"
	"kindlecord/internal/updater"
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
		if strings.Contains(exe, "go-build") {
			if cwd, err := os.Getwd(); err == nil {
				projectRoot = cwd
			}
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(cwd, "data")); err == nil {
			projectRoot = cwd
		}
	}
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

func serverName(g map[string]interface{}) string {
	if n, ok := g["name"].(string); ok {
		return n
	}
	return "?"
}

func channelName(c map[string]interface{}) string {
	if n, ok := c["name"].(string); ok {
		return "#" + n
	}
	return "#?"
}

func dmDisplayName(ch map[string]interface{}) string {
	if n, ok := ch["name"].(string); ok && n != "" {
		return n
	}
	if recipients, ok := ch["recipients"].([]interface{}); ok && len(recipients) > 0 {
		if r, ok := recipients[0].(map[string]interface{}); ok {
			if u, ok := r["username"].(string); ok {
				return u
			}
		}
	}
	return "DM"
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	cfg := loadConfig()

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
		finishSetup := make(chan struct{}, 1)

		loginScreen := ui.NewLoginScreen(url, func() { quitApp = true })
		loginScreen.OnFinish = func() {
			select {
			case finishSetup <- struct{}{}:
			default:
			}
		}
		app := ui.NewApp(disp)
		app.Add("login", loginScreen)
		app.Show("login", map[string]interface{}{
			"url":       url,
			"ssh_info":  sshInfo,
			"on_quit":   func() { quitApp = true },
			"on_finish": loginScreen.OnFinish,
			"page":      0,
		})

		powerCh := make(chan bool, 1)
		go func() {
			for {
				power.Poll()
				if power.IsDouble() {
					select {
					case powerCh <- true:
					default:
					}
					return
				}
				time.Sleep(30 * time.Millisecond)
			}
		}()

	outerLogin:
		for {
			select {
			case <-powerCh:
				quitApp = true
				break outerLogin
			case t := <-tokenCh:
				token = t
				// Show success page (step 3)
				app.Show("login", map[string]interface{}{"page": 2})
			case <-finishSetup:
				// User clicked "Concluir setup" on success page
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

	client := discord.NewClient(token, cfg.DiscordAPIBase)
	_, err := client.Login()
	if err != nil {
		log.Printf("[MAIN] login fail: %v", err)
		title := "Invalid token"
		if !strings.Contains(err.Error(), "401") {
			title = "Error"
		}
		lines := strings.Split(err.Error(), "\n")
		if len(lines) > 5 {
			lines = lines[:5]
		}
		lines = append(lines, "", "Token was removed.", "Tap Exit and try again.")
		disp.Clear(0xFF)
		disp.Refresh()
		errApp := ui.NewApp(disp)
		done := false
		errScreen := ui.NewErrorScreen(title, lines, func() {
			_ = os.Remove(tokenFile)
			done = true
		})
		errApp.Add("error", errScreen)
		errApp.Show("error", nil)
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
			default:
			}
			ev := reader.Poll(50 * time.Millisecond)
			if ev != nil && ev.Press {
				errApp.Touch(ev.X, ev.Y)
			}
		}
		disp.Clear(0xFF)
		disp.Refresh()
		return
	}

	type state struct {
		guilds    []map[string]interface{}
		dms       []map[string]interface{}
		channels  map[string][]map[string]interface{}
		selDM     bool
		selServer int
	}

	app := ui.NewApp(disp)
	st := &state{channels: make(map[string][]map[string]interface{})}

	homeScreen := ui.NewHomeScreen()
	app.Add("home", homeScreen)

	settingsScreen := ui.NewHomeScreen()
	app.Add("settings", settingsScreen)

	msgScreen := ui.NewMessageScreen()
	app.Add("messages", msgScreen)

	loadingScreen := ui.NewLoadingScreen()
	app.Add("loading", loadingScreen)

	statusDialog := ui.NewDialog("Status", "", nil)
	app.Add("status", statusDialog)

	var showDMs func()
	var showChannels func(int)
	var showStatus func(title, message string)
	var sidebarArgs func() map[string]interface{}
	var showSettings func()
	var resetRequested bool

	showStatus = func(title, message string) {
		statusDialog.OnOK = func() {
			app.Show("home", sidebarArgs())
		}
		app.Show("status", map[string]interface{}{
			"title":   title,
			"message": message,
			"on_ok":   statusDialog.OnOK,
		})
	}

	showLoading := func(msg string) {
		app.Show("loading", map[string]interface{}{"message": msg})
	}

	checkForUpdates := func() {
		go func() {
			tag, url, hasUpdate, err := updater.CheckLatest()
			if err != nil {
				log.Printf("[UPDATE] check failed: %v", err)
				showStatus("Update Error", err.Error())
				return
			}
			if !hasUpdate {
				showStatus("No Updates", "Already on latest version")
				return
			}
			log.Printf("[UPDATE] new version: %s", tag)
			// ask first, then download & install
			confirmDialog := ui.NewDialog("Update Available", "New version "+tag+" is available.\nDownload and install now?", nil)
			app.Add("update_confirm", confirmDialog)
			confirmDialog.OnOK = func() {
				showStatus("Downloading", "Downloading "+tag+"...")
				go func() {
					tmpPath, err := updater.Download(url)
					if err != nil {
						log.Printf("[UPDATE] download failed: %v", err)
						showStatus("Download Error", err.Error())
						return
					}
					log.Printf("[UPDATE] downloaded to %s, installing...", tmpPath)
					showStatus("Installing", "Installing "+tag+"...")
					if err := updater.Install(tmpPath); err != nil {
						log.Printf("[UPDATE] install failed: %v", err)
						showStatus("Install Error", err.Error())
						return
					}
					log.Printf("[UPDATE] installed, restarting")
					os.Exit(0)
				}()
			}
			app.Show("update_confirm", map[string]interface{}{
				"title":   "Update Available",
				"message": "New version " + tag + " is available.\nDownload and install now?",
				"on_ok":   confirmDialog.OnOK,
			})
		}()
	}

	sidebarArgs = func() map[string]interface{} {
		serverNames := make([]string, len(st.guilds))
		for i, g := range st.guilds {
			serverNames[i] = serverName(g)
		}
		return map[string]interface{}{
			"servers":     serverNames,
			"selected_dm": st.selDM,
			"server_idx":  st.selServer,
			"on_dm_click": func() {
				st.selDM = true
				showDMs()
			},
			"on_server_click": func(idx int) {
				st.selDM = false
				st.selServer = idx
				showChannels(idx)
			},
			"on_update_click": func() {
				checkForUpdates()
			},
			"on_settings_click": func() {
				showSettings()
			},
		}
	}

	showSettings = func() {
		args := sidebarArgs()
		args["title"] = "Settings"
		args["items"] = []string{
			"Check for updates",
			"Reset app to setup",
		}
		args["on_back"] = func() { showDMs() }
		args["on_select"] = func(idx int) {
			if idx == 0 {
				checkForUpdates()
			} else if idx == 1 {
				resetDialog := ui.NewDialog("Reset to Setup", "Remove your token and restart?\nYou will need to log in again.", func() {
					resetRequested = true
					app.Stop()
				})
				app.Add("reset_confirm", resetDialog)
				app.Show("reset_confirm", map[string]interface{}{
					"title":     "Reset to Setup",
					"message":   "Remove your token and restart?\nYou will need to log in again.",
					"on_ok":     resetDialog.OnOK,
					"on_cancel": func() { showSettings() },
				})
			}
		}
		app.Show("settings", args)
	}

	showDMs = func() {
		showLoading("Loading DMs...")
		go func() {
			dms, err := client.GetDMs()
			if err != nil {
				log.Printf("[MAIN] get DMs fail: %v", err)
				dms = nil
			}
			st.dms = dms
			items := make([]string, len(dms))
			for i, ch := range dms {
				items[i] = dmDisplayName(ch)
			}
			if len(items) == 0 {
				items = []string{"(no conversations)"}
			}
			args := sidebarArgs()
			args["title"] = "Direct Messages"
			args["items"] = items
			args["on_select"] = func(idx int) {
				if idx >= len(st.dms) {
					return
				}
				dmID, _ := st.dms[idx]["id"].(string)
				dmName := dmDisplayName(st.dms[idx])
				showLoading("Loading messages...")
				go func() {
					msgs, err := client.GetMessages(dmID, 50, "")
					if err != nil {
						log.Printf("[MAIN] get DM messages fail: %v", err)
						showStatus("Load Error", err.Error())
						return
					}
					for l, r := 0, len(msgs)-1; l < r; l, r = l+1, r-1 {
						msgs[l], msgs[r] = msgs[r], msgs[l]
					}
					msgArgs := sidebarArgs()
					msgArgs["title"] = dmName
					msgArgs["messages"] = msgs
					msgArgs["on_back"] = func() { showDMs() }
					app.Show("messages", msgArgs)
				}()
			}
			app.Show("home", args)
		}()
	}

	showChannels = func(guildIdx int) {
		if guildIdx >= len(st.guilds) {
			return
		}
		gid, _ := st.guilds[guildIdx]["id"].(string)
		gname := serverName(st.guilds[guildIdx])
		showLoading("Loading channels...")
		go func() {
			channels, err := client.GetChannels(gid)
			if err != nil {
				log.Printf("[MAIN] get channels fail: %v", err)
				channels = nil
			}
			st.channels[gid] = channels

			var textChannels []map[string]interface{}
			for _, c := range channels {
				if t, ok := c["type"].(float64); ok && t == 0 {
					textChannels = append(textChannels, c)
				} else if t, ok := c["type"].(int); ok && t == 0 {
					textChannels = append(textChannels, c)
				}
			}

			items := make([]string, len(textChannels))
			chIDs := make([]string, len(textChannels))
			for i, c := range textChannels {
				items[i] = channelName(c)
				chIDs[i], _ = c["id"].(string)
			}
			if len(items) == 0 {
				items = []string{"(no text channels)"}
			}

			args := sidebarArgs()
			args["title"] = gname
			args["items"] = items
			args["on_select"] = func(idx int) {
				if idx >= len(chIDs) {
					return
				}
				cid := chIDs[idx]
				cname := items[idx]
				showLoading("Loading messages...")
				go func() {
					msgs, err := client.GetMessages(cid, 50, "")
					if err != nil {
						log.Printf("[MAIN] get messages fail: %v", err)
						showStatus("Load Error", err.Error())
						return
					}
					for l, r := 0, len(msgs)-1; l < r; l, r = l+1, r-1 {
						msgs[l], msgs[r] = msgs[r], msgs[l]
					}
					msgArgs := sidebarArgs()
					msgArgs["title"] = cname
					msgArgs["messages"] = msgs
					msgArgs["on_back"] = func() { showChannels(guildIdx) }
					app.Show("messages", msgArgs)
				}()
			}
			app.Show("home", args)
		}()
	}

	guilds, err := client.GetGuilds()
	if err != nil {
		log.Printf("[MAIN] get guilds fail: %v", err)
	}
	st.guilds = guilds
	st.selDM = true
	showDMs()

	// prevent Kindle auto-suspend
	go func() {
		procs := []string{"com.lab126.powerd"}
		for {
			for _, svc := range procs {
				_ = exec.Command("lipc-set-prop", svc, "preventScreenSaver", "true").Run()
			}
			time.Sleep(20 * time.Second)
		}
	}()

	powerMainCh := make(chan bool, 1)
	go func() {
		for app.Running {
			power.Poll()
			if power.IsDouble() {
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
			app.Stop()
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

	if resetRequested {
		log.Println("[MAIN] reset requested, removing token and restarting for setup")
		_ = os.Remove(tokenFile)
		// start.sh restarts on exit code 42
		os.Exit(42)
	}

	disp.Clear(0xFF)
	disp.Refresh()
	log.Println("[MAIN] exit clean")
}
