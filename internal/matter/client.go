package matter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

// Client gestiona la conexión WebSocket persistente contra python-matter-server,
// la reconexión automática y el intercambio bidireccional de comandos y eventos.
type Client struct {
	url        string
	repo       repository.DeviceRepository
	conn       *websocket.Conn
	mu         sync.Mutex
	writeMu    sync.Mutex
	msgCounter uint64

	// pending correlaciona peticiones salientes con respuestas entrantes por message_id
	pending   map[string]chan CommandResponse
	pendingMu sync.Mutex

	stopCh chan struct{}
}

// NewClient crea una nueva instancia del cliente Matter.
func NewClient(url string, repo repository.DeviceRepository) *Client {
	if url == "" {
		url = "ws://localhost:5580/ws"
	}
	return &Client{
		url:     url,
		repo:    repo,
		pending: make(map[string]chan CommandResponse),
		stopCh:  make(chan struct{}),
	}
}

// Start arranca el ciclo de vida del cliente: gestiona la conexión inicial
// y entra en bucle de reconexión automática ante caídas.
func (c *Client) Start(ctx context.Context) {
	log.Printf("[Matter] Iniciando cliente WebSocket hacia %s", c.url)

	backoff := 1 * time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			log.Println("[Matter] Contexto cancelado, deteniendo cliente Matter")
			c.Close()
			return
		case <-c.stopCh:
			log.Println("[Matter] Cliente Matter detenido")
			return
		default:
		}

		err := c.connectAndListen(ctx)
		if err != nil {
			log.Printf("[Matter] Error en conexión WebSocket con %s: %v. Reintentando en %v...", c.url, err, backoff)
		}

		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-time.After(backoff):
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// connectAndListen establece la conexión WebSocket y procesa los mensajes entrantes.
func (c *Client) connectAndListen(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("marcar conexión websocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	log.Printf("[Matter] Conectado exitosamente a python-matter-server en %s", c.url)

	// Suscribirse a eventos iniciales de nodos
	go func() {
		subCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := c.SendCommand(subCtx, "start_listening", nil); err != nil {
			log.Printf("[Matter] Aviso al enviar start_listening: %v", err)
		}
	}()

	// Bucle de lectura
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			if c.conn == conn {
				c.conn = nil
			}
			c.mu.Unlock()
			return fmt.Errorf("lectura de mensaje websocket: %w", err)
		}

		c.handleIncomingMessage(msg)
	}
}

// handleIncomingMessage discrimina si el mensaje es una respuesta a comando o un evento espontáneo.
func (c *Client) handleIncomingMessage(data []byte) {
	// 1. Intentar decodificar como CommandResponse (tiene "message_id")
	var resp CommandResponse
	if err := json.Unmarshal(data, &resp); err == nil && resp.MessageID != "" {
		c.pendingMu.Lock()
		ch, exists := c.pending[resp.MessageID]
		if exists {
			delete(c.pending, resp.MessageID)
		}
		c.pendingMu.Unlock()

		if exists {
			ch <- resp
			return
		}
	}

	// 2. Intentar decodificar como EventMessage
	var ev EventMessage
	if err := json.Unmarshal(data, &ev); err == nil && ev.Event != "" {
		c.handleEvent(ev)
	}
}

// handleEvent procesa eventos de actualización de dispositivos y persiste en la base de datos.
func (c *Client) handleEvent(ev EventMessage) {
	log.Printf("[Matter] Evento recibido: %s", ev.Event)

	if c.repo == nil {
		return
	}

	switch ev.Event {
	case "node_updated", "attribute_updated":
		var nodeData NodeEventData
		if err := json.Unmarshal(ev.Data, &nodeData); err != nil {
			log.Printf("[Matter] Error decodificando NodeEventData: %v", err)
			return
		}

		nodeIdentifier := fmt.Sprintf("matter_node_%d", nodeData.NodeID)
		ctx := context.Background()

		// Actualizar disponibilidad
		if nodeData.Available != nil {
			_ = c.repo.SetOnline(ctx, nodeIdentifier, *nodeData.Available)
		}

		// Actualizar estado si vienen atributos
		if len(nodeData.Attributes) > 0 {
			stateBytes, err := json.Marshal(nodeData.Attributes)
			if err == nil {
				if updateErr := c.repo.UpdateState(ctx, nodeIdentifier, stateBytes); updateErr != nil {
					// Auto-registro inicial si el nodo aún no existe en base de datos
					newDevice := &domain.Device{
						Name:     nodeIdentifier,
						Protocol: domain.ProtocolMatter,
						Type:     domain.TypeLight, // Tipo por defecto para actuadores Matter
						State:    stateBytes,
						Meta:     json.RawMessage(fmt.Sprintf(`{"node_id": %d}`, nodeData.NodeID)),
						Online:   true,
					}
					if insErr := c.repo.Insert(ctx, newDevice); insErr != nil {
						log.Printf("[Matter] Error registrando dispositivo Matter %s: %v", nodeIdentifier, insErr)
					} else {
						log.Printf("[Matter] Dispositivo Matter %s registrado en BD con ID=%s", nodeIdentifier, newDevice.ID)
					}
				}
			}
		}
	}
}

// SendCommand envía una orden tipada JSON-RPC al servidor Matter y espera su respuesta de forma síncrona.
func (c *Client) SendCommand(ctx context.Context, command string, args map[string]any) (*CommandResponse, error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("conexión websocket con Matter no establecida")
	}

	msgID := fmt.Sprintf("%d", atomic.AddUint64(&c.msgCounter, 1))
	req := CommandRequest{
		MessageID: msgID,
		Command:   command,
		Args:      args,
	}

	respCh := make(chan CommandResponse, 1)

	c.pendingMu.Lock()
	c.pending[msgID] = respCh
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, msgID)
		c.pendingMu.Unlock()
	}()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar comando Matter: %w", err)
	}

	c.writeMu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, payload)
	c.writeMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("escribir en websocket Matter: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respCh:
		if resp.ErrorCode != nil {
			return &resp, fmt.Errorf("error de Matter Server (código %d)", *resp.ErrorCode)
		}
		return &resp, nil
	}
}

// SetDeviceState es un helper para enviar órdenes habituales (On/Off, Brillo, etc.) a un nodo Matter.
func (c *Client) SetDeviceState(ctx context.Context, nodeID uint64, endpoint uint, cluster uint, command string, payload map[string]any) error {
	args := map[string]any{
		"node_id":      nodeID,
		"endpoint_id":  endpoint,
		"cluster_id":   cluster,
		"command_name": command,
		"payload":      payload,
	}
	_, err := c.SendCommand(ctx, "device_command", args)
	return err
}

// Close cierra la conexión activa y detiene el cliente.
func (c *Client) Close() error {
	select {
	case <-c.stopCh:
		return nil
	default:
		close(c.stopCh)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
