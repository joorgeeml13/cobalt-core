package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joorgeeml13/cobalt-core/internal/drivers"
	"github.com/joorgeeml13/cobalt-core/internal/matter"
	"github.com/joorgeeml13/cobalt-core/internal/mqtt"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

// RouterConfig agrupa las dependencias necesarias para levantar la API REST de Cobalt.
type RouterConfig struct {
	DBPool       *pgxpool.Pool
	DeviceRepo   repository.DeviceRepository
	Registry     *drivers.Registry
	MQTTBroker   *mqtt.Broker
	MatterClient *matter.Client
}

// NewRouter crea y configura el router chi con todos los endpoints REST montados.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Middlewares esenciales
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Configuración de CORS permisiva para desarrollo local de frontend Vite (localhost:5173, etc.)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Instanciación de Handlers
	devHandler := NewDeviceHandler(cfg.DeviceRepo, cfg.Registry)
	discHandler := NewDiscoveryHandler(cfg.DeviceRepo, cfg.MQTTBroker, cfg.MatterClient)
	sysHandler := NewSystemHandler(cfg.DBPool)

	// Health check raíz y de API
	r.Get("/health", sysHandler.Health)

	// Agrupación bajo /api
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", sysHandler.Health)
		r.Get("/protocols", sysHandler.Protocols)

		// 1. Dispositivos (CRUD & Commands)
		r.Route("/devices", func(r chi.Router) {
			r.Get("/", devHandler.ListDevices)
			r.Post("/", devHandler.CreateDevice)

			// Subrutas de comisionado y descubrimiento
			r.Post("/commission/matter", discHandler.CommissionMatter)
			r.Post("/discover/mqtt", discHandler.DiscoverMQTT)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", devHandler.GetDevice)
				r.Delete("/", devHandler.DeleteDevice)
				r.Post("/command", devHandler.ExecuteCommand)
			})
		})

		// 2. Descubrimiento e inspección
		r.Route("/discovery", func(r chi.Router) {
			r.Get("/matter", discHandler.DiscoverMatter)
			r.Get("/mqtt/inspect", discHandler.InspectMQTT)
		})
	})

	return r
}
