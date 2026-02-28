package handler

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
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

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/zhaoxinyi02/ClawPanel/internal/config"
)

// === Admin Config ===

func GetAdminConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminCfg := loadAdminConfig(cfg)
		c.JSON(200, gin.H{"ok": true, "config": adminCfg})
	}
}

func SaveAdminConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"ok": false, "error": err.Error()})
			return
		}
		saveAdminConfigData(cfg, body)
		c.JSON(200, gin.H{"ok": true})
	}
}

func SaveAdminSection(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		section := c.Param("section")
		var body interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"ok": false, "error": err.Error()})
			return
		}
		adminCfg := loadAdminConfig(cfg)
		adminCfg[section] = body
		saveAdminConfigData(cfg, adminCfg)
		c.JSON(200, gin.H{"ok": true})
	}
}

func adminConfigPath(cfg *config.Config) string {
	return filepath.Join(cfg.DataDir, "admin-config.json")
}

func loadAdminConfig(cfg *config.Config) map[string]interface{} {
	result := map[string]interface{}{
		"server": map[string]interface{}{
			"token": cfg.AdminToken,
			"port":  cfg.Port,
		},
	}
	data, err := os.ReadFile(adminConfigPath(cfg))
	if err == nil {
		json.Unmarshal(data, &result)
	}
	return result
}

func saveAdminConfigData(cfg *config.Config, data map[string]interface{}) {
	out, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(adminConfigPath(cfg), out, 0644)
}

// === Admin Token ===

func GetAdminToken(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "token": cfg.AdminToken})
	}
}

// === Sudo Password ===

func GetSudoPassword(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminCfg := loadAdminConfig(cfg)
		sys, _ := adminCfg["system"].(map[string]interface{})
		hasPwd := false
		if sys != nil {
			if pwd, ok := sys["sudoPassword"].(string); ok && pwd != "" {
				hasPwd = true
			}
		}
		c.JSON(200, gin.H{"ok": true, "configured": hasPwd})
	}
}

func SetSudoPassword(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Password string `json:"password"`
		}
		c.ShouldBindJSON(&body)
		adminCfg := loadAdminConfig(cfg)
		sys, ok := adminCfg["system"].(map[string]interface{})
		if !ok {
			sys = map[string]interface{}{}
		}
		sys["sudoPassword"] = body.Password
		adminCfg["system"] = sys
		saveAdminConfigData(cfg, adminCfg)
		c.JSON(200, gin.H{"ok": true})
	}
}

// === Bot Operations (OneBot proxy) ===

func onebotWsUrl(cfg *config.Config) string {
	ocCfg := readOpenClawJSON(cfg)
	if channels, ok := ocCfg["channels"].(map[string]interface{}); ok {
		if qq, ok := channels["qq"].(map[string]interface{}); ok {
			if wsUrl, ok := qq["wsUrl"].(string); ok && wsUrl != "" {
				return wsUrl
			}
		}
	}
	return "ws://127.0.0.1:3001"
}

func GetBotGroups(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "groups": []interface{}{}})
	}
}

func GetBotFriends(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "friends": []interface{}{}})
	}
}

func BotSend(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	}
}

func BotReconnect(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	}
}

// === Requests (approval) ===

func GetRequests(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "requests": []interface{}{}})
	}
}

func ApproveRequest(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	}
}

func RejectRequest(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	}
}

// === NapCat Login Proxy ===

const NAPCAT_WEBUI = "http://127.0.0.1:6099"

var napcatCredential string

func napcatProxy(method, path string, body interface{}, credential string) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, NAPCAT_WEBUI+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if credential != "" {
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]interface{}{"raw": string(data)}, nil
	}
	return result, nil
}

// readNapCatWebuiToken reads the actual token from NapCat's webui.json (Docker or Windows Shell).
func readNapCatWebuiToken(cfg *config.Config) string {
	if runtime.GOOS == "windows" {
		napcatDir := getNapCatShellDir(cfg)
		if napcatDir != "" {
			data, err := os.ReadFile(filepath.Join(napcatDir, "config", "webui.json"))
			if err == nil {
				var webui map[string]interface{}
				if json.Unmarshal(data, &webui) == nil {
					if t, ok := webui["token"].(string); ok && t != "" {
						return t
					}
				}
			}
		}
	} else {
		out, err := exec.Command("docker", "exec", "openclaw-qq", "cat", "/app/napcat/config/webui.json").Output()
		if err == nil {
			var webui map[string]interface{}
			if json.Unmarshal(out, &webui) == nil {
				if t, ok := webui["token"].(string); ok && t != "" {
					return t
				}
			}
		}
	}
	return ""
}

