package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/matter"
	"github.com/joorgeeml13/cobalt-core/internal/mqtt"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

type DiscoveryHandler struct {
	repo         repository.DeviceRepository
	mqttBroker   *mqtt.Broker
	matterClient *matter.Client
}

func NewDiscoveryHandler(repo repository.DeviceRepository, mqttBroker *mqtt.Broker, matterClient *matter.Client) *DiscoveryHandler {
	return &DiscoveryHandler{
		repo:         repo,
		mqttBroker:   mqttBroker,
		matterClient: matterClient,
	}
}

type CommissionMatterRequest struct {
	Code      string `json:"code"`
	NetworkSSID string `json:"ssid,omitempty"`
	Password  string `json:"password,omitempty"`
}

// CommissionMatter maneja POST /api/devices/commission/matter
func (h *DiscoveryHandler) CommissionMatter(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var req CommissionMatterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_BODY", "Cuerpo JSON inválido", err.Error())
		return
	}

	if req.Code == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "El código numérico o setup code de Matter es requerido", nil)
		return
	}

	if h.matterClient == nil {
		Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Cliente Matter no disponible", nil)
		return
	}

	args := map[string]any{
		"code": req.Code,
	}
	if req.NetworkSSID != "" {
		args["network_ssid"] = req.NetworkSSID
		args["network_password"] = req.Password
	}

	resp, err := h.matterClient.SendCommand(ctx, "commission_with_code", args)
	if err != nil {
		Error(w, http.StatusInternalServerError, "COMMISSION_FAILED", "Fallo al iniciar el comisionado Matter", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"message": "Comisionado iniciado en python-matter-server",
		"result":  resp.Result,
	})
}

// DiscoverMatter maneja GET /api/discovery/matter
func (h *DiscoveryHandler) DiscoverMatter(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if h.matterClient == nil {
		Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Cliente Matter no disponible", nil)
		return
	}

	resp, err := h.matterClient.SendCommand(ctx, "get_nodes", nil)
	if err != nil {
		Error(w, http.StatusInternalServerError, "DISCOVERY_FAILED", "Error al consultar nodos en python-matter-server", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"nodes": resp.Result,
	})
}

type DiscoverMQTTRequest struct {
	TopicBase string `json:"topic_base"` // Opcional, default "home/#"
}

// DiscoverMQTT maneja POST /api/devices/discover/mqtt
func (h *DiscoveryHandler) DiscoverMQTT(w http.ResponseWriter, r *http.Request) {
	// Publica solicitud de descubrimiento en el broker si los dispositivos soportan broadcast
	if h.mqttBroker != nil {
		_ = h.mqttBroker.Publish("home/discovery/poll", []byte(`{"discover": true}`), false)
	}

	JSON(w, http.StatusOK, map[string]any{
		"message": "Sondeo de descubrimiento MQTT emitido a los dispositivos activos",
		"topics":  []string{"home/+/state", "home/+/availability"},
	})
}

// InspectMQTT maneja GET /api/discovery/mqtt/inspect
func (h *DiscoveryHandler) InspectMQTT(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mqttDevices, err := h.repo.ListByProtocol(ctx, domain.ProtocolMQTT)
	if err != nil {
		Error(w, http.StatusInternalServerError, "DB_ERROR", "Error al listar dispositivos MQTT activos", err.Error())
		return
	}

	activeTopics := make([]string, 0, len(mqttDevices))
	for _, dev := range mqttDevices {
		activeTopics = append(activeTopics, fmt.Sprintf("home/%s/state", dev.Name))
	}

	JSON(w, http.StatusOK, map[string]any{
		"broker_status": "running",
		"devices_count": len(mqttDevices),
		"active_topics": activeTopics,
	})
}
