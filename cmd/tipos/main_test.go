package main

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/GPE-Sistemas/mensajero/internal/plantillas"
	"github.com/GPE-Sistemas/mensajero/internal/sistemas"
)

func TestGenerar(t *testing.T) {
	cat, err := plantillas.Cargar(fstest.MapFS{
		"p/gas/base.html":       {Data: []byte(`{{template "contenido" .}}`)},
		"p/gas/hola/asunto.txt": {Data: []byte("{{.nombre}}")},
		"p/gas/hola/cuerpo.html": {Data: []byte(`{{define "contenido"}}{{.link}}{{end}}`)},
	}, "p")
	if err != nil {
		t.Fatal(err)
	}
	sis, _ := sistemas.Cargar(strings.NewReader(`{"gas":{"remitentes":{"default":"a@gpe.ar","alertas":"b@gpe.ar"}}}`))
	got := string(generar(cat, sis))
	for _, want := range []string{
		"export interface Plantillas {\n  'gas': {\n    'hola': { link: string; nombre: string };\n  };\n}",
		"export interface Remitentes {\n  'gas': 'alertas' | 'default';\n}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("falta:\n%s\nen:\n%s", want, got)
		}
	}
}

// TestClienteAlDia falla si alguien cambió una plantilla sin correr `make generar`.
func TestClienteAlDia(t *testing.T) {
	want, err := salida()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../cliente/plantillas.d.ts")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("cliente/plantillas.d.ts está desactualizado: corré `make generar`")
	}
}