// napcatLoginWithToken attempts to authenticate with NapCat WebUI using a given token.
func napcatLoginWithToken(token string) string {
	hash := sha256.Sum256([]byte(token + ".napcat"))
	hashStr := fmt.Sprintf("%x", hash)
	r, err := napcatProxy("POST", "/api/auth/login", map[string]string{"hash": hashStr}, "")
	if err == nil {
		if code, ok := r["code"].(float64); ok && code == 0 {
			if data, ok := r["data"].(map[string]interface{}); ok {
				if cred, ok := data["Credential"].(string); ok {
					return cred
				}
			}
		}
	}
	return ""
}

func napcatAuth(cfg *config.Config) string {
	if napcatCredential != "" {
		return napcatCredential
	}

	// 1. Try the actual token from NapCat's webui.json (ground truth)
	containerToken := readNapCatWebuiToken(cfg)
	if containerToken != "" {
		if cred := napcatLoginWithToken(containerToken); cred != "" {
			napcatCredential = cred
			return napcatCredential
		}
	}

	// 2. Fallback: admin-config or default token (may differ from container)
	adminCfg := loadAdminConfig(cfg)
	webuiToken := "clawpanel-qq"
	if napcat, ok := adminCfg["napcat"].(map[string]interface{}); ok {
		if t, ok := napcat["webuiToken"].(string); ok && t != "" {
			webuiToken = t
		}
	}
	if envToken := os.Getenv("WEBUI_TOKEN"); envToken != "" {
		webuiToken = envToken
	}
	if webuiToken != containerToken {
		if cred := napcatLoginWithToken(webuiToken); cred != "" {
			napcatCredential = cred
			return napcatCredential
		}
	}

	return napcatCredential
}

func isNapcatUnauthorized(r map[string]interface{}) bool {
	// Check code == -1 with unauthorized message
	if code, ok := r["code"].(float64); ok && code == -1 {
		if msg, ok := r["message"].(string); ok {
			lower := strings.ToLower(msg)
			if strings.Contains(lower, "unauthorized") || strings.Contains(lower, "auth") {
				return true
			}
		}
	}
	// Check raw response (NapCat may return non-JSON 401)
	if raw, ok := r["raw"].(string); ok {
		lower := strings.ToLower(raw)
		if strings.Contains(lower, "unauthorized") || strings.Contains(lower, "401") {
			return true
		}
	}
	return false
}

func napcatApiCall(cfg *config.Config, method, path string, body interface{}) (map[string]interface{}, error) {
	cred := napcatAuth(cfg)
	r, err := napcatProxy(method, path, body, cred)
	if err != nil {
		return nil, fmt.Errorf("NapCat 服务不可达，可能正在启动中，请稍候重试")
	}
	// If unauthorized, clear cache and retry with fresh auth
	if isNapcatUnauthorized(r) {
		napcatCredential = ""
		cred = napcatAuth(cfg)
		if cred != "" {
			return napcatProxy(method, path, body, cred)
		}
		// Auth still fails — NapCat may be restarting, return friendly message
		return nil, fmt.Errorf("NapCat 认证失败，服务可能正在重启中，请等待几秒后重试")
	}
	return r, nil
}

func NapcatLoginStatus(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, err := napcatApiCall(cfg, "POST", "/api/QQLogin/CheckLoginStatus", nil)
		if err != nil {
			c.JSON(200, gin.H{"ok": false, "error": err.Error()})
			return
		}
		r["ok"] = true
		c.JSON(200, r)
	}
}

