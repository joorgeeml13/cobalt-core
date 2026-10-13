# ==============================================================================
# Makefile - Cobalt Core Automation
# ==============================================================================

.DEFAULT_GOAL := help

# Intercepta subcomandos cuando el primer objetivo es 'dev' o 'prod'
ifeq ($(firstword $(MAKECMDGOALS)),dev)
  DEV_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # Convertir los argumentos siguientes en reglas no-op para que Make no los ejecute dos veces
  $(eval $(DEV_ARGS):;@:)
endif

ifeq ($(firstword $(MAKECMDGOALS)),prod)
  PROD_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(PROD_ARGS):;@:)
endif

.PHONY: help dev prod dev-up dev-down dev-build dev-logs ps

help:
	@echo "Cobalt Core - Comandos disponibles:"
	@echo ""
	@echo "  Desarrollo (Build local desde Dockerfile):"
	@echo "    make dev up         - Levanta el entorno de desarrollo en segundo plano"
	@echo "    make dev down       - Detiene y remueve los contenedores de desarrollo"
	@echo "    make dev build      - Recompila la imagen local de Cobalt"
	@echo "    make dev logs       - Muestra los logs en tiempo real"
	@echo ""
	@echo "  Producción (Imágenes desde GHCR / Docker Hub):"
	@echo "    make up             - Levanta el entorno de producción (o make prod up)"
	@echo "    make down           - Detiene el entorno de producción (o make prod down)"
	@echo "    make logs           - Muestra los logs del entorno de producción"
	@echo "    make prod pull      - Descarga las imágenes más recientes"
	@echo ""
	@echo "  General:"
	@echo "    make ps             - Muestra el estado de los contenedores"
	@echo ""

# Sintaxis: make dev [up|down|build|logs]
dev:
	@action="$(firstword $(DEV_ARGS))"; \
	if [ -z "$$action" ]; then \
		echo "Uso: make dev [up|down|build|logs]"; \
	elif [ "$$action" = "up" ]; then \
		docker compose -f compose.dev.yml up -d --build; \
	elif [ "$$action" = "down" ]; then \
		docker compose -f compose.dev.yml down; \
	elif [ "$$action" = "build" ]; then \
		docker compose -f compose.dev.yml build; \
	elif [ "$$action" = "logs" ]; then \
		docker compose -f compose.dev.yml logs -f; \
	else \
		docker compose -f compose.dev.yml $$action; \
	fi

# Alias directos con guion
dev-up:
	docker compose -f compose.dev.yml up -d --build

dev-down:
	docker compose -f compose.dev.yml down

dev-build:
	docker compose -f compose.dev.yml build

dev-logs:
	docker compose -f compose.dev.yml logs -f

# Sintaxis: make prod [up|down|logs|pull]
prod:
	@action="$(firstword $(PROD_ARGS))"; \
	if [ -z "$$action" ]; then \
		echo "Uso: make prod [up|down|logs|pull]"; \
	elif [ "$$action" = "up" ]; then \
		docker compose -f compose.yml up -d; \
	elif [ "$$action" = "down" ]; then \
		docker compose -f compose.yml down; \
	elif [ "$$action" = "logs" ]; then \
		docker compose -f compose.yml logs -f; \
	elif [ "$$action" = "pull" ]; then \
		docker compose -f compose.yml pull; \
	else \
		docker compose -f compose.yml $$action; \
	fi

# Objetivos directos de producción (solo activos si no se invoco 'make dev ...')
ifneq ($(firstword $(MAKECMDGOALS)),dev)
.PHONY: up down logs

up:
	docker compose -f compose.yml up -d

down:
	docker compose -f compose.yml down

logs:
	docker compose -f compose.yml logs -f
endif

ps:
	@echo "=== Contenedores de Desarrollo ==="
	@docker compose -f compose.dev.yml ps
	@echo ""
	@echo "=== Contenedores de Producción ==="
	@docker compose -f compose.yml ps
