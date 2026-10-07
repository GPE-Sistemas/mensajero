package envio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/GPE-Sistemas/mensajero/internal/smtpfalso"
)

var fecha = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func mensaje() Mensaje {
	return Mensaje{
		De:         mail.Address{Name: "Camuzzi", Address: "alertas@gpe.ar"},
		Para:       "a@x.com",
		ResponderA: "s@x.com",
		Asunto:     "Contraseña nueva",
		HTML:       "<p>hola ñ</p>",
		Texto:      "hola ñ",
	}
}

func TestArmarHTMLYTexto(t *testing.T) {
	b, err := Armar(mensaje(), "id1@mensajero.gpe.ar", fecha)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	de, _ := msg.Header.AddressList("From")
	if de[0].Name != "Camuzzi" || de[0].Address != "alertas@gpe.ar" {
		t.Errorf("From = %v", de[0])
	}
	asunto, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if asunto != "Contraseña nueva" {
		t.Errorf("Subject = %q", asunto)
	}
	for k, want := range map[string]string{"To": "a@x.com", "Reply-To": "s@x.com", "Message-Id": "<id1@mensajero.gpe.ar>"} {
		if got := msg.Header.Get(k); got != want {
			t.Errorf("%s = %q", k, got)
		}
	}
	tipo, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if tipo != "multipart/alternative" {
		t.Fatalf("Content-Type = %q", tipo)
	}
	mr := multipart.NewReader(msg.Body, params["boundary"])
	for _, want := range []struct{ tipo, cuerpo string }{{"text/plain", "hola ñ"}, {"text/html", "<p>hola ñ</p>"}} {
		parte, err := mr.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		cuerpo, _ := io.ReadAll(parte)
		if !strings.HasPrefix(parte.Header.Get("Content-Type"), want.tipo) || string(cuerpo) != want.cuerpo {
			t.Errorf("parte %s = %q", parte.Header.Get("Content-Type"), cuerpo)
		}
	}
}

func TestArmarSoloHTML(t *testing.T) {
	m := mensaje()
	m.Texto, m.ResponderA = "", ""
	b, _ := Armar(m, "id1@mensajero.gpe.ar", fecha)
	msg, _ := mail.ReadMessage(bytes.NewReader(b))
	if !strings.HasPrefix(msg.Header.Get("Content-Type"), "text/html") || msg.Header.Get("Reply-To") != "" {
		t.Fatalf("headers = %v", msg.Header)
	}
	cuerpo, _ := io.ReadAll(quotedprintable.NewReader(msg.Body))
	if string(cuerpo) != "<p>hola ñ</p>" {
		t.Fatalf("cuerpo = %q", cuerpo)
	}
}

func TestArmarRechazaSaltos(t *testing.T) {
	for _, romper := range []func(*Mensaje){
		func(m *Mensaje) { m.Para = "a@x.com\r\nBcc: z@y.com" },
		func(m *Mensaje) { m.ResponderA = "s@x.com\nBcc: z@y.com" },
		func(m *Mensaje) { m.Asunto = "a\r\nBcc: z@y.com" },
		func(m *Mensaje) { m.De.Name = "a\nBcc: z@y.com" },
	} {
		m := mensaje()
		romper(&m)
		if _, err := Armar(m, "id", fecha); err == nil {
			t.Errorf("armó con un salto: %+v", m)
		}
	}
}

func TestEnviar(t *testing.T) {
	srv := smtpfalso.Iniciar(t)
	id, err := SMTP{Addr: srv.Addr, Timeout: 5 * time.Second}.Enviar(context.Background(), mensaje())
	if err != nil {
		t.Fatal(err)
	}
	r := srv.Recibidos()
	if len(r) != 1 || r[0].De != "alertas@gpe.ar" || len(r[0].Para) != 1 || r[0].Para[0] != "a@x.com" {
		t.Fatalf("recibido = %+v", r)
	}
	if !strings.Contains(r[0].Datos, "Message-ID: <"+id+">") || !strings.HasSuffix(id, "@mensajero.gpe.ar") {
		t.Fatalf("id = %q, datos = %s", id, r[0].Datos)
	}
}

func TestEnviarRechazado(t *testing.T) {
	srv := smtpfalso.Iniciar(t)
	srv.Rechazar(true)
	if _, err := (SMTP{Addr: srv.Addr, Timeout: 5 * time.Second}).Enviar(context.Background(), mensaje()); !errors.Is(err, ErrSMTP) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnviarSinServidor(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	inicio := time.Now()
	_, err := SMTP{Addr: addr, Timeout: 5 * time.Second}.Enviar(context.Background(), mensaje())
	if !errors.Is(err, ErrSMTP) || time.Since(inicio) > 5*time.Second {
		t.Fatalf("err = %v en %v", err, time.Since(inicio))
	}
}
