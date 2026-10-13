package drivers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
)

// MQTTPublisher define la capacidad mínima requerida de un broker para publicar mensajes.
type MQTTPublisher interface {
	Publish(topic string, payload []byte, retain bool) error
}

// MQTTDriver implementa DeviceDriver para dispositivos controlados a través de MQTT (ESP32, Tasmota, Zigbee2MQTT).
type MQTTDriver struct {
	device    *domain.Device
	publisher MQTTPublisher
}

// NewMQTTDriver crea un driver para un dispositivo MQTT específico.
func NewMQTTDriver(device *domain.Device, publisher MQTTPublisher) *MQTTDriver {
	return &MQTTDriver{
		device:    device,
		publisher: publisher,
	}
}

// GetState devuelve el estado cacheado o persistido del dispositivo como mapa clave-valor.
func (d *MQTTDriver) GetState(ctx context.Context) (map[string]any, error) {
	result := make(map[string]any)
	if len(d.device.State) > 0 {
		if err := json.Unmarshal(d.device.State, &result); err != nil {
			return nil, fmt.Errorf("deserializar estado del dispositivo MQTT %s: %w", d.device.Name, err)
		}
	}
	return result, nil
}

// ExecuteCommand envía un comando publicando un mensaje en el tópico MQTT correspondiente.
// Por defecto publica en: home/{name}/set (o en el tópico configurado en Meta).
func (d *MQTTDriver) ExecuteCommand(ctx context.Context, action string, payload map[string]any) error {
	if d.publisher == nil {
		return fmt.Errorf("publicador MQTT no disponible")
	}

	// Extraer tópico base opcional de metadata
	baseTopic := fmt.Sprintf("home/%s", d.device.Name)
	if len(d.device.Meta) > 0 {
		var meta map[string]any
		if err := json.Unmarshal(d.device.Meta, &meta); err == nil {
			if t, ok := meta["topic_set"].(string); ok && t != "" {
				baseTopic = t
			} else if t, ok := meta["topic"].(string); ok && t != "" {
				baseTopic = fmt.Sprintf("%s/set", t)
			}
		}
	}

	topic := baseTopic
	if action != "" && action != "set" {
		topic = fmt.Sprintf("%s/%s", baseTopic, action)
	} else if action == "set" || action == "" {
		topic = fmt.Sprintf("%s/set", baseTopic)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializar payload de comando MQTT: %w", err)
	}

	if err := d.publisher.Publish(topic, payloadBytes, false); err != nil {
		return fmt.Errorf("publicar comando MQTT en tópico %s: %w", topic, err)
	}

	return nil
}
