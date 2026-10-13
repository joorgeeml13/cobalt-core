package repository_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joorgeeml13/cobalt-core/internal/database"
	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

// setupPool crea un pool de conexiones para los tests de integración.
// Si DATABASE_URL no está definida en el entorno, el test se omite automáticamente.
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL no configurado; omitiendo test de integración")
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("no se pudo conectar a la base de datos: %v", err)
	}

	// Limpieza al finalizar el test para no contaminar otros tests
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM devices")
		pool.Close()
	})

	return pool
}

func TestInsertAndGetByID(t *testing.T) {
	pool := setupPool(t)
	repo := repository.NewDeviceRepository(pool)
	ctx := context.Background()

	d := &domain.Device{
		Name:     "Sensor Salón",
		Protocol: domain.ProtocolMQTT,
		Type:     domain.TypeSensor,
		State:    json.RawMessage(`{"temperature": 22.5, "humidity": 65, "battery": 87}`),
		Meta:     json.RawMessage(`{"topic": "home/salon/sensor"}`),
		Online:   true,
	}

	if err := repo.Insert(ctx, d); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if d.ID == "" {
		t.Fatal("Insert debe rellenar d.ID con el UUID generado por Postgres")
	}
	if d.CreatedAt.IsZero() {
		t.Error("Insert debe rellenar d.CreatedAt")
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
	if got.Type != d.Type {
		t.Errorf("Type: got %q, want %q", got.Type, d.Type)
	}
	if !got.Online {
		t.Error("Online: el dispositivo debería estar online")
	}
}

func TestUpdateState(t *testing.T) {
	pool := setupPool(t)
	repo := repository.NewDeviceRepository(pool)
	ctx := context.Background()

	d := &domain.Device{
		Name:     "Bombilla Habitación",
		Protocol: domain.ProtocolMatter,
		Type:     domain.TypeLight,
		State:    json.RawMessage(`{"on": false, "brightness": 0}`),
		Meta:     json.RawMessage(`{"matter_node": "node-42"}`),
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
		t.Fatalf("GetByID: %v", err)
	}

	// Comparamos la representación JSON normalizada
	var wantMap, gotMap map[string]any
	if err := json.Unmarshal(newState, &wantMap); err != nil {
		t.Fatalf("unmarshal newState: %v", err)
	}
	if err := json.Unmarshal(got.State, &gotMap); err != nil {
		t.Fatalf("unmarshal got.State: %v", err)
	}
	if len(gotMap) != len(wantMap) {
		t.Errorf("State: got %s, want %s", got.State, newState)
	}
	// El nombre del dispositivo no debe haber cambiado
	if got.Name != d.Name {
		t.Errorf("UpdateState no debe modificar Name: got %q, want %q", got.Name, d.Name)
	}
}

func TestSetOnline(t *testing.T) {
	pool := setupPool(t)
	repo := repository.NewDeviceRepository(pool)
	ctx := context.Background()

	d := &domain.Device{
		Name:     "Sensor Entrada",
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
	got, err := repo.GetByID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !got.Online {
		t.Error("SetOnline(true): el dispositivo debería estar online")
	}

	if err := repo.SetOnline(ctx, d.ID, false); err != nil {
		t.Fatalf("SetOnline(false): %v", err)
	}
	got, err = repo.GetByID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Online {
		t.Error("SetOnline(false): el dispositivo debería estar offline")
	}
}

func TestListAll(t *testing.T) {
	pool := setupPool(t)
	repo := repository.NewDeviceRepository(pool)
	ctx := context.Background()

	fixtures := []*domain.Device{
		{
			Name: "TV Salón", Protocol: domain.ProtocolHTTP, Type: domain.TypeTV,
			State: json.RawMessage(`{"power": true, "input": "HDMI1"}`),
			Meta:  json.RawMessage(`{"ip": "192.168.1.50"}`),
		},
		{
			Name: "Sensor Cocina", Protocol: domain.ProtocolMQTT, Type: domain.TypeSensor,
			State: json.RawMessage(`{"temperature": 19.0}`),
			Meta:  json.RawMessage(`{"topic": "home/cocina/sensor"}`),
		},
	}
	for _, d := range fixtures {
		if err := repo.Insert(ctx, d); err != nil {
			t.Fatalf("Insert %q: %v", d.Name, err)
		}
	}

	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) < len(fixtures) {
		t.Errorf("ListAll: got %d devices, want >= %d", len(all), len(fixtures))
	}
}

func TestListByProtocol(t *testing.T) {
	pool := setupPool(t)
	repo := repository.NewDeviceRepository(pool)
	ctx := context.Background()

	mqttDevices := []*domain.Device{
		{
			Name: "Sensor Jardín", Protocol: domain.ProtocolMQTT, Type: domain.TypeSensor,
			State: json.RawMessage(`{"temperature": 15.0}`),
			Meta:  json.RawMessage(`{"topic": "home/jardin/sensor"}`),
		},
		{
			Name: "Sensor Garaje", Protocol: domain.ProtocolMQTT, Type: domain.TypeSensor,
			State: json.RawMessage(`{"motion": false}`),
			Meta:  json.RawMessage(`{"topic": "home/garaje/motion"}`),
		},
	}
	// Un dispositivo de otro protocolo para verificar el filtrado
	matterDevice := &domain.Device{
		Name: "Bombilla Pasillo", Protocol: domain.ProtocolMatter, Type: domain.TypeLight,
		State: json.RawMessage(`{"on": true}`),
		Meta:  json.RawMessage(`{"matter_node": "node-7"}`),
	}

	for _, d := range mqttDevices {
		if err := repo.Insert(ctx, d); err != nil {
			t.Fatalf("Insert %q: %v", d.Name, err)
		}
	}
	if err := repo.Insert(ctx, matterDevice); err != nil {
		t.Fatalf("Insert matter device: %v", err)
	}

	result, err := repo.ListByProtocol(ctx, domain.ProtocolMQTT)
	if err != nil {
		t.Fatalf("ListByProtocol: %v", err)
	}
	if len(result) < len(mqttDevices) {
		t.Errorf("ListByProtocol: got %d devices, want >= %d", len(result), len(mqttDevices))
	}
	for _, dev := range result {
		if dev.Protocol != domain.ProtocolMQTT {
			t.Errorf("ListByProtocol devolvió dispositivo con protocolo incorrecto: %q", dev.Protocol)
		}
	}
}
