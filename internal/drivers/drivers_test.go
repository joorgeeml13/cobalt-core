package drivers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/drivers"
)

// mockPublisher simula el broker MQTT.
type mockPublisher struct {
	lastTopic   string
	lastPayload []byte
}

func (m *mockPublisher) Publish(topic string, payload []byte, retain bool) error {
	m.lastTopic = topic
	m.lastPayload = payload
	return nil
}

// mockCommander simula el cliente Matter.
type mockCommander struct {
	lastNodeID   uint64
	lastEndpoint uint
	lastCluster  uint
	lastCommand  string
	lastPayload  map[string]any
}

func (m *mockCommander) SetDeviceState(ctx context.Context, nodeID uint64, endpoint uint, cluster uint, command string, payload map[string]any) error {
	m.lastNodeID = nodeID
	m.lastEndpoint = endpoint
	m.lastCluster = cluster
	m.lastCommand = command
	m.lastPayload = payload
	return nil
}

func TestDriversAndRegistry(t *testing.T) {
	ctx := context.Background()
	reg := drivers.NewRegistry(nil)

	pub := &mockPublisher{}
	cmd := &mockCommander{}

	// Registrar factories
	reg.RegisterDriverFactory(domain.ProtocolMQTT, func(d *domain.Device) (drivers.DeviceDriver, error) {
		return drivers.NewMQTTDriver(d, pub), nil
	})
	reg.RegisterDriverFactory(domain.ProtocolMatter, func(d *domain.Device) (drivers.DeviceDriver, error) {
		return drivers.NewMatterDriver(d, cmd), nil
	})

	// 1. Probar Driver MQTT
	mqttDev := &domain.Device{
		ID:       "dev-mqtt-1",
		Name:     "luz_salon",
		Protocol: domain.ProtocolMQTT,
		Type:     domain.TypeLight,
		State:    json.RawMessage(`{"state": "ON"}`),
	}

	mqttDriver, err := reg.GetDriver(mqttDev)
	if err != nil {
		t.Fatalf("GetDriver MQTT falló: %v", err)
	}

	state, err := mqttDriver.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState MQTT falló: %v", err)
	}
	if state["state"] != "ON" {
		t.Errorf("Estado inesperado: %v", state)
	}

	err = mqttDriver.ExecuteCommand(ctx, "set", map[string]any{"state": "OFF"})
	if err != nil {
		t.Fatalf("ExecuteCommand MQTT falló: %v", err)
	}
	if pub.lastTopic != "home/luz_salon/set" {
		t.Errorf("Tópico MQTT incorrecto: %s", pub.lastTopic)
	}

	// 2. Probar Driver Matter
	matterDev := &domain.Device{
		ID:       "dev-matter-1",
		Name:     "bombilla_rgb",
		Protocol: domain.ProtocolMatter,
		Type:     domain.TypeLight,
		Meta:     json.RawMessage(`{"node_id": 42, "endpoint_id": 1}`),
		State:    json.RawMessage(`{"on": true, "brightness": 100}`),
	}

	matterDriver, err := reg.GetDriver(matterDev)
	if err != nil {
		t.Fatalf("GetDriver Matter falló: %v", err)
	}

	err = matterDriver.ExecuteCommand(ctx, "turn_off", nil)
	if err != nil {
		t.Fatalf("ExecuteCommand Matter falló: %v", err)
	}
	if cmd.lastNodeID != 42 || cmd.lastCommand != "Off" || cmd.lastCluster != drivers.ClusterOnOff {
		t.Errorf("Comando Matter incorrecto: Node=%d Cmd=%s Cluster=%d", cmd.lastNodeID, cmd.lastCommand, cmd.lastCluster)
	}

	// 3. Probar protocolo no registrado
	unknownDev := &domain.Device{
		Protocol: domain.Protocol("desconocido"),
	}
	_, err = reg.GetDriver(unknownDev)
	if err == nil {
		t.Error("Se esperaba error al buscar driver para protocolo desconocido")
	}
}
