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
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL no configurada")
	}

	// Inicializar pool y ejecutar migraciones automáticamente
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("Conectar a base de datos: %v", err)
	}
	defer pool.Close()
	log.Println("Base de datos conectada y migraciones aplicadas")

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
		log.Printf("Cobalt Core monolith escuchando en :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Servidor HTTP: %v", err)
		}
	}()

	// Esperar señal de apagado (SIGINT/SIGTERM)
	<-ctx.Done()
	log.Println("Señal de apagado recibida, cerrando servidor...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error en graceful shutdown: %v", err)
	}
	log.Println("Servidor cerrado correctamente")
}
