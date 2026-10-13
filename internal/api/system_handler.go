package api

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemHandler struct {
	pool      *pgxpool.Pool
	startTime time.Time
}

func NewSystemHandler(pool *pgxpool.Pool) *SystemHandler {
	return &SystemHandler{
		pool:      pool,
		startTime: time.Now(),
	}
}

// Health maneja GET /api/health
func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if h.pool != nil {
		if err := h.pool.Ping(ctx); err != nil {
			dbStatus = "error: " + err.Error()
		}
	} else {
		dbStatus = "not_configured"
	}

	uptime := time.Since(h.startTime).Round(time.Second).String()

	JSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"uptime":   uptime,
		"database": dbStatus,
		"services": map[string]string{
			"http":   "online",
			"mqtt":   "online",
			"matter": "active",
		},
	})
}

// Protocols maneja GET /api/protocols
func (h *SystemHandler) Protocols(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{
		"supported": []map[string]any{
			{
				"protocol":    "mqtt",
				"transport":   "tcp:1883",
				"description": "Embedded Mochi MQTT broker (Zigbee2MQTT, ESP32, Tasmota)",
				"status":      "enabled",
			},
			{
				"protocol":    "matter",
				"transport":   "websocket",
				"description": "python-matter-server bridge (Thread, Wi-Fi Matter devices)",
				"status":      "enabled",
			},
			{
				"protocol":    "http",
				"transport":   "rest/json",
				"description": "Local HTTP endpoints (Smart TVs, Tasmota HTTP, REST bridges)",
				"status":      "enabled",
			},
		},
	})
}
