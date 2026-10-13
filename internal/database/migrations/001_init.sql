-- ==============================================================================
-- 001_init.sql — Cobalt Core: esquema inicial (idempotente)
-- ==============================================================================
-- Este script es seguro para ejecutarse múltiples veces (IF NOT EXISTS en todo).

-- pgcrypto provee gen_random_uuid() en PostgreSQL 16
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ------------------------------------------------------------------------------
-- Tabla polígota de dispositivos
-- Centraliza MQTT, Matter y HTTP bajo un único modelo relacional.
-- El estado heterogéneo de cada dispositivo se almacena en el campo JSONB `state`.
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS devices (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    protocol    TEXT        NOT NULL CHECK (protocol IN ('mqtt', 'matter', 'http')),
    type        TEXT        NOT NULL CHECK (type IN ('sensor', 'light', 'tv')),
    state       JSONB       NOT NULL DEFAULT '{}',
    meta        JSONB       NOT NULL DEFAULT '{}',
    online      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ------------------------------------------------------------------------------
-- Índices
-- ------------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_devices_protocol  ON devices(protocol);
CREATE INDEX IF NOT EXISTS idx_devices_type      ON devices(type);
CREATE INDEX IF NOT EXISTS idx_devices_online    ON devices(online);

-- GIN permite búsquedas eficientes dentro del JSONB:
--   WHERE state @> '{"on": true}'
--   WHERE state ? 'temperature'
CREATE INDEX IF NOT EXISTS idx_devices_state_gin ON devices USING GIN(state);

-- ------------------------------------------------------------------------------
-- Trigger: actualiza updated_at automáticamente en cada UPDATE
-- ------------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgname = 'trg_devices_updated_at'
          AND tgrelid = 'devices'::regclass
    ) THEN
        CREATE TRIGGER trg_devices_updated_at
        BEFORE UPDATE ON devices
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END;
$$;
