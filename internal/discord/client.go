package discord

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBase = "discord.com"
	APIVersion  = "/api/v10"
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
)

type Client struct {
	Token   string
	Base    string
	HTTP    *http.Client
	User    map[string]interface{}
}

type DiscordError struct {
	Status int
	Body   string
}

func (e *DiscordError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body)
}

func NewClient(token, base string) *Client {
	token = strings.TrimSpace(strings.Trim(token, `"' `))
	if base == "" {
		base = DefaultBase
	}
	// strip https:// and path if user passed full URL
	base = strings.TrimPrefix(base, "https://")
	base = strings.TrimPrefix(base, "http://")
	if idx := strings.Index(base, "/"); idx >= 0 {
		base = base[:idx]
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Kindle CA outdated
	}
	return &Client{
		Token: token,
		Base:  base,
		HTTP: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (c *Client) authHeader() string {
	t := c.Token
	if strings.HasPrefix(t, "Bot ") || strings.HasPrefix(t, "Bearer ") {
		return t
	}
	// User tokens must NOT have Bot prefix (fix bug from original)
	return t
}

func (c *Client) request(method, path string, data interface{}) ([]byte, error) {
	urlStr := "https://" + c.Base + APIVersion + path
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Host", c.Base)

	log.Printf("[DISCORD] %s %s", method, APIVersion+path)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &DiscordError{Status: 0, Body: fmt.Sprintf("Network error: %v", err)}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	log.Printf("[DISCORD] response %d", resp.StatusCode)
	if resp.StatusCode >= 400 {
		// handle rate limit
		if resp.StatusCode == 429 {
			var rl struct {
				RetryAfter float64 `json:"retry_after"`
			}
			_ = json.Unmarshal(b, &rl)
			if rl.RetryAfter > 0 {
				time.Sleep(time.Duration(rl.RetryAfter*1000) * time.Millisecond)
				return c.request(method, path, data)
			}
		}
		return nil, &DiscordError{Status: resp.StatusCode, Body: string(b)}
	}
	return b, nil
}

func (c *Client) Login() (map[string]interface{}, error) {
	b, err := c.request("GET", "/users/@me", nil)
	if err != nil {
		return nil, err
	}
	var u map[string]interface{}
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, err
	}
	c.User = u
	return u, nil
}

func (c *Client) GetGuilds() ([]map[string]interface{}, error) {
	b, err := c.request("GET", "/users/@me/guilds", nil)
	if err != nil {
		return nil, err
	}
	var v []map[string]interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (c *Client) GetChannels(guildID string) ([]map[string]interface{}, error) {
	b, err := c.request("GET", "/guilds/"+guildID+"/channels", nil)
	if err != nil {
		return nil, err
	}
	var v []map[string]interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (c *Client) GetMessages(channelID string, limit int, before string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/channels/%s/messages?limit=%d", channelID, limit)
	if before != "" {
		path += "&before=" + url.QueryEscape(before)
	}
	b, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var v []map[string]interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (c *Client) SendMessage(channelID, content string) (map[string]interface{}, error) {
	b, err := c.request("POST", "/channels/"+channelID+"/messages", map[string]string{"content": content})
	if err != nil {
		return nil, err
	}
	var v map[string]interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (c *Client) GetDMs() ([]map[string]interface{}, error) {
	b, err := c.request("GET", "/users/@me/channels", nil)
	if err != nil {
		return nil, err
	}
	var v []map[string]interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}
