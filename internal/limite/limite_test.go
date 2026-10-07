package limite

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPermitir(t *testing.T) {
	ahora := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	l := Nuevo(2, time.Hour)
	l.ahora = func() time.Time { return ahora }

	if !l.Permitir("a@x.com") || !l.Permitir("A@X.com") {
		t.Fatal("los dos primeros tienen que pasar")
	}
	if l.Permitir("a@x.com") {
		t.Fatal("el tercero en la hora no tiene que pasar")
	}
	if !l.Permitir("b@x.com") {
		t.Fatal("otro destinatario tiene su propio tope")
	}
	ahora = ahora.Add(time.Hour + time.Second)
	if !l.Permitir("a@x.com") {
		t.Fatal("pasada la ventana vuelve a pasar")
	}
}

func TestPermitirConcurrente(t *testing.T) {
	l := Nuevo(10, time.Hour)
	var pasaron atomic.Int32
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Permitir("a@x.com") {
				pasaron.Add(1)
			}
		}()
	}
	wg.Wait()
	if pasaron.Load() != 10 {
		t.Fatalf("pasaron %d, tenían que ser 10", pasaron.Load())
	}
}

func TestDevolver(t *testing.T) {
	ahora := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	l := Nuevo(1, time.Hour)
	l.ahora = func() time.Time { return ahora }

	l.Devolver("a@x.com") // sin entradas: no pasa nada
	if !l.Permitir("a@x.com") {
		t.Fatal("el primero tiene que pasar")
	}
	if l.Permitir("a@x.com") {
		t.Fatal("el segundo no tiene que pasar")
	}
	l.Devolver("A@X.com")
	if !l.Permitir("a@x.com") {
		t.Fatal("devuelto el cupo, vuelve a pasar")
	}
}
