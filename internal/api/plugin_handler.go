package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/plugin/host"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// PluginItemDTO describes an installed or running plugin binary and its contained providers.
type PluginItemDTO struct {
	PluginID             string                     `json:"pluginId"`
	PluginName           string                     `json:"pluginName"`
	PluginVersion        string                     `json:"pluginVersion"`
	SDKVersion           string                     `json:"sdkVersion"`
	SDKCompatible        bool                       `json:"sdkCompatible"`
	ExecutablePath       string                     `json:"executablePath"`
	PID                  int                        `json:"pid"`
	State                host.PluginState           `json:"state"`
	ErrorMessage         string                     `json:"errorMessage,omitempty"`
	LoadedAt             time.Time                  `json:"loadedAt"`
	Providers            []sdk.ProviderDescriptor   `json:"providers"`
	PluginSettingsSchema []sdk.SettingSpec          `json:"pluginSettingsSchema,omitempty"`
	GlobalConfig         map[string]string          `json:"globalConfig,omitempty"`
	ProviderConfigs      map[string]map[string]string `json:"providerConfigs,omitempty"`
}

// ReloadResponseDTO summarizes the outcome of a plugin reload operation.
type ReloadResponseDTO struct {
	Status          string   `json:"status"`
	Message         string   `json:"message"`
	ReloadedPlugins []string `json:"reloadedPlugins"`
	ActiveProviders int      `json:"activeProviders"`
}

// UpdatePluginConfigRequest carries scoped settings updates for a plugin.
type UpdatePluginConfigRequest struct {
	GlobalConfig    map[string]string            `json:"globalConfig,omitempty"`
	ProviderConfigs map[string]map[string]string `json:"providerConfigs,omitempty"`
}

// CandidateDTO represents a provider implementation candidate.
type CandidateDTO struct {
	PluginID  string `json:"pluginId"`
	Version   string `json:"version"`
	IsBuiltIn bool   `json:"isBuiltIn"`
	Selected  bool   `json:"selected"`
}

// ProviderCollisionDTO describes a detected provider ID collision with resolution state.
type ProviderCollisionDTO struct {
	ProviderID string         `json:"providerId"`
	Selected   string         `json:"selected"` // "builtin" or pluginID
	Candidates []CandidateDTO `json:"candidates"`
}

// SetPreferenceRequest specifies user preference for collision resolution.
type SetPreferenceRequest struct {
	ProviderID string `json:"providerId"`
	Preference string `json:"preference"`
}

// listPlugins handles GET /api/v1/plugins
func (h *Handler) listPlugins(c echo.Context) error {
	if h.pluginManager == nil {
		return c.JSON(http.StatusOK, []PluginItemDTO{})
	}

	statuses := h.pluginManager.ListPlugins()
	items := make([]PluginItemDTO, 0, len(statuses))

	for _, s := range statuses {
		sdkCompatible := host.CheckSDKCompatibility(sdk.Version, s.SDKVersion) == nil
		globalCfg, provCfgs := h.pluginManager.GetPluginConfig(s.PluginID)

		items = append(items, PluginItemDTO{
			PluginID:             s.PluginID,
			PluginName:           s.PluginName,
			PluginVersion:        s.PluginVersion,
			SDKVersion:           s.SDKVersion,
			SDKCompatible:        sdkCompatible,
			ExecutablePath:       s.ExecutablePath,
			PID:                  s.PID,
			State:                s.State,
			ErrorMessage:         s.ErrorMessage,
			LoadedAt:             s.LoadedAt,
			Providers:            s.Providers,
			PluginSettingsSchema: s.PluginSettingsSchema,
			GlobalConfig:         globalCfg,
			ProviderConfigs:      provCfgs,
		})
	}

	return c.JSON(http.StatusOK, items)
}

// reloadPlugins handles POST /api/v1/plugins/reload
func (h *Handler) reloadPlugins(c echo.Context) error {
	if h.pluginManager == nil {
		return c.JSON(http.StatusOK, ReloadResponseDTO{
			Status:          "ok",
			Message:         "No plugin manager active",
			ReloadedPlugins: []string{},
			ActiveProviders: len(h.registry.ListInfo()),
		})
	}

	ctx := c.Request().Context()
	if err := h.pluginManager.ReloadAll(ctx); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "failed to reload plugins",
			"details": err.Error(),
		})
	}

	statuses := h.pluginManager.ListPlugins()
	reloaded := make([]string, 0, len(statuses))
	for _, s := range statuses {
		reloaded = append(reloaded, s.PluginID)
	}

	activeCount := len(h.registry.ListInfo())
	return c.JSON(http.StatusOK, ReloadResponseDTO{
		Status:          "ok",
		Message:         fmt.Sprintf("Reloaded %d plugin(s) successfully", len(reloaded)),
		ReloadedPlugins: reloaded,
		ActiveProviders: activeCount,
	})
}

