// Package limite frena envíos repetidos a una misma dirección.
package limite

import (
	"strings"
	"sync"
	"time"
)

// Limite es un freno contra bugs y loops, no una contabilidad.
// ponytail: en memoria, se pierde con el pod y vale para una réplica; pasar a Redis si hay más.
// Las direcciones que no vuelven a aparecer quedan en el mapa: con el volumen de mails
// transaccionales es despreciable.
type Limite struct {
	mu      sync.Mutex
	max     int
	ventana time.Duration
	ahora   func() time.Time
	envios  map[string][]time.Time
}

func Nuevo(max int, ventana time.Duration) *Limite {
	return &Limite{max: max, ventana: ventana, ahora: time.Now, envios: map[string][]time.Time{}}
}

// Devolver quita el envío más reciente a destinatario: para cuando el envío falló y no tiene que gastar cupo.
func (l *Limite) Devolver(destinatario string) {
	destinatario = strings.ToLower(destinatario)
	l.mu.Lock()
	defer l.mu.Unlock()
	if e := l.envios[destinatario]; len(e) > 0 {
		l.envios[destinatario] = e[:len(e)-1]
	}
}

// Permitir registra un envío a destinatario si no pasa el tope dentro de la ventana.
// Los rechazados no cuentan.
func (l *Limite) Permitir(destinatario string) bool {
	destinatario = strings.ToLower(destinatario)
	l.mu.Lock()
	defer l.mu.Unlock()

	ahora := l.ahora()
	corte := ahora.Add(-l.ventana)
	var vigentes []time.Time
	for _, t := range l.envios[destinatario] {
		if t.After(corte) {
			vigentes = append(vigentes, t)
		}
	}
	if len(vigentes) >= l.max {
		l.envios[destinatario] = vigentes
		return false
	}
	l.envios[destinatario] = append(vigentes, ahora)
	return true
}
