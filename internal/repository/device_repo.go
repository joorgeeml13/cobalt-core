package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joorgeeml13/cobalt-core/internal/domain"
)

// DeviceRepository define el contrato de persistencia para dispositivos.
// Los handlers HTTP y los workers de protocolo dependen de esta interfaz,
// nunca de la implementación concreta.
type DeviceRepository interface {
	// Insert persiste un nuevo dispositivo. Rellena d.ID, d.CreatedAt y
	// d.UpdatedAt con los valores generados por la base de datos.
	Insert(ctx context.Context, d *domain.Device) error

	// UpdateState reemplaza el campo JSONB state del dispositivo indicado.
	// Devuelve error si el dispositivo no existe.
	UpdateState(ctx context.Context, id string, state json.RawMessage) error

	// SetOnline actualiza el campo online del dispositivo indicado.
	// Devuelve error si el dispositivo no existe.
	SetOnline(ctx context.Context, id string, online bool) error

	// GetByID devuelve el dispositivo con el UUID indicado.
	GetByID(ctx context.Context, id string) (*domain.Device, error)

	// ListAll devuelve todos los dispositivos ordenados por fecha de creación.
	ListAll(ctx context.Context) ([]domain.Device, error)

	// ListByProtocol devuelve los dispositivos que usan el protocolo indicado.
	ListByProtocol(ctx context.Context, protocol domain.Protocol) ([]domain.Device, error)
}

// pgDeviceRepository es la implementación de DeviceRepository respaldada por PostgreSQL
// a través de pgxpool. Usa SQL puro parametrizado; cero ORMs.
type pgDeviceRepository struct {
	pool *pgxpool.Pool
}

// NewDeviceRepository devuelve un DeviceRepository listo para usar.
// Inyección por constructor: el pool se pasa explícitamente, sin magia global.
func NewDeviceRepository(pool *pgxpool.Pool) DeviceRepository {
	return &pgDeviceRepository{pool: pool}
}

func (r *pgDeviceRepository) Insert(ctx context.Context, d *domain.Device) error {
	const query = `
		INSERT INTO devices (name, protocol, type, state, meta, online)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		d.Name,
		string(d.Protocol),
		string(d.Type),
		[]byte(d.State),
		[]byte(d.Meta),
		d.Online,
	)
	if err := row.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return fmt.Errorf("insertar dispositivo: %w", err)
	}
	return nil
}

func (r *pgDeviceRepository) UpdateState(ctx context.Context, id string, state json.RawMessage) error {
	const query = `UPDATE devices SET state = $1 WHERE id = $2`

	tag, err := r.pool.Exec(ctx, query, []byte(state), id)
	if err != nil {
		return fmt.Errorf("actualizar estado del dispositivo %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("dispositivo %s no encontrado", id)
	}
	return nil
}

func (r *pgDeviceRepository) SetOnline(ctx context.Context, id string, online bool) error {
	const query = `UPDATE devices SET online = $1 WHERE id = $2`

	tag, err := r.pool.Exec(ctx, query, online, id)
	if err != nil {
		return fmt.Errorf("actualizar online del dispositivo %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("dispositivo %s no encontrado", id)
	}
	return nil
}

func (r *pgDeviceRepository) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	const query = `
		SELECT id, name, protocol, type, state, meta, online, created_at, updated_at
		FROM devices
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	d, err := scanDevice(row)
	if err != nil {
		return nil, fmt.Errorf("obtener dispositivo %s: %w", id, err)
	}
	return d, nil
}

func (r *pgDeviceRepository) ListAll(ctx context.Context) ([]domain.Device, error) {
	const query = `
		SELECT id, name, protocol, type, state, meta, online, created_at, updated_at
		FROM devices
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listar dispositivos: %w", err)
	}
	defer rows.Close()

	return collectDevices(rows)
}

func (r *pgDeviceRepository) ListByProtocol(ctx context.Context, protocol domain.Protocol) ([]domain.Device, error) {
	const query = `
		SELECT id, name, protocol, type, state, meta, online, created_at, updated_at
		FROM devices
		WHERE protocol = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, string(protocol))
	if err != nil {
		return nil, fmt.Errorf("listar dispositivos por protocolo %s: %w", protocol, err)
	}
	defer rows.Close()

	return collectDevices(rows)
}

// scanDevice lee un único dispositivo desde una pgx.Row.
func scanDevice(row pgx.Row) (*domain.Device, error) {
	var d domain.Device
	var proto, devType string
	var stateBytes, metaBytes []byte

	if err := row.Scan(
		&d.ID, &d.Name, &proto, &devType,
		&stateBytes, &metaBytes,
		&d.Online, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return nil, err
	}

	d.Protocol = domain.Protocol(proto)
	d.Type = domain.DeviceType(devType)
	d.State = json.RawMessage(stateBytes)
	d.Meta = json.RawMessage(metaBytes)
	return &d, nil
}

// collectDevices itera sobre pgx.Rows y acumula los dispositivos en un slice.
func collectDevices(rows pgx.Rows) ([]domain.Device, error) {
	var devices []domain.Device

	for rows.Next() {
		var d domain.Device
		var proto, devType string
		var stateBytes, metaBytes []byte

		if err := rows.Scan(
			&d.ID, &d.Name, &proto, &devType,
			&stateBytes, &metaBytes,
			&d.Online, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("escanear dispositivo: %w", err)
		}

		d.Protocol = domain.Protocol(proto)
		d.Type = domain.DeviceType(devType)
		d.State = json.RawMessage(stateBytes)
		d.Meta = json.RawMessage(metaBytes)
		devices = append(devices, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar dispositivos: %w", err)
	}
	return devices, nil
}
