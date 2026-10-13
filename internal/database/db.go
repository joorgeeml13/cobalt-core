package database

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// initSQL contiene el DDL embebido en el binario en tiempo de compilación.
// Es idempotente: seguro para ejecutarse en cada arranque del servidor.
//
//go:embed migrations/001_init.sql
var initSQL string

// NewPool crea y configura un pool de conexiones pgx contra la base de datos
// indicada por databaseURL. Ejecuta las migraciones automáticamente antes de
// devolver el pool listo para usar.
//
// El tamaño del pool se controla vía variables de entorno:
//   - DB_MAX_CONNS (default: 10)
//   - DB_MIN_CONNS (default: 2)
//
// El llamador es responsable de cerrar el pool cuando el proceso finalice:
//
//	pool, err := database.NewPool(ctx, dbURL)
//	if err != nil { ... }
//	defer pool.Close()
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsear config de base de datos: %w", err)
	}

	cfg.MaxConns = envInt32("DB_MAX_CONNS", 10)
	cfg.MinConns = envInt32("DB_MIN_CONNS", 2)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("crear pool pgx: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping a base de datos: %w", err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ejecutar migraciones: %w", err)
	}

	return pool, nil
}

// runMigrations ejecuta el script SQL embebido contra el pool.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, initSQL); err != nil {
		return fmt.Errorf("aplicar 001_init.sql: %w", err)
	}
	return nil
}

// envInt32 lee una variable de entorno como int32.
// Devuelve el valor por defecto si la variable no existe o no es un número válido.
func envInt32(key string, defaultVal int32) int32 {
	if s := os.Getenv(key); s != "" {
		if n, err := strconv.ParseInt(s, 10, 32); err == nil && n > 0 {
			return int32(n)
		}
	}
	return defaultVal
}
