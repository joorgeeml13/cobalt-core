package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joorgeeml13/cobalt-core/internal/database"
	"github.com/joorgeeml13/cobalt-core/internal/domain"
	"github.com/joorgeeml13/cobalt-core/internal/drivers"
	"github.com/joorgeeml13/cobalt-core/internal/matter"
	"github.com/joorgeeml13/cobalt-core/internal/mqtt"
	"github.com/joorgeeml13/cobalt-core/internal/repository"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mqttPort := os.Getenv("MQTT_PORT")
	if mqttPort == "" {
		mqttPort = "1883"
	}

	matterURL := os.Getenv("MATTER_SERVER_URL")
	if matterURL == "" {
		matterURL = "ws://localhost:5580/ws"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL no configurada")
	}

	// 1. Inicializar pool de base de datos y ejecutar migraciones embebidas
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("Conectar a base de datos: %v", err)
	}
	defer pool.Close()
	log.Println("Base de datos conectada y migraciones aplicadas")

	// 2. Instanciar repositorio de dispositivos
	deviceRepo := repository.NewDeviceRepository(pool)

	// 3. Inicializar y arrancar el broker MQTT embebido en segundo plano
	mqttBroker, err := mqtt.NewBroker(mqttPort, deviceRepo)
	if err != nil {
		log.Fatalf("Iniciar broker MQTT: %v", err)
	}

	go func() {
		if err := mqttBroker.Start(); err != nil {
			log.Printf("Broker MQTT terminado: %v", err)
		}
	}()

	// 4. Inicializar y arrancar cliente WebSocket de Matter
	matterClient := matter.NewClient(matterURL, deviceRepo)
	go matterClient.Start(ctx)

	// 5. Inicializar Registry central de Drivers políglotas
	driverRegistry := drivers.NewRegistry(deviceRepo)
	driverRegistry.RegisterDriverFactory(domain.ProtocolMQTT, func(d *domain.Device) (drivers.DeviceDriver, error) {
		return drivers.NewMQTTDriver(d, mqttBroker), nil
	})
	driverRegistry.RegisterDriverFactory(domain.ProtocolMatter, func(d *domain.Device) (drivers.DeviceDriver, error) {
		return drivers.NewMatterDriver(d, matterClient), nil
	})
	driverRegistry.RegisterDriverFactory(domain.ProtocolHTTP, func(d *domain.Device) (drivers.DeviceDriver, error) {
		return drivers.NewHTTPDriver(d), nil
	})
	log.Println("Driver Registry inicializado con soporte para MQTT, Matter y HTTP")

	// 6. Servidor HTTP
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Cobalt Core HTTP escuchando en :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Servidor HTTP: %v", err)
		}
	}()

	// 7. Esperar señal de apagado (SIGINT/SIGTERM)
	<-ctx.Done()
	log.Println("Señal de apagado recibida, cerrando servicios...")

	// Graceful shutdown de HTTP
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error en graceful shutdown HTTP: %v", err)
	}

	// Detener cliente Matter
	if err := matterClient.Close(); err != nil {
		log.Printf("Error cerrando cliente Matter: %v", err)
	}

	// Detener broker MQTT
	if err := mqttBroker.Close(); err != nil {
		log.Printf("Error cerrando broker MQTT: %v", err)
	}

	log.Println("Servidor cerrado correctamente")
}
