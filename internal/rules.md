# Reglas de Backend (Go / Base de Datos)

## 1. Patrones y Estilo en Go
- **Cero ORMs:** Prohibido usar GORM o mapeadores pesados. Todo el acceso a datos se hace mediante SQL plano usando el driver `pgx/v5`.
- **Manejo de Errores:** Nunca ignores un error con `_`. Envuelve siempre los errores con contexto explícito usando `fmt.Errorf("...: %w", err)`.
- **Inyección de Dependencias:** Sin frameworks mágicos. Inyección manual por constructor en las estructuras (ej. `NewRepository(db *pgxpool.Pool)`).

## 2. Base de Datos y Persistencia
- Las consultas SQL deben estar parametrizadas estrictamente (`$1`, `$2`) para prevenir inyecciones.
- Mantén la separación estricta de capas: los handlers HTTP no contienen lógica de negocio ni SQL; los repositorios no manejan peticiones HTTP.