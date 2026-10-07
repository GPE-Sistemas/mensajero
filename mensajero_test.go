package mensajero

import (
	"maps"
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
			// Verificar que todos los valores de datos aparecen en el texto plano
			for k, v := range datos {
				if !strings.Contains(r.Texto, v) {
					t.Errorf("el texto no muestra %s=%q", k, v)
				}
			}
			// Verificar que los campos de la plantilla coinciden con las claves de datos
			gotCampos := slices.Sorted(maps.Keys(datos))
			if !slices.Equal(p.Campos, gotCampos) {
				t.Errorf("plantilla campos = %v, esperaba %v", p.Campos, gotCampos)
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
	cat, err := plantillas.Cargar(Archivos, "plantillas")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range cat.Todas() {
		if _, ok := sis[p.Sistema]; !ok {
			t.Errorf("hay plantillas de %q pero no está en sistemas.json", p.Sistema)
		}
	}
}