func NapcatGetQRCode(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retry up to 5 times (total ~10s) to wait for NapCat to become ready
		var r map[string]interface{}
		var err error
		for i := 0; i < 5; i++ {
			r, err = napcatApiCall(cfg, "POST", "/api/QQLogin/GetQQLoginQrcode", nil)
			if err == nil {
				break
			}
			// NapCat not ready yet, wait and retry
			time.Sleep(2 * time.Second)
			napcatCredential = "" // clear stale credential
		}
		if err != nil {
			c.JSON(200, gin.H{"ok": false, "error": err.Error()})
			return
		}
		r["ok"] = true
		// NapCat returns a URL in data.qrcode — convert to base64 QR image
		if data, ok := r["data"].(map[string]interface{}); ok {
			if qrURL, ok := data["qrcode"].(string); ok && qrURL != "" && !strings.HasPrefix(qrURL, "data:") {
				if png, err := qrcode.Encode(qrURL, qrcode.Medium, 256); err == nil {
					data["qrcode"] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
				}
			}
		}
		c.JSON(200, r)
	}
}

func NapcatRefreshQRCode(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the old QR URL first so we can detect if it actually changed
		oldR, _ := napcatApiCall(cfg, "POST", "/api/QQLogin/GetQQLoginQrcode", nil)
		oldURL := ""
		if oldData, ok := oldR["data"].(map[string]interface{}); ok {
			oldURL, _ = oldData["qrcode"].(string)
		}

		// Call RefreshQRcode to invalidate old QR
		napcatApiCall(cfg, "POST", "/api/QQLogin/RefreshQRcode", nil)
		time.Sleep(500 * time.Millisecond)

		// Retry up to 5 times to get a genuinely new QR code
		var r map[string]interface{}
		var err error
		for i := 0; i < 5; i++ {
			r, err = napcatApiCall(cfg, "POST", "/api/QQLogin/GetQQLoginQrcode", nil)
			if err == nil {
				if data, ok := r["data"].(map[string]interface{}); ok {
					if newURL, ok := data["qrcode"].(string); ok && newURL != "" && newURL != oldURL {
						break // Got a new QR code
					}
				}
			}
			time.Sleep(800 * time.Millisecond)
		}

		if err != nil {
			c.JSON(200, gin.H{"ok": false, "error": err.Error()})
			return
		}
		r["ok"] = true
		// NapCat returns a URL in data.qrcode — convert to base64 QR image
		if data, ok := r["data"].(map[string]interface{}); ok {
			if qrURL, ok := data["qrcode"].(string); ok && qrURL != "" && !strings.HasPrefix(qrURL, "data:") {
				if png, err := qrcode.Encode(qrURL, qrcode.Medium, 256); err == nil {
					data["qrcode"] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
				}
			}
		}
		c.JSON(200, r)
	}
}

func NapcatQuickLoginList(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, err := napcatApiCall(cfg, "POST", "/api/QQLogin/GetQuickLoginQQ", nil)
		if err != nil {
			c.JSON(200, gin.H{"ok": false, "error": err.Error()})
			return
		}
		r["ok"] = true
		c.JSON(200, r)
	}
}

func NapcatQuickLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Uin string `json:"uin"`
		}
		c.ShouldBindJSON(&body)
		r, err := napcatApiCall(cfg, "POST", "/api/QQLogin/SetQuickLogin", map[string]string{"uin": body.Uin})
		if err != nil {
			c.JSON(200, gin.H{"ok": false, "error": err.Error()})
			return
		}
		r["ok"] = true
		c.JSON(200, r)
	}
}

func NapcatPasswordLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Uin      string `json:"uin"`
			Password string `json:"password"`
		}
		c.ShouldBindJSON(&body)
		pwd := body.Password
		hash := md5.Sum([]byte(pwd))
		passwordMd5 := fmt.Sprintf("%x", hash)
		r, err := napcatApiCall(cfg, "POST", "/api/QQLogin/PasswordLogin", map[string]string{"uin": body.Uin, "passwordMd5": passwordMd5})
		if err != nil {
			c.JSON(200, gin.H{"ok": false, "error": err.Error()})
			return
		}
		r["ok"] = true
		c.JSON(200, r)
	}
}

func NapcatLoginInfo(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, err := napcatApiCall(cfg, "POST", "/api/QQLogin/GetQQLoginInfo", nil)
		if err != nil {
			c.JSON(200, gin.H{"ok": false, "error": err.Error()})
			return
		}
		r["ok"] = true
		c.JSON(200, r)
	}
}

