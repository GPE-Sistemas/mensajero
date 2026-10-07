package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"testing"
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

	malos := map[string]map[string]string{
		"sin claves":       {},
		"sistema sin json": {"CLAVES": `{"agro":"` + hashHex("k") + `"}`},
		"tope inválido":    {"CLAVES": `{"gas":"` + hashHex("k") + `"}`, "TOPE_POR_DESTINATARIO_HORA": "mucho"},
		"tope cero":        {"CLAVES": `{"gas":"` + hashHex("k") + `"}`, "TOPE_POR_DESTINATARIO_HORA": "0"},
	}
	for nombre, e := range malos {
		if _, _, err := configurar(env(e), log); err == nil {
			t.Errorf("%s: configuró", nombre)
		}
	}
}
