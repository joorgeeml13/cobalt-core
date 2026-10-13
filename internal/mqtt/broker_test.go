package mqtt_test

import (
	"testing"
	"time"

	"github.com/joorgeeml13/cobalt-core/internal/mqtt"
)

func TestBrokerLifecycle(t *testing.T) {
	// Usamos un puerto alternativo para test local
	b, err := mqtt.NewBroker("11883", nil)
	if err != nil {
		t.Fatalf("NewBroker error: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- b.Start()
	}()

	// Dejar unos milisegundos para que el listener se enlace
	time.Sleep(100 * time.Millisecond)

	// Test de publicación interna inline
	if err := b.Publish("home/test_sensor/state", []byte(`{"temperature": 21.5}`), false); err != nil {
		t.Errorf("Publish error: %v", err)
	}

	// Detención limpia
	if err := b.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Logf("Start terminó con: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout esperando finalización del broker")
	}
}