func NapcatLogout(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		go func() {
			restartNapCatProcess(cfg)
		}()
		c.JSON(200, gin.H{"ok": true, "message": "QQ 正在退出登录，NapCat 重启中..."})
	}
}

func RestartNapcat(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		go func() {
			restartNapCatProcess(cfg)
		}()
		c.JSON(200, gin.H{"ok": true, "message": "NapCat 正在重启..."})
	}
}

// restartNapCatProcess restarts NapCat based on platform: Docker on Linux/macOS, Shell process on Windows
func restartNapCatProcess(cfg *config.Config) {
	if runtime.GOOS == "windows" {
		// Kill NapCat processes on Windows
		exec.Command("taskkill", "/F", "/IM", "NapCatWinBootMain.exe").Run()
		exec.Command("taskkill", "/F", "/IM", "napcat.exe").Run()
		exec.Command("taskkill", "/F", "/IM", "QQ.exe").Run()
		// Restart: find and launch napcat.bat
		napcatDir := getNapCatShellDir(cfg)
		if napcatDir != "" {
			batPath := filepath.Join(napcatDir, "napcat.bat")
			if _, err := os.Stat(batPath); err == nil {
				cmd := exec.Command("cmd", "/C", "start", "/B", batPath)
				cmd.Dir = napcatDir
				cmd.Start()
				return
			}
			exePath := filepath.Join(napcatDir, "NapCatWinBootMain.exe")
			if _, err := os.Stat(exePath); err == nil {
				cmd := exec.Command(exePath)
				cmd.Dir = napcatDir
				cmd.Start()
			}
		}
	} else {
		exec.Command("docker", "restart", "openclaw-qq").Run()
	}
}

// === WeChat API ===

func WechatStatus(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "connected": false, "loggedIn": false, "name": ""})
	}
}

func WechatLoginUrl(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminCfg := loadAdminConfig(cfg)
		token := ""
		if wc, ok := adminCfg["wechat"].(map[string]interface{}); ok {
			if t, ok := wc["token"].(string); ok {
				token = t
			}
		}
		host := c.Request.Host
		if idx := strings.Index(host, ":"); idx > 0 {
			host = host[:idx]
		}
		externalUrl := fmt.Sprintf("http://%s:3002/login?token=%s", host, token)
		c.JSON(200, gin.H{"ok": true, "externalUrl": externalUrl, "internalUrl": ""})
	}
}

func WechatSend(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	}
}

func WechatSendFile(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	}
}

func WechatGetConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminCfg := loadAdminConfig(cfg)
		wc := adminCfg["wechat"]
		if wc == nil {
			wc = map[string]interface{}{}
		}
		c.JSON(200, gin.H{"ok": true, "config": wc})
	}
}

func WechatUpdateConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]interface{}
		c.ShouldBindJSON(&body)
		adminCfg := loadAdminConfig(cfg)
		existing, ok := adminCfg["wechat"].(map[string]interface{})
		if !ok {
			existing = map[string]interface{}{}
		}
		for k, v := range body {
			existing[k] = v
		}
		adminCfg["wechat"] = existing
		saveAdminConfigData(cfg, adminCfg)
		c.JSON(200, gin.H{"ok": true})
	}
}

// === ClawHub Sync ===

func ClawHubSync(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		cachePath := filepath.Join(cfg.OpenClawDir, "clawhub-cache.json")
		// Try to read cache
		if data, err := os.ReadFile(cachePath); err == nil {
			var cached map[string]interface{}
			if json.Unmarshal(data, &cached) == nil {
				c.JSON(200, gin.H{"ok": true, "skills": cached["skills"], "source": "cache", "syncedAt": cached["syncedAt"]})
				return
			}
		}
		c.JSON(200, gin.H{"ok": true, "skills": []interface{}{}, "source": "empty"})
	}
}

// helper to read openclaw.json
func readOpenClawJSON(cfg *config.Config) map[string]interface{} {
	data, err := os.ReadFile(filepath.Join(cfg.OpenClawDir, "openclaw.json"))
	if err != nil {
		return map[string]interface{}{}
	}
	var result map[string]interface{}
	if json.Unmarshal(data, &result) != nil {
		return map[string]interface{}{}
	}
	return result
}
