package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// CacheStatsResponse represents the JSON response for cache stats.
type CacheStatsResponse struct {
	SizeBytes int64 `json:"size_bytes"`
	ItemCount int   `json:"item_count"`
}

// ClearCacheResponse represents the JSON response for clear cache.
type ClearCacheResponse struct {
	Status string `json:"status"`
}

// getCacheStats handles GET /api/v1/system/cache.
func (h *Handler) getCacheStats(c echo.Context) error {
	if h.imageCache == nil {
		return c.JSON(http.StatusOK, CacheStatsResponse{
			SizeBytes: 0,
			ItemCount: 0,
		})
	}

	sizeBytes, itemCount, err := h.imageCache.Stats()
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve cache stats"})
	}

	return c.JSON(http.StatusOK, CacheStatsResponse{
		SizeBytes: sizeBytes,
		ItemCount: itemCount,
	})
}

// clearCache handles POST /api/v1/system/cache/clear.
func (h *Handler) clearCache(c echo.Context) error {
	if h.imageCache == nil {
		return c.JSON(http.StatusOK, ClearCacheResponse{
			Status: "ok",
		})
	}

	if err := h.imageCache.Clear(); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to clear cache"})
	}

	return c.JSON(http.StatusOK, ClearCacheResponse{
		Status: "ok",
	})
}
