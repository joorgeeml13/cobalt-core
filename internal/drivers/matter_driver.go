package drivers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
)

// MatterCommander define el contrato necesario para enviar órdenes a Matter.
type MatterCommander interface {
	SetDeviceState(ctx context.Context, nodeID uint64, endpoint uint, cluster uint, command string, payload map[string]any) error
}

// Constantes estándar de Clusters de Matter (Zigbee/Matter Specification)
const (
	ClusterOnOff        uint = 6  // 0x0006 On/Off
	ClusterLevelControl uint = 8  // 0x0008 Level Control (Brillo)
	ClusterColorControl uint = 768 // 0x0300 Color Control
)

// MatterDriver implementa DeviceDriver interactuando con python-matter-server.
type MatterDriver struct {
	device    *domain.Device
	commander MatterCommander
}

// NewMatterDriver crea un nuevo driver para dispositivos Matter.
func NewMatterDriver(device *domain.Device, commander MatterCommander) *MatterDriver {
	return &MatterDriver{
		device:    device,
		commander: commander,
	}
}

// GetState devuelve el estado actual decodificado de la tabla devices.
func (d *MatterDriver) GetState(ctx context.Context) (map[string]any, error) {
	result := make(map[string]any)
	if len(d.device.State) > 0 {
		if err := json.Unmarshal(d.device.State, &result); err != nil {
			return nil, fmt.Errorf("deserializar estado del dispositivo Matter %s: %w", d.device.Name, err)
		}
	}
	return result, nil
}

// ExecuteCommand traduce acciones genéricas (turn_on, turn_off, toggle, set_brightness)
// a llamadas de clusters Matter hacia python-matter-server.
func (d *MatterDriver) ExecuteCommand(ctx context.Context, action string, payload map[string]any) error {
	if d.commander == nil {
		return fmt.Errorf("comandante Matter no configurado")
	}

	// Extraer node_id y endpoint de metadata
	nodeID := uint64(1)
	endpoint := uint(1)

	if len(d.device.Meta) > 0 {
		var meta map[string]any
		if err := json.Unmarshal(d.device.Meta, &meta); err == nil {
			if n, ok := meta["node_id"].(float64); ok {
				nodeID = uint64(n)
			}
			if ep, ok := meta["endpoint_id"].(float64); ok {
				endpoint = uint(ep)
			}
		}
	}

	switch action {
	case "turn_on", "on":
		return d.commander.SetDeviceState(ctx, nodeID, endpoint, ClusterOnOff, "On", payload)

	case "turn_off", "off":
		return d.commander.SetDeviceState(ctx, nodeID, endpoint, ClusterOnOff, "Off", payload)

	case "toggle":
		return d.commander.SetDeviceState(ctx, nodeID, endpoint, ClusterOnOff, "Toggle", payload)

	case "set_brightness", "brightness":
		return d.commander.SetDeviceState(ctx, nodeID, endpoint, ClusterLevelControl, "MoveToLevel", payload)

	case "set_color", "color":
		return d.commander.SetDeviceState(ctx, nodeID, endpoint, ClusterColorControl, "MoveToColor", payload)

	default:
		// Comando directo de cluster
		cluster := ClusterOnOff
		if c, ok := payload["cluster_id"].(float64); ok {
			cluster = uint(c)
		}
		return d.commander.SetDeviceState(ctx, nodeID, endpoint, cluster, action, payload)
	}
}