// getPluginLogs handles GET /api/v1/plugins/:id/logs
func (h *Handler) getPluginLogs(c echo.Context) error {
	pluginID := c.Param("id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	if h.pluginManager == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("plugin %q not found", pluginID),
		})
	}

	if _, ok := h.pluginManager.GetPluginStatus(pluginID); !ok {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("plugin %q not found", pluginID),
		})
	}

	logs := h.pluginManager.GetPluginLogs(pluginID)
	if logs == nil {
		logs = []host.PluginLogEntry{}
	}

	return c.JSON(http.StatusOK, logs)
}

// updatePluginConfig handles POST /api/v1/plugins/:id/config
func (h *Handler) updatePluginConfig(c echo.Context) error {
	pluginID := c.Param("id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	if h.pluginManager == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("plugin %q not found", pluginID),
		})
	}

	if _, ok := h.pluginManager.GetPluginStatus(pluginID); !ok {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("plugin %q not found", pluginID),
		})
	}

	var req UpdatePluginConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid request body",
			"details": err.Error(),
		})
	}

	ctx := c.Request().Context()
	if err := h.pluginManager.UpdatePluginConfig(ctx, pluginID, req.GlobalConfig, req.ProviderConfigs); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "failed to update plugin config",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status":   "ok",
		"message":  fmt.Sprintf("Configuration updated for plugin %q", pluginID),
		"pluginId": pluginID,
	})
}

// listCollisions handles GET /api/v1/plugins/collisions
func (h *Handler) listCollisions(c echo.Context) error {
	if h.registry == nil {
		return c.JSON(http.StatusOK, []ProviderCollisionDTO{})
	}

	candidatesMap := h.registry.Candidates()
	collisions := make([]ProviderCollisionDTO, 0)

	for baseID, cands := range candidatesMap {
		if len(cands) <= 1 {
			continue
		}

		// Determine which candidate is currently winning
		activeWinner, _ := h.registry.Get(baseID)
		activePluginID := ""
		activeIsBuiltin := false
		if activeWinner != nil {
			if pi, ok := activeWinner.(interface{ PluginID() string }); ok {
				activePluginID = pi.PluginID()
			}
			if bi, ok := activeWinner.(interface{ IsBuiltIn() bool }); ok {
				activeIsBuiltin = bi.IsBuiltIn()
			} else if activePluginID == "" {
				activeIsBuiltin = true
			}
		}

		selected := "builtin"
		if !activeIsBuiltin && activePluginID != "" {
			selected = activePluginID
		}

		candDTOs := make([]CandidateDTO, 0, len(cands))
		for _, cand := range cands {
			isSelected := false
			if cand.IsBuiltIn && selected == "builtin" {
				isSelected = true
			} else if !cand.IsBuiltIn && cand.PluginID == selected {
				isSelected = true
			}

			id := cand.PluginID
			if cand.IsBuiltIn {
				id = "builtin"
			}

			candDTOs = append(candDTOs, CandidateDTO{
				PluginID:  id,
				Version:   cand.Version,
				IsBuiltIn: cand.IsBuiltIn,
				Selected:  isSelected,
			})
		}

		collisions = append(collisions, ProviderCollisionDTO{
			ProviderID: baseID,
			Selected:   selected,
			Candidates: candDTOs,
		})
	}

	return c.JSON(http.StatusOK, collisions)
}

// setPluginPreference handles POST /api/v1/plugins/preference
func (h *Handler) setPluginPreference(c echo.Context) error {
	var req SetPreferenceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid request body",
			"details": err.Error(),
		})
	}

	if req.ProviderID == "" || req.Preference == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "providerId and preference are required",
		})
	}

	if h.registry != nil {
		h.registry.SetUserPreference(req.ProviderID, req.Preference)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status":     "ok",
		"message":    fmt.Sprintf("Preference for provider %q set to %q", req.ProviderID, req.Preference),
		"providerId": req.ProviderID,
		"preference": req.Preference,
	})
}
