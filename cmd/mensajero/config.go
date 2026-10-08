package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/GPE-Sistemas/mensajero"
	"github.com/GPE-Sistemas/mensajero/internal/envio"
	"github.com/GPE-Sistemas/mensajero/internal/httpapi"
	"github.com/GPE-Sistemas/mensajero/internal/limite"
	"github.com/GPE-Sistemas/mensajero/internal/plantillas"
	"github.com/GPE-Sistemas/mensajero/internal/sistemas"
)

func configurar(getenv func(string) string, log *slog.Logger) (*httpapi.Servidor, string, error) {
	porDefecto := func(k, d string) string {
		if v := getenv(k); v != "" {
			return v
		}
		return d
	}
	cat, err := plantillas.Cargar(mensajero.Archivos, "plantillas")
	if err != nil {
		return nil, "", err
	}
	f, err := mensajero.Archivos.Open("sistemas.json")
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	sis, err := sistemas.Cargar(f)
	if err != nil {
		return nil, "", err
	}
	claves, err := sistemas.CargarClaves(getenv("CLAVES"))
	if err != nil {
		return nil, "", err
	}
	for s := range claves {
		if _, ok := sis[s]; !ok {
			return nil, "", fmt.Errorf("CLAVES: %q no está en sistemas.json", s)
		}
	}
	tope, err := strconv.Atoi(porDefecto("TOPE_POR_DESTINATARIO_HORA", "30"))
	if err != nil || tope < 1 {
		return nil, "", fmt.Errorf("TOPE_POR_DESTINATARIO_HORA tiene que ser un entero mayor a 0")
	}
	var dominios map[string]bool
	for _, d := range strings.Split(getenv("DOMINIOS_PERMITIDOS"), ",") {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			if dominios == nil {
				dominios = map[string]bool{}
			}
			dominios[d] = true
		}
	}
	return &httpapi.Servidor{
		Catalogo: cat,
		Sistemas: sis,
		Claves:   claves,
		Limite:   limite.Nuevo(tope, time.Hour),
		SMTP:     envio.SMTP{Addr: porDefecto("SMTP_ADDR", "postfix.mail.svc.cluster.local:587"), Timeout: 10 * time.Second},
		Log:      log,
		Dominios: dominios,
	}, porDefecto("PUERTO", "8080"), nil
}
