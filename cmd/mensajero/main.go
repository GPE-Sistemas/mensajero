// mensajero recibe pedidos de mail de los sistemas y se los entrega a Postfix.
//
//	mensajero                                 levanta el servidor
//	mensajero vista-previa gas/reset-password renderiza con ejemplo.json e imprime la ruta del HTML
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/GPE-Sistemas/mensajero"
	"github.com/GPE-Sistemas/mensajero/internal/plantillas"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "vista-previa" {
		if err := vistaPrevia(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := correr(log); err != nil {
		log.Error("el mensajero se detuvo", "error", err.Error())
		os.Exit(1)
	}
}

func correr(log *slog.Logger) error {
	srv, puerto, err := configurar(os.Getenv, log)
	if err != nil {
		return err
	}
	hs := &http.Server{Addr: ":" + puerto, Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	apagado := make(chan struct{})
	go func() {
		defer close(apagado)
		<-ctx.Done()
		apagar, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// deja terminar los envíos en curso
		if err := hs.Shutdown(apagar); err != nil {
			log.Error("apagado incompleto", "error", err.Error())
		}
	}()

	log.Info("escuchando", "puerto", puerto, "plantillas", len(srv.Catalogo.Todas()), "smtp", srv.SMTP.Addr, "dominios", slices.Sorted(maps.Keys(srv.Dominios)))
	if err := hs.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-apagado // ListenAndServe vuelve apenas arranca Shutdown; esperar a que termine de drenar
	return nil
}

func vistaPrevia(arg string) error {
	sistema, nombre, ok := strings.Cut(arg, "/")
	if !ok {
		return errors.New("uso: mensajero vista-previa <sistema>/<plantilla>")
	}
	cat, err := plantillas.Cargar(mensajero.Archivos, "plantillas")
	if err != nil {
		return err
	}
	p, ok := cat.Buscar(sistema, nombre)
	if !ok {
		return fmt.Errorf("no existe %s", arg)
	}
	datos, err := plantillas.Ejemplo(mensajero.Archivos, "plantillas", sistema, nombre)
	if err != nil {
		return err
	}
	r, err := p.Renderizar(datos)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp("", "vista-previa-*.html")
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "<!-- Asunto: %s -->\n%s", r.Asunto, r.HTML)
	fmt.Println(f.Name())
	return nil
}
