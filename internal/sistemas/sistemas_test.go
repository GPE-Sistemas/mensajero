package sistemas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestCargar(t *testing.T) {
	s, err := Cargar(strings.NewReader(`{"gas":{"remitentes":{"default":"no-reply@gpe.ar","alertas":"alertas@horatech.ar"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := s["gas"].Remitente(""); d != "no-reply@gpe.ar" {
		t.Fatalf("default = %q", d)
	}
	if d, _ := s["gas"].Remitente("alertas"); d != "alertas@horatech.ar" {
		t.Fatalf("alertas = %q", d)
	}
	if _, err := s["gas"].Remitente("otro"); !errors.Is(err, ErrRemitente) {
		t.Fatalf("otro: %v", err)
	}
	if _, err := (Sistema{}).Remitente(""); !errors.Is(err, ErrRemitente) {
		t.Fatalf("sistema vacío: %v", err)
	}
}

func TestCargarRechaza(t *testing.T) {
	casos := map[string]string{
		"dominio ajeno": `{"gas":{"remitentes":{"default":"x@gmail.com"}}}`,
		"subdominio":    `{"gas":{"remitentes":{"default":"x@acceso.gpe.ar"}}}`,
		"sin default":   `{"gas":{"remitentes":{"alertas":"a@gpe.ar"}}}`,
		"con nombre":    `{"gas":{"remitentes":{"default":"GPE <x@gpe.ar>"}}}`,
		"campo de más":  `{"gas":{"remitentes":{"default":"x@gpe.ar"},"otro":1}}`,
		"vacío":         `{}`,
	}
	for nombre, j := range casos {
		if _, err := Cargar(strings.NewReader(j)); err == nil {
			t.Errorf("%s: cargó", nombre)
		}
	}
}

func TestClaves(t *testing.T) {
	h := sha256.Sum256([]byte("secreta"))
	c, err := CargarClaves(`{"gas":"` + hex.EncodeToString(h[:]) + `"}`)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := c.Sistema("secreta"); !ok || s != "gas" {
		t.Fatalf("Sistema = %q, %v", s, ok)
	}
	for _, k := range []string{"", "otra", "secreta "} {
		if _, ok := c.Sistema(k); ok {
			t.Errorf("%q aceptada", k)
		}
	}
	for _, j := range []string{``, `{}`, `{"gas":"corta"}`, `{"gas":"zz` + strings.Repeat("0", 62) + `"}`} {
		if _, err := CargarClaves(j); err == nil {
			t.Errorf("%q aceptado", j)
		}
	}
}
