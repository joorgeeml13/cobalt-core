package domain

import (
	"encoding/json"
	"time"
)

// Protocol representa el protocolo de comunicación de un dispositivo.
type Protocol string

// DeviceType representa la categoría funcional de un dispositivo.
type DeviceType string

const (
	ProtocolMQTT   Protocol = "mqtt"
	ProtocolMatter Protocol = "matter"
	ProtocolHTTP   Protocol = "http"

	TypeSensor DeviceType = "sensor"
	TypeLight  DeviceType = "light"
	TypeTV     DeviceType = "tv"
)

// Device es la entidad central del dominio Cobalt. Representa cualquier dispositivo
// del hogar independientemente de su protocolo de comunicación.
//
// El campo State almacena el estado actual del dispositivo como JSON arbitrario,
// permitiendo representar datos heterogéneos:
//   - Sensor MQTT:     {"temperature": 22.5, "humidity": 65, "battery": 87}
//   - Bombilla Matter: {"on": true, "brightness": 80, "color_temp": 4000}
//   - Smart TV:        {"power": true, "input": "HDMI1", "channel": 5, "volume": 30}
//
// El campo Meta almacena la configuración inmutable del dispositivo
// (topic MQTT, IP, matter node ID, etc.).
type Device struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Protocol  Protocol        `json:"protocol"`
	Type      DeviceType      `json:"type"`
	State     json.RawMessage `json:"state"`
	Meta      json.RawMessage `json:"meta"`
	Online    bool            `json:"online"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
