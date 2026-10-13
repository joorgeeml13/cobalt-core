package mqtt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

// DeviceStateRouterHook es un hook personalizado de mochi-mqtt para interceptar eventos
// de conexión y publicaciones de dispositivos (ESP32, Zigbee2MQTT, etc.).
type DeviceStateRouterHook struct {
	mqtt.HookBase
	repo repository.DeviceRepository
}

// NewDeviceStateRouterHook crea una nueva instancia del hook de enrutamiento MQTT.
func NewDeviceStateRouterHook(repo repository.DeviceRepository) *DeviceStateRouterHook {
	return &DeviceStateRouterHook{
		repo: repo,
	}
}

// ID identifica de forma única este hook dentro del servidor mochi-mqtt.
func (h *DeviceStateRouterHook) ID() string {
	return "device-state-router"
}

// Provides declara qué eventos de ciclo de vida intercepta este hook.
func (h *DeviceStateRouterHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
		mqtt.OnPublish,
	}, []byte{b})
}

// OnConnect se ejecuta cuando un cliente MQTT se autentica/conecta.
func (h *DeviceStateRouterHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	log.Printf("[MQTT] Cliente conectado: ID=%s, RemoteAddr=%s", cl.ID, cl.Net.Remote)
	return nil
}

// OnDisconnect se ejecuta cuando un cliente MQTT pierde la conexión o se desconecta.
func (h *DeviceStateRouterHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	if err != nil {
		log.Printf("[MQTT] Cliente desconectado con error: ID=%s, error=%v", cl.ID, err)
	} else {
		log.Printf("[MQTT] Cliente desconectado limpiamente: ID=%s", cl.ID)
	}
}

// OnPublish se ejecuta cada vez que un mensaje es publicado en cualquier tópico.
// Si el mensaje encaja con tópicos de dispositivos, procesa el payload hacia la base de datos.
func (h *DeviceStateRouterHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	topic := pk.TopicName
	payload := pk.Payload

	// Formato esperado de tópicos estándar para IoT:
	// - home/{device_id_or_name}/state         -> Telemetría / nuevo estado JSON
	// - home/{device_id_or_name}/availability  -> Estado de conexión (online/offline)
	// - home/{device_id_or_name}/set           -> Comando hacia el actuador
	parts := strings.Split(topic, "/")
	if len(parts) >= 3 && parts[0] == "home" {
		deviceID := parts[1]
		action := parts[2]

		go h.handleDeviceMessage(deviceID, action, payload)
	}

	// Devolvemos el paquete inalterado para que el broker lo distribuya a los suscriptores
	return pk, nil
}

// handleDeviceMessage gestiona asíncronamente el guardado en base de datos.
func (h *DeviceStateRouterHook) handleDeviceMessage(identifier, action string, payload []byte) {
	if h.repo == nil {
		return
	}

	ctx := context.Background()

	switch action {
	case "availability", "status":
		statusStr := strings.TrimSpace(strings.ToLower(string(payload)))
		isOnline := statusStr == "online" || statusStr == "true" || statusStr == "1"
		if err := h.repo.SetOnline(ctx, identifier, isOnline); err != nil {
			// Es posible que el identificador sea un nombre o aún no esté insertado; lo registramos como debug
			log.Printf("[MQTT] No se pudo actualizar online para %s: %v", identifier, err)
		} else {
			log.Printf("[MQTT] Dispositivo %s marcado como online=%v", identifier, isOnline)
		}

	case "state":
		// Aseguramos que sea un JSON válido; si es un valor plano (ej: "23.5"), lo encapsulamos
		var rawState json.RawMessage
		if json.Valid(payload) {
			rawState = json.RawMessage(payload)
		} else {
			// Encapsular primitivos
			cleanVal := strings.TrimSpace(string(payload))
			formatted := fmt.Sprintf(`{"value": %q}`, cleanVal)
			rawState = json.RawMessage(formatted)
		}

		if err := h.repo.UpdateState(ctx, identifier, rawState); err != nil {
			// Si no existe, podemos auto-registrar el dispositivo si es un nuevo sensor
			log.Printf("[MQTT] Dispositivo %s no encontrado para actualizar estado (intentando auto-registro)", identifier)
			newDevice := &domain.Device{
				Name:     identifier,
				Protocol: domain.ProtocolMQTT,
				Type:     domain.TypeSensor,
				State:    rawState,
				Meta:     json.RawMessage(fmt.Sprintf(`{"topic_base": "home/%s"}`, identifier)),
				Online:   true,
			}
			if insertErr := h.repo.Insert(ctx, newDevice); insertErr != nil {
				log.Printf("[MQTT] Error al auto-registrar dispositivo %s: %v", identifier, insertErr)
			} else {
				log.Printf("[MQTT] Dispositivo %s auto-registrado con ID=%s", identifier, newDevice.ID)
			}
		} else {
			log.Printf("[MQTT] Estado actualizado para dispositivo %s: %s", identifier, string(rawState))
		}

	case "set":
		log.Printf("[MQTT] Comando recibido para %s: %s", identifier, string(payload))
	}
}
