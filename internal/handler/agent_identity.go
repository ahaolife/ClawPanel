package handler

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhaoxinyi02/ClawPanel/internal/config"
)

const agentAvatarMaxBytes int64 = 2 * 1024 * 1024

var agentAvatarSchemeRE = regexp.MustCompile(`^[a-z][a-z0-9+.-]*:`)
var agentAvatarHTTPRE = regexp.MustCompile(`^https?://`)
var agentAvatarDataRE = regexp.MustCompile(`^data:`)
var agentAvatarMimeByExt = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
}
var agentIdentityAllowedKeys = map[string]struct{}{
	"name":   {},
	"theme":  {},
	"emoji":  {},
	"avatar": {},
}
var agentIdentityLegacyThemeKeys = []string{"description", "vibe", "tone", "creature"}

func trimStringField(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func isAgentAvatarDataURL(value string) bool {
	return agentAvatarDataRE.MatchString(strings.ToLower(strings.TrimSpace(value)))
}

func isAgentAvatarHTTPURL(value string) bool {
	return agentAvatarHTTPRE.MatchString(strings.ToLower(strings.TrimSpace(value)))
}

func hasAgentAvatarScheme(value string) bool {
	return agentAvatarSchemeRE.MatchString(strings.TrimSpace(value))
}

func resolveAgentIdentityWorkspace(cfg *config.Config, agentID string, agent map[string]interface{}) string {
	if agent != nil {
		if workspace := strings.TrimSpace(toString(agent["workspace"])); workspace != "" {
			if filepath.IsAbs(workspace) {
				return filepath.Clean(workspace)
			}
			return filepath.Join(filepath.Dir(cfg.OpenClawDir), workspace)
		}
	}
	if workspace := resolveAgentWorkspacePath(cfg, agentID); workspace != "" {
		return workspace
	}
	return filepath.Join(filepath.Dir(cfg.OpenClawDir), "workspaces", agentID)
}

func resolveAgentIdentityRealWorkspaceRoot(workspaceRoot string) string {
	if realRoot, err := filepath.EvalSymlinks(workspaceRoot); err == nil && strings.TrimSpace(realRoot) != "" {
		return realRoot
	}
	return workspaceRoot
}

func resolveAgentIdentityRealCandidatePath(workspaceRoot, relativePath string) (string, string) {
	candidate := filepath.Clean(filepath.Join(workspaceRoot, relativePath))
	if realCandidate, err := filepath.EvalSymlinks(candidate); err == nil && strings.TrimSpace(realCandidate) != "" {
		return candidate, realCandidate
	}
	parent := filepath.Dir(candidate)
	if parent != "" && parent != candidate {
		if realParent, err := filepath.EvalSymlinks(parent); err == nil && strings.TrimSpace(realParent) != "" {
			return candidate, filepath.Join(realParent, filepath.Base(candidate))
		}
	}
	return candidate, candidate
}

func cleanAgentIdentityConfig(agent map[string]interface{}) map[string]interface{} {
	if agent == nil {
		return nil
	}
	rawIdentity, ok := agent["identity"]
	if !ok || rawIdentity == nil {
		return nil
	}
	identity, ok := rawIdentity.(map[string]interface{})
	if !ok {
		return nil
	}
	if trimStringField(identity["theme"]) == "" {
		for _, key := range agentIdentityLegacyThemeKeys {
			if value := trimStringField(identity[key]); value != "" {
				identity["theme"] = value
				break
			}
		}
	}
	cleaned := map[string]interface{}{}
	for key := range agentIdentityAllowedKeys {
		if value := trimStringField(identity[key]); value != "" {
			cleaned[key] = value
		}
	}
	if len(cleaned) == 0 {
		delete(agent, "identity")
		return nil
	}
	agent["identity"] = cleaned
	return cleaned
}

func validateAgentIdentityConfig(cfg *config.Config, agentID string, agent map[string]interface{}) error {
	if rawIdentity, ok := agent["identity"]; ok && rawIdentity != nil {
		if _, ok := rawIdentity.(map[string]interface{}); !ok {
			return fmt.Errorf("identity 必须是对象")
		}
	}
	identity := cleanAgentIdentityConfig(agent)
	if identity == nil {
		return nil
	}
	avatar := trimStringField(identity["avatar"])
	if avatar == "" {
		return nil
	}
	if isAgentAvatarDataURL(avatar) || isAgentAvatarHTTPURL(avatar) {
		return nil
	}
	if strings.HasPrefix(avatar, "~") || filepath.IsAbs(avatar) || hasAgentAvatarScheme(avatar) {
		return fmt.Errorf("identity.avatar 必须是工作区相对路径、http(s) URL 或 data URI")
	}
	workspace := resolveAgentIdentityWorkspace(cfg, agentID, agent)
	workspaceRoot, err := filepath.Abs(workspace)
	if err != nil {
		return fmt.Errorf("identity.avatar 工作区路径无效")
	}
	realWorkspaceRoot := resolveAgentIdentityRealWorkspaceRoot(workspaceRoot)
	_, realResolved := resolveAgentIdentityRealCandidatePath(workspaceRoot, avatar)
	if pathEscapesBase(realWorkspaceRoot, realResolved) {
		return fmt.Errorf("identity.avatar 必须位于 agent workspace 内")
	}
	return nil
}

func parseIdentityMarkdownContent(content string) map[string]string {
	identity := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		label := strings.ToLower(strings.Trim(strings.TrimSpace(parts[0]), "*_ "))
		value := strings.Trim(strings.TrimSpace(parts[1]), "*_ ")
		value = strings.Trim(value, "() ")
		if value == "" {
			continue
		}
		switch label {
		case "name", "emoji", "theme", "creature", "vibe", "avatar":
			identity[label] = value
		}
	}
	return identity
}

