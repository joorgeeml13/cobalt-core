package drivers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
)

// HTTPDriver implementa DeviceDriver para dispositivos controlados vía HTTP local (REST, Smart TV, Tasmota HTTP).
type HTTPDriver struct {
	device *domain.Device
	client *http.Client
}

// NewHTTPDriver crea un nuevo driver HTTP.
func NewHTTPDriver(device *domain.Device) *HTTPDriver {
	return &HTTPDriver{
		device: device,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetState devuelve el estado cacheado en el dispositivo.
func (d *HTTPDriver) GetState(ctx context.Context) (map[string]any, error) {
	result := make(map[string]any)
	if len(d.device.State) > 0 {
		if err := json.Unmarshal(d.device.State, &result); err != nil {
			return nil, fmt.Errorf("deserializar estado del dispositivo HTTP %s: %w", d.device.Name, err)
		}
	}
	return result, nil
}

// ExecuteCommand envía una petición HTTP al dispositivo según la IP/URL de sus metadatos.
func (d *HTTPDriver) ExecuteCommand(ctx context.Context, action string, payload map[string]any) error {
	baseURL := ""
	if len(d.device.Meta) > 0 {
		var meta map[string]any
		if err := json.Unmarshal(d.device.Meta, &meta); err == nil {
			if u, ok := meta["url"].(string); ok {
				baseURL = u
			} else if ip, ok := meta["ip"].(string); ok {
				baseURL = fmt.Sprintf("http://%s", ip)
			}
		}
	}

	if baseURL == "" {
		return fmt.Errorf("dispositivo HTTP %s no tiene URL ni IP configurada en meta", d.device.Name)
	}

	targetURL := fmt.Sprintf("%s/api/%s", baseURL, action)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializar payload HTTP: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("crear petición HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("ejecutar comando HTTP en %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dispositivo HTTP respondió con código %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}
