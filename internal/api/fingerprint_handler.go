package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/pkg/fingerprint"
)

// ClientHintsDTO mirrors fingerprint.ClientHints for API JSON serialization.
type ClientHintsDTO struct {
	UA              string `json:"ua"`
	Platform        string `json:"platform"`
	Mobile          string `json:"mobile"`
	PlatformVersion string `json:"platformVersion"`
}

// FingerprintResponse represents the response payload for provider fingerprint endpoints.
type FingerprintResponse struct {
	ProviderID  string            `json:"providerId"`
	UserAgent   string            `json:"userAgent"`
	ClientHints *ClientHintsDTO   `json:"clientHints"`
	Cookies     map[string]string `json:"cookies"`
	TLSProfile  string            `json:"tlsProfile"`
}

// FingerprintRequest represents the request payload for setting a provider fingerprint.
type FingerprintRequest struct {
	UserAgent   string            `json:"userAgent"`
	ClientHints *ClientHintsDTO   `json:"clientHints"`
	Cookies     map[string]string `json:"cookies"`
	TLSProfile  string            `json:"tlsProfile"`
}

func profileToResponse(providerID string, prof fingerprint.Profile) FingerprintResponse {
	resp := FingerprintResponse{
		ProviderID: providerID,
		UserAgent:  prof.UserAgent,
		Cookies:    prof.Cookies,
		TLSProfile: string(prof.TLSProfile),
	}
	if prof.ClientHints != nil {
		resp.ClientHints = &ClientHintsDTO{
			UA:              prof.ClientHints.UA,
			Platform:        prof.ClientHints.Platform,
			Mobile:          prof.ClientHints.Mobile,
			PlatformVersion: prof.ClientHints.PlatformVersion,
		}
	}
	return resp
}

func requestToProfile(req FingerprintRequest) fingerprint.Profile {
	prof := fingerprint.Profile{
		UserAgent:  req.UserAgent,
		Cookies:    req.Cookies,
		TLSProfile: fingerprint.TLSProfile(req.TLSProfile),
	}
	if req.ClientHints != nil {
		prof.ClientHints = &fingerprint.ClientHints{
			UA:              req.ClientHints.UA,
			Platform:        req.ClientHints.Platform,
			Mobile:          req.ClientHints.Mobile,
			PlatformVersion: req.ClientHints.PlatformVersion,
		}
	}
	return prof
}

func (h *Handler) handleGetFingerprint(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		c.Set("handler_error", "providerId is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "providerId is required"})
	}

	if h.fpStore == nil {
		c.Set("handler_error", fingerprint.ErrUnknownSource.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": fingerprint.ErrUnknownSource.Error()})
	}

	prof, err := h.fpStore.Get(providerID)
	if err != nil {
		c.Set("handler_error", err.Error())
		if errors.Is(err, fingerprint.ErrUnknownSource) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, profileToResponse(providerID, prof))
}

func (h *Handler) handlePutFingerprint(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		c.Set("handler_error", "providerId is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "providerId is required"})
	}

	var req FingerprintRequest
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	prof := requestToProfile(req)
	if h.fpStore == nil {
		h.fpStore = fingerprint.NewMemoryStore()
	}

	if err := h.fpStore.Set(providerID, prof); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, profileToResponse(providerID, prof))
}

func (h *Handler) handleDeleteFingerprint(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		c.Set("handler_error", "providerId is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "providerId is required"})
	}

	if h.fpStore != nil {
		if err := h.fpStore.Delete(providerID); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
	}

	return c.NoContent(http.StatusNoContent)
}
