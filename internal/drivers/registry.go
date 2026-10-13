package drivers

import (
	"context"
	"fmt"
	"sync"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

// DriverFactory crea un DeviceDriver específico a partir de un dispositivo.
type DriverFactory func(device *domain.Device) (DeviceDriver, error)

// Registry gestiona la resolución y despacho de drivers políglotas.
type Registry struct {
	mu        sync.RWMutex
	factories map[domain.Protocol]DriverFactory
	repo      repository.DeviceRepository
}

// NewRegistry crea un nuevo registro central de drivers.
func NewRegistry(repo repository.DeviceRepository) *Registry {
	return &Registry{
		factories: make(map[domain.Protocol]DriverFactory),
		repo:      repo,
	}
}

// RegisterDriverFactory asocia una función creadora de driver a un protocolo determinado.
func (r *Registry) RegisterDriverFactory(protocol domain.Protocol, factory DriverFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[protocol] = factory
}

// GetDriver resuelve y devuelve el DeviceDriver adecuado para el dispositivo dado.
func (r *Registry) GetDriver(device *domain.Device) (DeviceDriver, error) {
	if device == nil {
		return nil, fmt.Errorf("dispositivo es nulo")
	}

	r.mu.RLock()
	factory, exists := r.factories[device.Protocol]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no hay driver registrado para el protocolo %q", device.Protocol)
	}

	return factory(device)
}

// ExecuteCommandByID busca un dispositivo en base de datos por su ID y ejecuta la acción sobre su driver.
func (r *Registry) ExecuteCommandByID(ctx context.Context, deviceID string, action string, payload map[string]any) error {
	if r.repo == nil {
		return fmt.Errorf("repositorio de dispositivos no configurado en el registry")
	}

	dev, err := r.repo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("obtener dispositivo %s: %w", deviceID, err)
	}

	driver, err := r.GetDriver(dev)
	if err != nil {
		return fmt.Errorf("resolver driver para dispositivo %s: %w", deviceID, err)
	}

	return driver.ExecuteCommand(ctx, action, payload)
}
