package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/joorgeeml13/cobalt-core/internal/database"
	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

// setupContainerPool levanta un contenedor Postgres efímero para los tests.
// No requiere que docker compose esté levantado: el contenedor se gestiona
// automáticamente por testcontainers-go y se destruye al finalizar el test.
func setupContainerPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image: "pgvector/pgvector:pg16",
		Env: map[string]string{
			"POSTGRES_DB":       "cobalt_test",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "password",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2),
	}

	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("no se pudo iniciar el contenedor de Postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Terminate(ctx); err != nil {
			t.Logf("advertencia: error al terminar contenedor Postgres: %v", err)
		}
	})

	host, err := pg.Host(ctx)
	if err != nil {
		t.Fatalf("obtener host del contenedor: %v", err)
	}
	mappedPort, err := pg.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("obtener puerto mapeado: %v", err)
	}

	connStr := fmt.Sprintf(
		"postgres://postgres:password@%s:%s/cobalt_test?sslmode=disable",
		host, mappedPort.Port(),
	)

	pool, err := database.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("conectar al contenedor Postgres: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	return pool
}

// runRepoTests ejecuta el suite completo de tests del repositorio contra el pool dado.
// Reutilizado por los tests de contenedor para no duplicar lógica.
func runRepoTests(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	repo := repository.NewDeviceRepository(pool)

	t.Run("insert_y_get_por_id", func(t *testing.T) {
		d := &domain.Device{
			Name:     "Sensor Salón",
			Protocol: domain.ProtocolMQTT,
			Type:     domain.TypeSensor,
			State:    json.RawMessage(`{"temperature": 22.5, "humidity": 65}`),
			Meta:     json.RawMessage(`{"topic": "home/salon/sensor"}`),
			Online:   true,
		}
		if err := repo.Insert(ctx, d); err != nil {
			t.Fatalf("Insert: %v", err)
		}
		if d.ID == "" {
			t.Fatal("Insert debe rellenar el ID")
		}

		got, err := repo.GetByID(ctx, d.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != d.Name {
			t.Errorf("Name: got %q, want %q", got.Name, d.Name)
		}
		if got.Protocol != d.Protocol {
			t.Errorf("Protocol: got %q, want %q", got.Protocol, d.Protocol)
		}
		if !got.Online {
			t.Error("Online debería ser true")
		}
	})

	t.Run("update_state", func(t *testing.T) {
		d := &domain.Device{
			Name:     "Bombilla Matter",
			Protocol: domain.ProtocolMatter,
			Type:     domain.TypeLight,
			State:    json.RawMessage(`{"on": false, "brightness": 0}`),
			Meta:     json.RawMessage(`{"matter_node": "node-1"}`),
		}
		if err := repo.Insert(ctx, d); err != nil {
			t.Fatalf("Insert: %v", err)
		}

		newState := json.RawMessage(`{"on": true, "brightness": 80, "color_temp": 4000}`)
		if err := repo.UpdateState(ctx, d.ID, newState); err != nil {
			t.Fatalf("UpdateState: %v", err)
		}

		got, err := repo.GetByID(ctx, d.ID)
		if err != nil {
			t.Fatalf("GetByID tras UpdateState: %v", err)
		}
		// Verificar que el campo `on` fue actualizado
		var state map[string]any
		if err := json.Unmarshal(got.State, &state); err != nil {
			t.Fatalf("unmarshal state: %v", err)
		}
		if on, ok := state["on"].(bool); !ok || !on {
			t.Errorf("state['on'] debería ser true, got: %v", state["on"])
		}
		// El nombre no debe haber cambiado
		if got.Name != d.Name {
			t.Errorf("UpdateState no debe modificar Name: got %q", got.Name)
		}
	})

	t.Run("set_online", func(t *testing.T) {
		d := &domain.Device{
			Name:     "Sensor Offline",
			Protocol: domain.ProtocolMQTT,
			Type:     domain.TypeSensor,
			State:    json.RawMessage(`{}`),
			Meta:     json.RawMessage(`{}`),
			Online:   false,
		}
		if err := repo.Insert(ctx, d); err != nil {
			t.Fatalf("Insert: %v", err)
		}

		if err := repo.SetOnline(ctx, d.ID, true); err != nil {
			t.Fatalf("SetOnline(true): %v", err)
		}
		got, _ := repo.GetByID(ctx, d.ID)
		if !got.Online {
			t.Error("debería estar online tras SetOnline(true)")
		}

		if err := repo.SetOnline(ctx, d.ID, false); err != nil {
			t.Fatalf("SetOnline(false): %v", err)
		}
		got, _ = repo.GetByID(ctx, d.ID)
		if got.Online {
			t.Error("debería estar offline tras SetOnline(false)")
		}
	})

	t.Run("list_all", func(t *testing.T) {
		// Insertar dos dispositivos de tipos distintos
		fixtures := []*domain.Device{
			{Name: "TV-listall", Protocol: domain.ProtocolHTTP, Type: domain.TypeTV,
				State: json.RawMessage(`{"power": true}`), Meta: json.RawMessage(`{}`)},
			{Name: "Sensor-listall", Protocol: domain.ProtocolMQTT, Type: domain.TypeSensor,
				State: json.RawMessage(`{"temp": 20}`), Meta: json.RawMessage(`{}`)},
		}
		for i, d := range fixtures {
			if err := repo.Insert(ctx, d); err != nil {
				t.Fatalf("Insert dispositivo %d: %v", i, err)
			}
		}

		all, err := repo.ListAll(ctx)
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		if len(all) < 2 {
			t.Errorf("ListAll: got %d, want >= 2", len(all))
		}
	})

	t.Run("list_by_protocol", func(t *testing.T) {
		mqttDevice := &domain.Device{
			Name:     "Sensor MQTT Único",
			Protocol: domain.ProtocolMQTT,
			Type:     domain.TypeSensor,
			State:    json.RawMessage(`{"battery": 99}`),
			Meta:     json.RawMessage(`{"topic": "home/test"}`),
		}
		if err := repo.Insert(ctx, mqttDevice); err != nil {
			t.Fatalf("Insert: %v", err)
		}

		result, err := repo.ListByProtocol(ctx, domain.ProtocolMQTT)
		if err != nil {
			t.Fatalf("ListByProtocol: %v", err)
		}
		for _, dev := range result {
			if dev.Protocol != domain.ProtocolMQTT {
				t.Errorf("protocolo incorrecto: %q", dev.Protocol)
			}
		}
	})
}

// TestContainerDeviceRepository ejecuta el suite completo levantando un
// contenedor Postgres efímero. No requiere DATABASE_URL ni docker compose previo.
func TestContainerDeviceRepository(t *testing.T) {
	pool := setupContainerPool(t)
	runRepoTests(t, pool)
}
