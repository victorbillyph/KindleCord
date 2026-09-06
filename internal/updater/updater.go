package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	Repo      = "victorbillyph/KindleCord"
	AssetName = "kindlecord-arm"
	APIURL    = "https://api.github.com/repos/%s/releases/latest"
)

var (
	CurrentVersion = "v0.3.6"
	ExecPath       func() string
)

func exePath() string {
	if ExecPath != nil {
		return ExecPath()
	}
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return ""
}

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// CheckLatest queries GitHub for the latest release. Returns (tag, downloadURL, hasUpdate, error).
func CheckLatest() (string, string, bool, error) {
	url := fmt.Sprintf(APIURL, Repo)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", false, fmt.Errorf("github status %d", resp.StatusCode)
	}
	var rel releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", false, err
	}
	tag := rel.TagName
	if tag == "" {
		return "", "", false, fmt.Errorf("no tag in release")
	}
	if strings.TrimPrefix(tag, "v") == strings.TrimPrefix(CurrentVersion, "v") {
		return tag, "", false, nil
	}
	if !newerVersion(tag, CurrentVersion) {
		return tag, "", false, nil
	}
	for _, a := range rel.Assets {
		if a.Name == AssetName {
			return tag, a.BrowserDownloadURL, true, nil
		}
	}
	return tag, "", true, nil
}

func newerVersion(a, b string) bool {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	max := len(pa)
	if len(pb) > max {
		max = len(pb)
	}
	for i := 0; i < max; i++ {
		var na, nb int
		if i < len(pa) {
			fmt.Sscanf(pa[i], "%d", &na)
		}
		if i < len(pb) {
			fmt.Sscanf(pb[i], "%d", &nb)
		}
		if na != nb {
			return na > nb
		}
	}
	return false
}

// Download fetches url to a temp file. Returns tmp path.
func Download(url string) (string, error) {
	tmp, err := os.CreateTemp("", "kc-update-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	resp, err := http.Get(url)
	if err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	tmp.Close()
	return tmpPath, nil
}

// Install replaces the current executable with newFile, chmod, then exec the new binary.
func Install(newFile string) error {
	dst := exePath()
	if dst == "" {
		return fmt.Errorf("cannot find executable path")
	}
	if err := os.Chmod(newFile, 0755); err != nil {
		return err
	}
	// on Kindle, replace the binary on the extension mount
	if err := os.Rename(newFile, dst); err != nil {
		// cross-device; fallback to copy
		if err2 := copyFile(newFile, dst); err2 != nil {
			return err2
		}
		os.Remove(newFile)
	}
	// start.sh restarts via the extension script; just exec ourselves
	dir := filepath.Dir(dst)
	sh := filepath.Join(dir, "bin", "start.sh")
	if _, err := os.Stat(sh); err == nil {
		// detach and let the launcher take over
		cmd := exec.Command(sh)
		cmd.Dir = dir
		if err := cmd.Start(); err == nil {
			os.Exit(0)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

// unused otherwise
var _ = runtime.GOOS