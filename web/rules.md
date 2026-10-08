# Reglas de Frontend (React / Vite)

## 1. Stack y Estrictez
- **TypeScript Estricto:** Cero uso de `any`. Define interfaces claras para los estados de dispositivos, respuestas de la API y eventos de WebSocket.
- **Estilos:** Tailwind CSS puro. Mantén una interfaz limpia, oscura, de aspecto técnico/DevOps (alineada con la estética de herramientas de infraestructura).

## 2. Comunicación
- Centraliza las llamadas HTTP al backend de Go y la gestión del socket en hooks o clientes dedicados. Maneja los estados de carga y errores de forma visual en la UI.