# Reglas de Documentación y Obsidian (docs/)

## 1. Convenciones de Enlaces y Markdown
- **WikiLinks de Obsidian:** Utiliza siempre enlaces internos de Obsidian (`[[Nombre de la nota]]`) para interconectar conceptos técnicos, arquitectura y ADRs.
- **Estructura de ADRs:** Todo Architectural Decision Record dentro de `docs/adr/` debe seguir el formato estándar: `ADR-[Número]-[Título-Breve].md` (Contexto, Decisión, Consecuencias).

## 2. Actualización del Roadmap
- Mantén el archivo `roadmap.md` actualizado conforme se vayan completando o modificando las tareas de las fases principales.
- No borres el histórico de progreso; usa los checkboxes (`[x]`) para marcar las tareas finalizadas.

## 3. Sincronización con el Código
- Si realizas un cambio estructural importante en el monolito (Go, Docker, base de datos), asegúrate de reflejarlo en la carpeta `docs/Cobalt/architecture/` antes de darlo por bueno.