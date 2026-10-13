package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/drivers"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

type DeviceHandler struct {
	repo     repository.DeviceRepository
	registry *drivers.Registry
}

func NewDeviceHandler(repo repository.DeviceRepository, registry *drivers.Registry) *DeviceHandler {
	return &DeviceHandler{
		repo:     repo,
		registry: registry,
	}
}

// ListDevices maneja GET /api/devices (con filtro opcional ?protocol=... y ?type=...)
func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	protocol := r.URL.Query().Get("protocol")

	var devices []domain.Device
	var err error

	if protocol != "" {
		devices, err = h.repo.ListByProtocol(ctx, domain.Protocol(protocol))
	} else {
		devices, err = h.repo.ListAll(ctx)
	}

	if err != nil {
		Error(w, http.StatusInternalServerError, "DB_ERROR", "Error al listar dispositivos", err.Error())
		return
	}

	if devices == nil {
		devices = []domain.Device{}
	}

	JSON(w, http.StatusOK, map[string]any{
		"devices": devices,
		"count":   len(devices),
	})
}

// GetDevice maneja GET /api/devices/{id}
func (h *DeviceHandler) GetDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	dev, err := h.repo.GetByID(ctx, id)
	if err != nil {
		Error(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Dispositivo no encontrado", err.Error())
		return
	}

	JSON(w, http.StatusOK, dev)
}

type CreateDeviceRequest struct {
	Name     string          `json:"name"`
	Protocol domain.Protocol `json:"protocol"`
	Type     domain.DeviceType `json:"type"`
	State    json.RawMessage `json:"state,omitempty"`
	Meta     json.RawMessage `json:"meta,omitempty"`
}

// CreateDevice maneja POST /api/devices
func (h *DeviceHandler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_BODY", "Cuerpo de petición no es JSON válido", err.Error())
		return
	}

	if req.Name == "" || req.Protocol == "" || req.Type == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Campos 'name', 'protocol' y 'type' son requeridos", nil)
		return
	}

	if len(req.State) == 0 {
		req.State = json.RawMessage(`{}`)
	}
	if len(req.Meta) == 0 {
		req.Meta = json.RawMessage(`{}`)
	}

	device := &domain.Device{
		Name:     req.Name,
		Protocol: req.Protocol,
		Type:     req.Type,
		State:    req.State,
		Meta:     req.Meta,
		Online:   true,
	}

	if err := h.repo.Insert(ctx, device); err != nil {
		Error(w, http.StatusInternalServerError, "DB_ERROR", "Error al registrar dispositivo", err.Error())
		return
	}

	JSON(w, http.StatusCreated, device)
}

// DeleteDevice maneja DELETE /api/devices/{id}
func (h *DeviceHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(ctx, id); err != nil {
		Error(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Dispositivo no encontrado para eliminar", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"message": "Dispositivo eliminado correctamente",
		"id":      id,
	})
}

type CommandRequest struct {
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload"`
}

// ExecuteCommand maneja POST /api/devices/{id}/command
func (h *DeviceHandler) ExecuteCommand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var cmd CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_BODY", "Cuerpo de petición no es JSON válido", err.Error())
		return
	}

	if cmd.Action == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "El campo 'action' es obligatorio", nil)
		return
	}

	if cmd.Payload == nil {
		cmd.Payload = make(map[string]any)
	}

	if err := h.registry.ExecuteCommandByID(ctx, id, cmd.Action, cmd.Payload); err != nil {
		Error(w, http.StatusInternalServerError, "DRIVER_ERROR", "Error al ejecutar comando en el driver del dispositivo", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"message": "Comando enviado al dispositivo",
		"id":      id,
		"action":  cmd.Action,
		"payload": cmd.Payload,
	})
}
