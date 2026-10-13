package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joorgeeml13/cobalt-core/internal/api"
	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/drivers"
)

// mockRepo implementa repository.DeviceRepository en memoria para pruebas de endpoints.
type mockRepo struct {
	devices map[string]domain.Device
}

func newMockRepo() *mockRepo {
	return &mockRepo{devices: make(map[string]domain.Device)}
}

func (m *mockRepo) Insert(ctx context.Context, d *domain.Device) error {
	d.ID = "dev-uuid-123"
	m.devices[d.ID] = *d
	return nil
}

func (m *mockRepo) UpdateState(ctx context.Context, id string, state json.RawMessage) error {
	d, ok := m.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.State = state
	m.devices[id] = d
	return nil
}

func (m *mockRepo) SetOnline(ctx context.Context, id string, online bool) error {
	d, ok := m.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Online = online
	m.devices[id] = d
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	d, ok := m.devices[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &d, nil
}

func (m *mockRepo) ListAll(ctx context.Context) ([]domain.Device, error) {
	list := make([]domain.Device, 0, len(m.devices))
	for _, d := range m.devices {
		list = append(list, d)
	}
	return list, nil
}

func (m *mockRepo) ListByProtocol(ctx context.Context, protocol domain.Protocol) ([]domain.Device, error) {
	var list []domain.Device
	for _, d := range m.devices {
		if d.Protocol == protocol {
			list = append(list, d)
		}
	}
	return list, nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	delete(m.devices, id)
	return nil
}

func (m *mockRepo) Update(ctx context.Context, d *domain.Device) error {
	m.devices[d.ID] = *d
	return nil
}

func TestAPIRoutes(t *testing.T) {
	repo := newMockRepo()
	registry := drivers.NewRegistry(repo)

	// Crear router
	router := api.NewRouter(api.RouterConfig{
		DeviceRepo: repo,
		Registry:   registry,
	})

	// 1. Probar GET /api/health
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health returned %d, want 200", rec.Code)
	}

	var healthResp api.Response
	if err := json.NewDecoder(rec.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if !healthResp.Success {
		t.Errorf("Expected success: true")
	}

	// 2. Probar POST /api/devices
	createBody := []byte(`{"name":"Sensor Test","protocol":"mqtt","type":"sensor"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/devices", bytes.NewReader(createBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/devices returned %d, want 201", rec.Code)
	}

	// 3. Probar GET /api/devices
	req = httptest.NewRequest(http.MethodGet, "/api/devices", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/devices returned %d, want 200", rec.Code)
	}

	// 4. Probar GET /api/devices/dev-uuid-123
	req = httptest.NewRequest(http.MethodGet, "/api/devices/dev-uuid-123", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/devices/{id} returned %d, want 200", rec.Code)
	}

	// 5. Probar DELETE /api/devices/dev-uuid-123
	req = httptest.NewRequest(http.MethodDelete, "/api/devices/dev-uuid-123", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/devices/{id} returned %d, want 200", rec.Code)
	}

	// 6. Probar GET /api/protocols
	req = httptest.NewRequest(http.MethodGet, "/api/protocols", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/protocols returned %d, want 200", rec.Code)
	}
}
