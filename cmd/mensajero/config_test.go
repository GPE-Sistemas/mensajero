package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func hashHex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func TestConfigurar(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, puerto, err := configurar(env(map[string]string{"CLAVES": `{"gas":"` + hashHex("k") + `"}`}), log)
	if err != nil {
		t.Fatal(err)
	}
	if puerto != "8080" || srv.SMTP.Addr != "postfix.mail.svc.cluster.local:587" {
		t.Fatalf("puerto = %q, smtp = %q", puerto, srv.SMTP.Addr)
	}
	if _, ok := srv.Catalogo.Buscar("gas", "reset-password"); !ok {
		t.Fatal("no cargó las plantillas embebidas")
	}

	srv, puerto, err = configurar(env(map[string]string{
		"CLAVES": `{"gas":"` + hashHex("k") + `"}`, "SMTP_ADDR": "x:25", "PUERTO": "9090", "TOPE_POR_DESTINATARIO_HORA": "5",
	}), log)
	if err != nil {
		t.Fatal(err)
	}
	if srv.SMTP.Addr != "x:25" || puerto != "9090" || srv.SMTP.Timeout != 10*time.Second {
		t.Fatalf("overrides: smtp = %q, puerto = %q, timeout = %v", srv.SMTP.Addr, puerto, srv.SMTP.Timeout)
	}

	malos := map[string]struct {
		env map[string]string
		msg string
	}{
		"sin claves":       {map[string]string{}, "CLAVES"},
		"sistema sin json": {map[string]string{"CLAVES": `{"agro":"` + hashHex("k") + `"}`}, `"agro"`},
		"tope inválido":    {map[string]string{"CLAVES": `{"gas":"` + hashHex("k") + `"}`, "TOPE_POR_DESTINATARIO_HORA": "mucho"}, "TOPE_POR_DESTINATARIO_HORA"},
		"tope cero":        {map[string]string{"CLAVES": `{"gas":"` + hashHex("k") + `"}`, "TOPE_POR_DESTINATARIO_HORA": "0"}, "TOPE_POR_DESTINATARIO_HORA"},
	}
	for nombre, c := range malos {
		_, _, err := configurar(env(c.env), log)
		if err == nil {
			t.Errorf("%s: configuró", nombre)
		} else if !strings.Contains(err.Error(), c.msg) {
			t.Errorf("%s: error %q no contiene %q", nombre, err, c.msg)
		}
	}
}
