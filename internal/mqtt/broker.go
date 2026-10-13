package mqtt

import (
	"fmt"
	"log"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"

	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

// Broker encapsula la instancia de mochi-mqtt/server/v2 y su configuración.
type Broker struct {
	server *mqtt.Server
	port   string
}

// NewBroker crea y configura un nuevo broker MQTT embebido.
func NewBroker(port string, repo repository.DeviceRepository) (*Broker, error) {
	if port == "" {
		port = "1883"
	}

	server := mqtt.New(&mqtt.Options{
		InlineClient: true, // Permite publicar/suscribir internamente sin socket si se requiere
	})

	// Por defecto, permitimos conexiones sin auth (red local doméstica de confianza)
	if err := server.AddHook(new(auth.AllowHook), nil); err != nil {
		return nil, fmt.Errorf("añadir auth allow hook a broker MQTT: %w", err)
	}

	// Añadimos el hook de enrutamiento hacia la base de datos de Cobalt
	routerHook := NewDeviceStateRouterHook(repo)
	if err := server.AddHook(routerHook, nil); err != nil {
		return nil, fmt.Errorf("añadir router hook a broker MQTT: %w", err)
	}

	tcpAddr := fmt.Sprintf(":%s", port)
	tcpListener := listeners.NewTCP(listeners.Config{
		ID:      "tcp-local",
		Address: tcpAddr,
	})
	if err := server.AddListener(tcpListener); err != nil {
		return nil, fmt.Errorf("añadir TCP listener en %s: %w", tcpAddr, err)
	}

	return &Broker{
		server: server,
		port:   port,
	}, nil
}

// Start arranca el servicio del broker. Bloquea hasta que el servidor se cierra o falla,
// por lo que debe invocarse habitualmente dentro de una goroutine.
func (b *Broker) Start() error {
	log.Printf("[MQTT] Broker embebido escuchando en :%s", b.port)
	if err := b.server.Serve(); err != nil {
		return fmt.Errorf("ejecutar broker MQTT: %w", err)
	}
	return nil
}

// Publish publica un mensaje de forma interna a través del broker.
func (b *Broker) Publish(topic string, payload []byte, retain bool) error {
	return b.server.Publish(topic, payload, retain, 0)
}

// Close detiene limpiamente los listeners y conexiones del broker.
func (b *Broker) Close() error {
	log.Println("[MQTT] Deteniendo broker embebido...")
	return b.server.Close()
}