func loadIdentityFromWorkspaceFile(workspace string) map[string]string {
	if workspace == "" {
		return nil
	}
	content, err := os.ReadFile(filepath.Join(workspace, "IDENTITY.md"))
	if err != nil {
		return nil
	}
	parsed := parseIdentityMarkdownContent(string(content))
	if len(parsed) == 0 {
		return nil
	}
	return parsed
}

func resolveAgentIdentityAvatarValue(cfg *config.Config, agentID string) string {
	ocConfig, _ := cfg.ReadOpenClawJSON()
	if item := findAgentConfig(ocConfig, agentID); item != nil {
		if identity, ok := item["identity"].(map[string]interface{}); ok {
			if avatar := trimStringField(identity["avatar"]); avatar != "" {
				return avatar
			}
		}
	}
	workspace := resolveAgentIdentityWorkspace(cfg, agentID, findAgentConfig(ocConfig, agentID))
	if parsed := loadIdentityFromWorkspaceFile(workspace); len(parsed) > 0 {
		if avatar := strings.TrimSpace(parsed["avatar"]); avatar != "" {
			return avatar
		}
	}
	return ""
}

func resolveAgentIdentityAvatarFile(cfg *config.Config, agentID string) (string, string, error) {
	avatar := resolveAgentIdentityAvatarValue(cfg, agentID)
	if avatar == "" {
		return "", "", os.ErrNotExist
	}
	if isAgentAvatarHTTPURL(avatar) || isAgentAvatarDataURL(avatar) {
		return "", "", fmt.Errorf("identity.avatar 不是本地工作区文件")
	}
	ocConfig, _ := cfg.ReadOpenClawJSON()
	workspace := resolveAgentIdentityWorkspace(cfg, agentID, findAgentConfig(ocConfig, agentID))
	workspaceRoot, err := filepath.Abs(workspace)
	if err != nil {
		return "", "", err
	}
	realWorkspaceRoot := resolveAgentIdentityRealWorkspaceRoot(workspaceRoot)
	candidate, realResolved := resolveAgentIdentityRealCandidatePath(workspaceRoot, avatar)
	if pathEscapesBase(realWorkspaceRoot, realResolved) {
		return "", "", fmt.Errorf("identity.avatar 必须位于 agent workspace 内")
	}
	info, err := os.Lstat(candidate)
	if err != nil {
		return "", "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("identity.avatar 不能是符号链接")
	}
	info, err = os.Lstat(realResolved)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		return "", "", fmt.Errorf("identity.avatar 不能指向目录")
	}
	if info.Size() > agentAvatarMaxBytes {
		return "", "", fmt.Errorf("identity.avatar 文件过大")
	}
	ext := strings.ToLower(filepath.Ext(realResolved))
	mime := agentAvatarMimeByExt[ext]
	if mime == "" {
		return "", "", fmt.Errorf("identity.avatar 仅支持 png/jpg/jpeg/gif/webp/svg")
	}
	return realResolved, mime, nil
}

func GetOpenClawAgentIdentityAvatar(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		agentID := strings.TrimSpace(c.Param("id"))
		if err := validateAgentID(agentID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
			return
		}
		filePath, mime, err := resolveAgentIdentityAvatarFile(cfg, agentID)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "identity.avatar 本地文件不存在"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.Header("Content-Type", mime)
		c.File(filePath)
	}
}
