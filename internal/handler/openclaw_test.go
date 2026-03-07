package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhaoxinyi02/ClawPanel/internal/config"
)

func TestSaveOpenClawConfigPreservesCriticalFields(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	cfg := &config.Config{OpenClawDir: dir}

	initial := map[string]interface{}{
		"tools": map[string]interface{}{
			"agentToAgent": true,
		},
		"session": map[string]interface{}{
			"maxMessages": 50,
		},
		"cron": map[string]interface{}{
			"jobs": []interface{}{
				map[string]interface{}{"id": "job_1"},
			},
		},
		"models": map[string]interface{}{
			"providers": map[string]interface{}{},
		},
	}
	raw, _ := json.Marshal(initial)
	if err := os.WriteFile(filepath.Join(dir, "openclaw.json"), raw, 0644); err != nil {
		t.Fatalf("write openclaw.json: %v", err)
	}

	r := gin.New()
	r.PUT("/openclaw/config", SaveOpenClawConfig(cfg))

	body, _ := json.Marshal(map[string]interface{}{"config": initial})
	req := httptest.NewRequest(http.MethodPut, "/openclaw/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	saved, err := cfg.ReadOpenClawJSON()
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if _, ok := saved["tools"]; !ok {
		t.Fatalf("tools should be preserved")
	}
	if _, ok := saved["session"]; !ok {
		t.Fatalf("session should be preserved")
	}
	cron, ok := saved["cron"].(map[string]interface{})
	if !ok {
		t.Fatalf("cron should be preserved")
	}
	if _, ok := cron["jobs"]; !ok {
		t.Fatalf("cron.jobs should be preserved")
	}
}

func TestPatchModelsJSONForAgentUsesConfiguredAgentDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{OpenClawDir: dir}
	writeJSONRaw := func(path string, data map[string]interface{}) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		raw, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("marshal json: %v", err)
		}
		if err := os.WriteFile(path, raw, 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeJSONRaw(filepath.Join(dir, "openclaw.json"), map[string]interface{}{
		"agents": map[string]interface{}{
			"list": []interface{}{
				map[string]interface{}{"id": "work", "agentDir": "custom/work-agent"},
			},
		},
	})
	modelsPath := filepath.Join(dir, "custom", "work-agent", "agent", "models.json")
	writeJSONRaw(modelsPath, map[string]interface{}{
		"providers": map[string]interface{}{
			"deepseek": map[string]interface{}{
				"baseUrl": "https://api.deepseek.com/v1",
				"models": []interface{}{
					map[string]interface{}{"id": "deepseek-chat", "compat": map[string]interface{}{"supportsDeveloperRole": true}},
				},
			},
		},
	})

	patchModelsJSONForAgent(cfg, "work")

	raw, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("read models.json: %v", err)
	}
	var saved map[string]interface{}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("decode models.json: %v", err)
	}
	providers, _ := saved["providers"].(map[string]interface{})
	provider, _ := providers["deepseek"].(map[string]interface{})
	models, _ := provider["models"].([]interface{})
	model, _ := models[0].(map[string]interface{})
	compat, _ := model["compat"].(map[string]interface{})
	if got, _ := compat["supportsDeveloperRole"].(bool); got {
		t.Fatalf("expected compat.supportsDeveloperRole to be forced false")
	}
}

func TestSaveChannelRejectsQQWhenPluginMissing(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	cfg := &config.Config{OpenClawDir: dir}
	r := gin.New()
	r.PUT("/openclaw/channels/:id", SaveChannel(cfg, nil))

	body := []byte(`{"enabled":true,"wsUrl":"ws://127.0.0.1:3001"}`)
	req := httptest.NewRequest(http.MethodPut, "/openclaw/channels/qq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestToggleChannelRejectsQQWhenPluginMissing(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	cfg := &config.Config{OpenClawDir: dir}
	r := gin.New()
	r.PUT("/openclaw/channels/toggle", ToggleChannel(cfg, nil, nil))

	body := []byte(`{"channelId":"qq","enabled":true}`)
	req := httptest.NewRequest(http.MethodPut, "/openclaw/channels/toggle", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestSaveChannelQQReturnsMessageWithoutProcessManager(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "extensions", "qq"), 0755); err != nil {
		t.Fatalf("mkdir qq extension: %v", err)
	}
	cfg := &config.Config{OpenClawDir: dir}
	r := gin.New()
	r.PUT("/openclaw/channels/:id", SaveChannel(cfg, nil))

	body := []byte(`{"enabled":true,"wsUrl":"ws://127.0.0.1:3001","notifications":{"antiRecall":false}}`)
	req := httptest.NewRequest(http.MethodPut, "/openclaw/channels/qq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("expected ok response, got %#v", resp)
	}
}
