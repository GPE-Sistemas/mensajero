// Package httpapi expone POST /v1/email.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/GPE-Sistemas/mensajero/internal/envio"
	"github.com/GPE-Sistemas/mensajero/internal/limite"
	"github.com/GPE-Sistemas/mensajero/internal/plantillas"
	"github.com/GPE-Sistemas/mensajero/internal/sistemas"
)

const maxCuerpo = 64 << 10

type Servidor struct {
	Catalogo *plantillas.Catalogo
	Sistemas map[string]sistemas.Sistema
	Claves   sistemas.Claves
	Limite   *limite.Limite
	SMTP     envio.SMTP
	Log      *slog.Logger
}

type pedido struct {
	Plantilla  string            `json:"plantilla"`
	Para       string            `json:"para"`
	Remitente  string            `json:"remitente"`
	Nombre     string            `json:"nombre"`
	ResponderA string            `json:"responderA"`
	Datos      map[string]string `json:"datos"`
}

func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/email", s.email)
	mux.HandleFunc("GET /salud", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	return mux
}

func (s *Servidor) email(w http.ResponseWriter, r *http.Request) {
	apikey, conBearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	sistema, valida := s.Claves.Sistema(apikey)
	if !conBearer || !valida {
		s.rechazar(w, http.StatusUnauthorized, "", "", "", "apikey inválida")
		return
	}

	var p pedido
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCuerpo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		s.rechazar(w, http.StatusBadRequest, sistema, "", "", "JSON inválido: "+err.Error())
		return
	}
	pl, ok := s.Catalogo.Buscar(sistema, p.Plantilla)
	if !ok {
		s.rechazar(w, http.StatusBadRequest, sistema, p.Plantilla, "", fmt.Sprintf("la plantilla %q no existe para %s", p.Plantilla, sistema))
		return
	}
	de, err := s.Sistemas[sistema].Remitente(p.Remitente)
	if err != nil {
		s.rechazar(w, http.StatusForbidden, sistema, p.Plantilla, "", err.Error())
		return
	}
	para, err := direccion(p.Para)
	if err != nil {
		s.rechazar(w, http.StatusBadRequest, sistema, p.Plantilla, "", "para: "+err.Error())
		return
	}
	responderA := ""
	if p.ResponderA != "" {
		if responderA, err = direccion(p.ResponderA); err != nil {
			s.rechazar(w, http.StatusBadRequest, sistema, p.Plantilla, para, "responderA: "+err.Error())
			return
		}
	}
	if strings.ContainsAny(p.Nombre, "\r\n") {
		s.rechazar(w, http.StatusBadRequest, sistema, p.Plantilla, para, "nombre: no puede tener saltos de línea")
		return
	}
	ren, err := pl.Renderizar(p.Datos)
	if errors.Is(err, plantillas.ErrDatos) {
		s.rechazar(w, http.StatusBadRequest, sistema, p.Plantilla, para, err.Error())
		return
	}
	if err != nil {
		s.rechazar(w, http.StatusInternalServerError, sistema, p.Plantilla, para, "la plantilla falló: "+err.Error())
		return
	}
	if !s.Limite.Permitir(para) {
		s.rechazar(w, http.StatusTooManyRequests, sistema, p.Plantilla, para, "se pasó el tope de envíos a esta dirección")
		return
	}

	id, err := s.SMTP.Enviar(r.Context(), envio.Mensaje{
		De:         mail.Address{Name: p.Nombre, Address: de},
		Para:       para,
		ResponderA: responderA,
		Asunto:     ren.Asunto,
		HTML:       ren.HTML,
		Texto:      ren.Texto,
	})
	if err != nil {
		s.rechazar(w, http.StatusBadGateway, sistema, p.Plantilla, para, err.Error())
		return
	}
	s.Log.Info("envío", "sistema", sistema, "plantilla", p.Plantilla, "para", para, "de", de, "id", id, "resultado", "ok")
	responder(w, http.StatusOK, map[string]string{"id": id})
}

// rechazar loguea y responde el error. Nunca recibe los datos ni la apikey.
func (s *Servidor) rechazar(w http.ResponseWriter, codigo int, sistema, plantilla, para, motivo string) {
	s.Log.Warn("envío rechazado", "sistema", sistema, "plantilla", plantilla, "para", para, "codigo", codigo, "motivo", motivo)
	responder(w, codigo, map[string]string{"error": motivo})
}

func responder(w http.ResponseWriter, codigo int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(codigo)
	json.NewEncoder(w).Encode(cuerpo)
}

// direccion acepta sólo una dirección sola, sin nombre ni nada alrededor.
func direccion(s string) (string, error) {
	a, err := mail.ParseAddress(s)
	if err != nil {
		return "", err
	}
	if a.Address != s {
		return "", errors.New("tiene que ser sólo la dirección")
	}
	return a.Address, nil
}
