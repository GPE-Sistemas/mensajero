package mensajero

import (
	"slices"
	"strings"
	"testing"

	"github.com/GPE-Sistemas/mensajero/internal/plantillas"
	"github.com/GPE-Sistemas/mensajero/internal/sistemas"
)

func TestPlantillasReales(t *testing.T) {
	cat, err := plantillas.Cargar(Archivos, "plantillas")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gas/cambio-password", "gas/nuevo-usuario", "gas/reset-password", "gas/scada-fuera-limite", "gas/scada-reestablecido"}
	var got []string
	for _, p := range cat.Todas() {
		got = append(got, p.Sistema+"/"+p.Nombre)
		t.Run(p.Sistema+"/"+p.Nombre, func(t *testing.T) {
			datos, err := plantillas.Ejemplo(Archivos, "plantillas", p.Sistema, p.Nombre)
			if err != nil {
				t.Fatal(err)
			}
			r, err := p.Renderizar(datos)
			if err != nil {
				t.Fatalf("no renderiza con su ejemplo.json: %v", err)
			}
			if r.Asunto == "" || !strings.Contains(r.HTML, "<html") || r.Texto == "" {
				t.Fatalf("renderizado incompleto: %+v", r)
			}
		})
	}
	if !slices.Equal(got, want) {
		t.Fatalf("plantillas = %v", got)
	}
}

func TestSistemasReal(t *testing.T) {
	f, err := Archivos.Open("sistemas.json")
	if err != nil {
		t.Fatal(err)
	}
	sis, err := sistemas.Cargar(f)
	if err != nil {
		t.Fatal(err)
	}
	cat, _ := plantillas.Cargar(Archivos, "plantillas")
	for _, p := range cat.Todas() {
		if _, ok := sis[p.Sistema]; !ok {
			t.Errorf("hay plantillas de %q pero no está en sistemas.json", p.Sistema)
		}
	}
}
