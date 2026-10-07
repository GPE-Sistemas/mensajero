package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/GPE-Sistemas/mensajero/internal/envio"
	"github.com/GPE-Sistemas/mensajero/internal/limite"
	"github.com/GPE-Sistemas/mensajero/internal/plantillas"
	"github.com/GPE-Sistemas/mensajero/internal/sistemas"
	"github.com/GPE-Sistemas/mensajero/internal/smtpfalso"
)

type entorno struct {
	h    http.Handler
	smtp *smtpfalso.Servidor
	log  *bytes.Buffer
}

func nuevo(t *testing.T) entorno {
	t.Helper()
	cat, err := plantillas.Cargar(fstest.MapFS{
		"p/gas/base.html":                  {Data: []byte(`{{template "contenido" .}}`)},
		"p/gas/reset/asunto.txt":           {Data: []byte("Reseteá tu clave")},
		"p/gas/reset/cuerpo.html":          {Data: []byte(`{{define "contenido"}}<a href="{{.link}}">ir</a>{{end}}`)},
		"p/acceso/base.html":               {Data: []byte(`{{template "contenido" .}}`)},
		"p/acceso/solo-acceso/asunto.txt":  {Data: []byte("x")},
		"p/acceso/solo-acceso/cuerpo.html": {Data: []byte(`{{define "contenido"}}x{{end}}`)},
	}, "p")
	if err != nil {
		t.Fatal(err)
	}
	sis, err := sistemas.Cargar(strings.NewReader(`{"gas":{"remitentes":{"default":"no-reply@gpe.ar","alertas":"alertas@gpe.ar"}},"acceso":{"remitentes":{"default":"no-reply@horatech.ar"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	smtp := smtpfalso.Iniciar(t)
	var log bytes.Buffer
	s := &Servidor{
		Catalogo: cat,
		Sistemas: sis,
		Claves:   sistemas.Claves{"gas": sha256.Sum256([]byte("clave-gas")), "acceso": sha256.Sum256([]byte("clave-acceso"))},
		Limite:   limite.Nuevo(2, time.Hour),
		SMTP:     envio.SMTP{Addr: smtp.Addr, Timeout: 5 * time.Second},
		Log:      slog.New(slog.NewJSONHandler(&log, nil)),
	}
	return entorno{h: s.Handler(), smtp: smtp, log: &log}
}

func (e entorno) pedir(apikey, cuerpo string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/v1/email", strings.NewReader(cuerpo))
	if apikey != "" {
		r.Header.Set("Authorization", "Bearer "+apikey)
	}
	w := httptest.NewRecorder()
	e.h.ServeHTTP(w, r)
	return w
}

const ok = `{"plantilla":"reset","para":"a@x.com","datos":{"link":"https://iot-test.horatech.com.ar/#/login/reset?token=SECRETO123"}}`

func TestEnviaYDevuelveID(t *testing.T) {
	e := nuevo(t)
	w := e.pedir("clave-gas", ok)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	var resp struct{ ID string }
	json.Unmarshal(w.Body.Bytes(), &resp)
	r := e.smtp.Recibidos()
	if resp.ID == "" || len(r) != 1 || r[0].De != "no-reply@gpe.ar" || !strings.Contains(r[0].Datos, resp.ID) {
		t.Fatalf("id = %q, recibido = %+v", resp.ID, r)
	}
}

func TestRemitenteYNombre(t *testing.T) {
	e := nuevo(t)
	w := e.pedir("clave-gas", `{"plantilla":"reset","para":"a@x.com","remitente":"alertas","nombre":"Camuzzi","responderA":"s@x.com","datos":{"link":"https://x"}}`)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	d := e.smtp.Recibidos()[0].Datos
	if !strings.Contains(d, `From: "Camuzzi" <alertas@gpe.ar>`) || !strings.Contains(d, "Reply-To: s@x.com") {
		t.Fatalf("datos = %s", d)
	}
}

func TestErrores(t *testing.T) {
	casos := []struct {
		nombre, apikey, cuerpo string
		codigo                 int
	}{
		{"sin apikey", "", ok, 401},
		{"apikey mala", "otra", ok, 401},
		{"JSON roto", "clave-gas", `{`, 400},
		{"campo desconocido", "clave-gas", `{"plantilla":"reset","para":"a@x.com","datos":{"link":"x"},"bcc":"z@y.com"}`, 400},
		{"dato no string", "clave-gas", `{"plantilla":"reset","para":"a@x.com","datos":{"link":3}}`, 400},
		{"plantilla inexistente", "clave-gas", `{"plantilla":"no","para":"a@x.com","datos":{}}`, 400},
		{"plantilla de otro sistema", "clave-gas", `{"plantilla":"solo-acceso","para":"a@x.com","datos":{}}`, 400},
		{"remitente ajeno", "clave-gas", `{"plantilla":"reset","para":"a@x.com","remitente":"otro","datos":{"link":"x"}}`, 403},
		{"para inválido", "clave-gas", `{"plantilla":"reset","para":"no-es-mail","datos":{"link":"x"}}`, 400},
		{"para con nombre", "clave-gas", `{"plantilla":"reset","para":"Ana <a@x.com>","datos":{"link":"x"}}`, 400},
		{"para con salto", "clave-gas", `{"plantilla":"reset","para":"a@x.com\r\nBcc: z@y.com","datos":{"link":"x"}}`, 400},
		{"responderA inválido", "clave-gas", `{"plantilla":"reset","para":"a@x.com","responderA":"x","datos":{"link":"x"}}`, 400},
		{"nombre con salto", "clave-gas", `{"plantilla":"reset","para":"a@x.com","nombre":"a\nBcc: z@y.com","datos":{"link":"x"}}`, 400},
		{"falta dato", "clave-gas", `{"plantilla":"reset","para":"a@x.com","datos":{}}`, 400},
		{"sobra dato", "clave-gas", `{"plantilla":"reset","para":"a@x.com","datos":{"link":"x","y":"z"}}`, 400},
		{"cuerpo enorme", "clave-gas", `{"plantilla":"reset","para":"a@x.com","datos":{"link":"` + strings.Repeat("x", 70<<10) + `"}}`, 400},
	}
	e := nuevo(t)
	for _, c := range casos {
		if w := e.pedir(c.apikey, c.cuerpo); w.Code != c.codigo {
			t.Errorf("%s: %d %s", c.nombre, w.Code, w.Body)
		}
	}
	if n := len(e.smtp.Recibidos()); n != 0 {
		t.Fatalf("salieron %d mails con pedidos inválidos", n)
	}
}

func TestTope(t *testing.T) {
	e := nuevo(t)
	for i, want := range []int{200, 200, 429} {
		if w := e.pedir("clave-gas", ok); w.Code != want {
			t.Fatalf("envío %d: %d %s", i+1, w.Code, w.Body)
		}
	}
}

func TestPostfixRechaza(t *testing.T) {
	e := nuevo(t)
	e.smtp.Rechazar(true)
	if w := e.pedir("clave-gas", ok); w.Code != 502 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}

func TestNoLogueaDatosNiApikey(t *testing.T) {
	e := nuevo(t)
	e.pedir("clave-gas", ok)
	e.pedir("clave-gas", `{"plantilla":"reset","para":"a@x.com","datos":{"link":"SECRETO123","y":"z"}}`)
	e.smtp.Rechazar(true)
	e.pedir("clave-gas", ok)
	for _, prohibido := range []string{"SECRETO123", "clave-gas"} {
		if strings.Contains(e.log.String(), prohibido) {
			t.Fatalf("el log tiene %q:\n%s", prohibido, e.log)
		}
	}
	if !strings.Contains(e.log.String(), `"para":"a@x.com"`) {
		t.Fatalf("el log no tiene el destinatario:\n%s", e.log)
	}
}

func TestSalud(t *testing.T) {
	w := httptest.NewRecorder()
	nuevo(t).h.ServeHTTP(w, httptest.NewRequest("GET", "/salud", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}

func TestEnvioFallidoNoConsumeTope(t *testing.T) {
	e := nuevo(t)
	e.smtp.Rechazar(true)
	for i := 0; i < 2; i++ {
		if w := e.pedir("clave-gas", ok); w.Code != 502 {
			t.Fatalf("envío %d: %d %s", i+1, w.Code, w.Body)
		}
	}
	e.smtp.Rechazar(false)
	if w := e.pedir("clave-gas", ok); w.Code != 200 {
		t.Fatalf("tras el fallo: %d %s", w.Code, w.Body)
	}
}

func TestDireccionConEspacios(t *testing.T) {
	e := nuevo(t)
	w := e.pedir("clave-gas", `{"plantilla":"reset","para":" a@x.com ","datos":{"link":"l"}}`)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if r := e.smtp.Recibidos(); len(r) != 1 || len(r[0].Para) != 1 || r[0].Para[0] != "a@x.com" {
		t.Fatalf("recibidos = %+v", r)
	}
}

func TestDireccionNoASCII(t *testing.T) {
	e := nuevo(t)
	w := e.pedir("clave-gas", `{"plantilla":"reset","para":"josé@x.com","datos":{"link":"l"}}`)
	if w.Code != 400 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}
