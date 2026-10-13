package drivers

import (
	"context"
)

// DeviceDriver es la interfaz unificada que todo dispositivo debe satisfacer,
// independientemente de si habla MQTT, Matter o HTTP local.
type DeviceDriver interface {
	// GetState obtiene el estado actual del dispositivo (temperatura, on/off, brillo, etc.).
	GetState(ctx context.Context) (map[string]any, error)

	// ExecuteCommand ejecuta una acción sobre el dispositivo (ej. "turn_on", "set_brightness", "set_channel")
	// junto con los parámetros requeridos.
	ExecuteCommand(ctx context.Context, action string, payload map[string]any) error
}
