# ==============================================================================
# Makefile - Cobalt Core Automation
# ==============================================================================

.DEFAULT_GOAL := help

# Intercepta subcomandos cuando el primer objetivo es 'dev' o 'prod'
ifeq ($(firstword $(MAKECMDGOALS)),dev)
  DEV_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(DEV_ARGS):;@:)
endif

ifeq ($(firstword $(MAKECMDGOALS)),prod)
  PROD_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(PROD_ARGS):;@:)
endif

.PHONY: help dev prod dev-up dev-down dev-build dev-logs restart dev-restart up down logs ps

help:
	@echo "Cobalt Core - Comandos disponibles:"
	@echo ""
	@echo "  Desarrollo (Usa compose.yml + compose.override.yml con build local):"
	@echo "    make dev up          - Levanta el entorno de desarrollo en segundo plano"
	@echo "    make dev down        - Detiene los contenedores de desarrollo"
	@echo "    make dev restart     - Reinicia recreando contenedores con cambios de .env/código"
	@echo "    make dev build       - Recompila la imagen local de Cobalt"
	@echo "    make dev logs        - Muestra los logs en tiempo real"
	@echo ""
	@echo "  Producción (Solo compose.yml usando imágenes de registro):"
	@echo "    make prod up         - Levanta producción (o make up)"
	@echo "    make prod down       - Detiene producción (o make down)"
	@echo "    make prod restart    - Reinicia producción"
	@echo "    make prod logs       - Muestra logs de producción"
	@echo "    make prod pull       - Descarga las imágenes más recientes"
	@echo ""
	@echo "  General:"
	@echo "    make ps              - Muestra el estado de los contenedores"
	@echo "    make restart         - Alias de make dev restart"
	@echo ""

# Sintaxis: make dev [up|down|restart|build|logs]
dev:
	@action="$(firstword $(DEV_ARGS))"; \
	if [ -z "$$action" ]; then \
		echo "Uso: make dev [up|down|restart|build|logs]"; \
	elif [ "$$action" = "up" ]; then \
		docker compose up -d --build; \
	elif [ "$$action" = "down" ]; then \
		docker compose down; \
	elif [ "$$action" = "restart" ]; then \
		docker compose up -d --force-recreate --build; \
	elif [ "$$action" = "build" ]; then \
		docker compose build; \
	elif [ "$$action" = "logs" ]; then \
		docker compose logs -f; \
	else \
		docker compose $$action; \
	fi

# Alias directos para desarrollo
dev-up:
	docker compose up -d --build

dev-down:
	docker compose down

dev-restart:
	docker compose up -d --force-recreate --build

dev-build:
	docker compose build

dev-logs:
	docker compose logs -f

# Sintaxis: make prod [up|down|restart|logs|pull]
prod:
	@action="$(firstword $(PROD_ARGS))"; \
	if [ -z "$$action" ]; then \
		echo "Uso: make prod [up|down|restart|logs|pull]"; \
	elif [ "$$action" = "up" ]; then \
		docker compose -f compose.yml up -d; \
	elif [ "$$action" = "down" ]; then \
		docker compose -f compose.yml down; \
	elif [ "$$action" = "restart" ]; then \
		docker compose -f compose.yml up -d --force-recreate; \
	elif [ "$$action" = "logs" ]; then \
		docker compose -f compose.yml logs -f; \
	elif [ "$$action" = "pull" ]; then \
		docker compose -f compose.yml pull; \
	else \
		docker compose -f compose.yml $$action; \
	fi

# Objetivos directos si no se invoca 'make dev ...'
ifneq ($(firstword $(MAKECMDGOALS)),dev)
.PHONY: up down restart logs

up:
	docker compose -f compose.yml up -d

down:
	docker compose -f compose.yml down

logs:
	docker compose -f compose.yml logs -f
endif

# Alias general de reinicio
restart: dev-restart

ps:
	@docker compose ps
