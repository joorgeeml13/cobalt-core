package matter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/joorgeeml13/cobalt-core/internal/matter"
)

var upgrader = websocket.Upgrader{}

func TestMatterClientCommunication(t *testing.T) {
	// 1. Simular servidor WebSocket de python-matter-server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				break
			}

			var req matter.CommandRequest
			if err := json.Unmarshal(msg, &req); err == nil {
				// Responder al comando con éxito usando su mismo MessageID
				resp := matter.CommandResponse{
					MessageID: req.MessageID,
					Result:    json.RawMessage(`{"success": true}`),
				}
				respBytes, _ := json.Marshal(resp)
				_ = c.WriteMessage(websocket.TextMessage, respBytes)
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	// 2. Instanciar cliente Matter apuntando al servidor mock
	client := matter.NewClient(wsURL, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go client.Start(ctx)

	// Esperar que la conexión se complete
	time.Sleep(100 * time.Millisecond)

	// 3. Probar envío de comando correlativo
	cmdCtx, cmdCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cmdCancel()

	resp, err := client.SendCommand(cmdCtx, "test_command", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("SendCommand falló: %v", err)
	}

	if resp.MessageID != "1" && resp.MessageID != "2" {
		t.Errorf("MessageID inesperado: %s", resp.MessageID)
	}

	// 4. Detener cliente
	if err := client.Close(); err != nil {
		t.Errorf("Error al cerrar cliente Matter: %v", err)
	}
}
