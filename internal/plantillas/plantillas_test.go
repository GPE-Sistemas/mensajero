package plantillas

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func fsPrueba() fstest.MapFS {
	return fstest.MapFS{
		"p/gas/base.html":             {Data: []byte(`<html><body>{{template "contenido" .}}<footer>{{.firma}}</footer></body></html>`)},
		"p/gas/hola/asunto.txt":       {Data: []byte("Hola {{.nombre}}\n")},
		"p/gas/hola/cuerpo.html":      {Data: []byte(`{{define "contenido"}}<p>Hola {{.nombre}}</p>{{if .link}}<a href="{{.link}}">ir</a>{{end}}{{end}}`)},
		"p/gas/hola/cuerpo.txt":       {Data: []byte("Hola {{.nombre}}")},
		"p/gas/hola/ejemplo.json":     {Data: []byte(`{"nombre":"Ana","link":"https://x","firma":"GPE"}`)},
		"p/gas/solo-html/asunto.txt":  {Data: []byte("Aviso")},
		"p/gas/solo-html/cuerpo.html": {Data: []byte(`{{define "contenido"}}<p>{{.texto}}</p>{{end}}`)},
	}
}

func cargar(t *testing.T) *Catalogo {
	t.Helper()
	c, err := Cargar(fsPrueba(), "p")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCampos(t *testing.T) {
	c := cargar(t)
	p, ok := c.Buscar("gas", "hola")
	if !ok {
		t.Fatal("no encontró gas/hola")
	}
	if want := []string{"firma", "link", "nombre"}; !slices.Equal(p.Campos, want) {
		t.Fatalf("Campos = %v, want %v", p.Campos, want)
	}
	if _, ok := c.Buscar("acceso", "hola"); ok {
		t.Fatal("encontró una plantilla de otro sistema")
	}
	if n := len(c.Todas()); n != 2 {
		t.Fatalf("Todas = %d", n)
	}
}

func TestRenderizar(t *testing.T) {
	p, _ := cargar(t).Buscar("gas", "hola")
	r, err := p.Renderizar(map[string]string{"nombre": "Ana <b>", "link": "https://x/?a=1", "firma": "GPE"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Asunto != "Hola Ana <b>" {
		t.Errorf("Asunto = %q", r.Asunto)
	}
	if !strings.Contains(r.HTML, "Hola Ana &lt;b&gt;") || !strings.Contains(r.HTML, `href="https://x/?a=1"`) || !strings.Contains(r.HTML, "<footer>GPE</footer>") {
		t.Errorf("HTML = %s", r.HTML)
	}
	if r.Texto != "Hola Ana <b>" {
		t.Errorf("Texto = %q", r.Texto)
	}
}

func TestLinkPeligroso(t *testing.T) {
	p, _ := cargar(t).Buscar("gas", "hola")
	r, err := p.Renderizar(map[string]string{"nombre": "A", "link": "javascript:alert(1)", "firma": ""})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.HTML, "javascript:") || !strings.Contains(r.HTML, "#ZgotmplZ") {
		t.Fatalf("HTML = %s", r.HTML)
	}
}

func TestSoloHTML(t *testing.T) {
	p, _ := cargar(t).Buscar("gas", "solo-html")
	r, err := p.Renderizar(map[string]string{"texto": "x", "firma": "GPE"})
	if err != nil || r.Texto != "" {
		t.Fatalf("Texto = %q, err = %v", r.Texto, err)
	}
}

func TestRenderizarDatosMal(t *testing.T) {
	p, _ := cargar(t).Buscar("gas", "hola")
	casos := map[string]map[string]string{
		"falta":           {"nombre": "A", "firma": ""},
		"sobra":           {"nombre": "A", "link": "", "firma": "", "x": ""},
		"salto en asunto": {"nombre": "A\r\nBcc: z@y.com", "link": "", "firma": ""},
	}
	for nombre, datos := range casos {
		if _, err := p.Renderizar(datos); !errors.Is(err, ErrDatos) {
			t.Errorf("%s: err = %v", nombre, err)
		}
	}
}

func TestCargarRechaza(t *testing.T) {
	casos := map[string]func(fstest.MapFS){
		"anidado": func(f fstest.MapFS) {
			f["p/gas/hola/cuerpo.html"] = &fstest.MapFile{Data: []byte(`{{define "contenido"}}{{.a.b}}{{end}}`)}
		},
		"range": func(f fstest.MapFS) {
			f["p/gas/hola/cuerpo.html"] = &fstest.MapFile{Data: []byte(`{{define "contenido"}}{{range .x}}{{end}}{{end}}`)}
		},
		"with": func(f fstest.MapFS) {
			f["p/gas/hola/cuerpo.html"] = &fstest.MapFile{Data: []byte(`{{define "contenido"}}{{with .x}}{{end}}{{end}}`)}
		},
		"sin base":   func(f fstest.MapFS) { delete(f, "p/gas/base.html") },
		"sin asunto": func(f fstest.MapFS) { delete(f, "p/gas/hola/asunto.txt") },
		"sintaxis":   func(f fstest.MapFS) { f["p/gas/hola/asunto.txt"] = &fstest.MapFile{Data: []byte(`{{.nombre`)} },
	}
	for nombre, romper := range casos {
		f := fsPrueba()
		romper(f)
		if _, err := Cargar(f, "p"); err == nil {
			t.Errorf("%s: cargó", nombre)
		}
	}
}

func TestEjemplo(t *testing.T) {
	d, err := Ejemplo(fsPrueba(), "p", "gas", "hola")
	if err != nil || d["nombre"] != "Ana" {
		t.Fatalf("d = %v, err = %v", d, err)
	}
}
