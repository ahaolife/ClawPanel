package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhaoxinyi02/ClawPanel/internal/plugin"
)

// GetPluginList returns all installed plugins + registry plugins
func GetPluginList(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		installed := pm.ListInstalled()
		reg := pm.GetRegistry()
		if len(reg.Plugins) == 0 {
			if fetched, err := pm.FetchRegistry(); err == nil && fetched != nil {
				reg = fetched
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":        true,
			"installed": installed,
			"registry":  reg.Plugins,
		})
	}
}

// GetInstalledPlugins returns only installed plugins
func GetInstalledPlugins(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":      true,
			"plugins": pm.ListInstalled(),
		})
	}
}

// GetPluginDetail returns a single plugin's details
func GetPluginDetail(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		p := pm.GetPlugin(id)
		if p == nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "插件未安装"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "plugin": p})
	}
}

// RefreshPluginRegistry fetches the latest registry
func RefreshPluginRegistry(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		reg, err := pm.FetchRegistry()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"registry": reg,
		})
	}
}

// InstallPlugin installs a plugin from registry or custom URL
func InstallPlugin(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			PluginID string `json:"pluginId"`
			Source   string `json:"source,omitempty"` // custom git/archive URL
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.PluginID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "pluginId required"})
			return
		}

		// Check conflicts first
		conflicts := pm.CheckConflicts(req.PluginID)
		if len(conflicts) > 0 {
			c.JSON(http.StatusConflict, gin.H{"ok": false, "error": conflicts[0], "conflicts": conflicts})
			return
		}

		if err := pm.Install(req.PluginID, req.Source); err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "插件安装成功"})
	}
}

// UninstallPlugin removes a plugin
func UninstallPlugin(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := pm.Uninstall(id); err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "插件已卸载"})
	}
}

// TogglePlugin enables or disables a plugin
func TogglePlugin(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "参数错误"})
			return
		}

		var err error
		if req.Enabled {
			err = pm.Enable(id)
		} else {
			err = pm.Disable(id)
		}
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetPluginConfig returns a plugin's configuration and schema
func GetPluginConfig(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		cfg, schema, err := pm.GetConfig(id)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"config": cfg,
			"schema": schema,
		})
	}
}

// UpdatePluginConfig updates a plugin's configuration
func UpdatePluginConfig(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var cfg map[string]interface{}
		if err := c.ShouldBindJSON(&cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "参数错误"})
			return
		}

		if err := pm.UpdateConfig(id, cfg); err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetPluginLogs returns a plugin's log output
func GetPluginLogs(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logs, err := pm.GetPluginLogs(id)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "logs": logs})
	}
}

// UpdatePlugin updates a plugin to the latest version
func UpdatePluginVersion(pm *plugin.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := pm.Update(id); err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "插件更新成功"})
	}
}
