// Package sistemas sabe qué sistemas pueden mandar mail, desde qué remitentes y con qué apikey.
package sistemas

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"slices"
	"strings"
)

// DominiosPermitidos son los verificados en SES y los que deja la política del usuario IAM ses-smtp-relay.
var DominiosPermitidos = []string{"gpe.ar", "horatech.ar"}

var ErrRemitente = errors.New("remitente no permitido")

type Sistema struct {
	Remitentes map[string]string `json:"remitentes"`
}

// Cargar lee sistemas.json. Falla si un sistema no tiene remitente "default" o si un remitente
// no es una dirección sola de un dominio permitido.
func Cargar(r io.Reader) (map[string]Sistema, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var m map[string]Sistema
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, errors.New("no hay sistemas")
	}
	for nombre, s := range m {
		if _, ok := s.Remitentes["default"]; !ok {
			return nil, fmt.Errorf("%s: falta el remitente default", nombre)
		}
		for clave, dir := range s.Remitentes {
			if !dominioPermitido(dir) {
				return nil, fmt.Errorf("%s.%s: %q no es una dirección de un dominio permitido", nombre, clave, dir)
			}
		}
	}
	return m, nil
}

func dominioPermitido(dir string) bool {
	a, err := mail.ParseAddress(dir)
	if err != nil || a.Address != dir {
		return false
	}
	dominio := strings.ToLower(dir[strings.LastIndex(dir, "@")+1:])
	return slices.Contains(DominiosPermitidos, dominio)
}

// Remitente devuelve la dirección del remitente con ese nombre; "" es "default".
func (s Sistema) Remitente(nombre string) (string, error) {
	if nombre == "" {
		nombre = "default"
	}
	d, ok := s.Remitentes[nombre]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrRemitente, nombre)
	}
	return d, nil
}
