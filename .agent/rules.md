# Manifiesto Global: Cobalt Core

## 1. Identidad y Propósito del Proyecto
- **Cobalt** es un sistema domótico autohospedado (*self-hosted*), modular y privado, diseñado desde cero para ser controlado por código local y, en fases posteriores, por un motor de Inteligencia Artificial local (Ollama).
- Privilegia la simplicidad de despliegue, el rendimiento en crudo y el control absoluto sobre los dispositivos de la casa.

## 2. Directiva de Contexto Jerárquico
- Antes de modificar código, comprueba si existe un archivo `rules.md` específico en la subcarpeta de destino (ej. `internal/rules.md` o `web/rules.md`). Las reglas locales tienen prioridad estricta para sus respectivos dominios.

## 3. Principios Arquitectónicos Globales
- **Monolito en Go + 3 Contenedores:** El despliegue de producción consta exclusivamente de `postgres` (con `pgvector`), `python-matter-server` (para dispositivos Matter) y `cobalt` (el monolito en Go que incluye un broker MQTT embebido y el frontend empaquetado).
- **Cero ORMs en el Backend:** Prohibido usar GORM o mapeadores pesados. Todo el acceso a datos debe hacerse con SQL puro mediante `pgx/v5`.
- **Abstracción Políglota de Dispositivos:** Todos los dispositivos (MQTT, Matter, HTTP local) se unifican bajo una misma tabla relacional con campos `JSONB` para los estados.
- **Sincronización con Obsidian:** Toda decisión de diseño mayor debe documentarse en `docs/Cobalt/adr/` manteniendo la estructura de enlaces de la bóveda.