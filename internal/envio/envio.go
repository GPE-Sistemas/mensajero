// Package envio arma el mail MIME y se lo entrega a Postfix.
package envio

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

const dominioID = "mensajero.gpe.ar"

// ErrSMTP envuelve cualquier falla al hablar con Postfix: el que llama puede reintentar.
var ErrSMTP = errors.New("smtp")

type Mensaje struct {
	De         mail.Address
	Para       string
	ResponderA string // opcional
	Asunto     string
	HTML       string
	Texto      string // opcional
}

type SMTP struct {
	Addr    string        // host:puerto de Postfix
	Timeout time.Duration // total por envío
}

// Enviar entrega el mensaje y devuelve su Message-ID (sin <>). Postfix se encarga de reintentar contra SES.
func (s SMTP) Enviar(ctx context.Context, m Mensaje) (string, error) {
	id := nuevoID()
	cuerpo, err := Armar(m, id, time.Now())
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", s.Addr)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSMTP, err)
	}
	defer conn.Close()
	if fin, ok := ctx.Deadline(); ok {
		conn.SetDeadline(fin)
	}
	host, _, _ := net.SplitHostPort(s.Addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSMTP, err)
	}
	defer c.Close()

	pasos := []func() error{
		func() error { return c.Hello(dominioID) },
		func() error { return c.Mail(m.De.Address) },
		func() error { return c.Rcpt(m.Para) },
		func() error {
			w, err := c.Data()
			if err != nil {
				return err
			}
			if _, err := w.Write(cuerpo); err != nil {
				return err
			}
			return w.Close() // acá Postfix acepta o rechaza el mail
		},
	}
	for _, paso := range pasos {
		if err := paso(); err != nil {
			return "", fmt.Errorf("%w: %v", ErrSMTP, err)
		}
	}
	_ = c.Quit()
	return id, nil
}

// Armar genera el mail completo. Un salto de línea en cualquier header es error: nada de inyectar Bcc.
func Armar(m Mensaje, id string, fecha time.Time) ([]byte, error) {
	for _, v := range []string{m.Para, m.ResponderA, m.Asunto, m.De.Name, m.De.Address} {
		if strings.ContainsAny(v, "\r\n") {
			return nil, errors.New("salto de línea en un header")
		}
	}
	var b bytes.Buffer
	header := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	header("From", m.De.String())
	header("To", m.Para)
	if m.ResponderA != "" {
		header("Reply-To", m.ResponderA)
	}
	header("Subject", mime.QEncoding.Encode("utf-8", m.Asunto))
	header("Date", fecha.Format(time.RFC1123Z))
	header("Message-ID", "<"+id+">")
	header("MIME-Version", "1.0")

	if m.Texto == "" {
		header("Content-Type", `text/html; charset="utf-8"`)
		header("Content-Transfer-Encoding", "quoted-printable")
		b.WriteString("\r\n")
		if err := escribirQP(&b, m.HTML); err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}

	mw := multipart.NewWriter(&b)
	header("Content-Type", `multipart/alternative; boundary="`+mw.Boundary()+`"`)
	b.WriteString("\r\n")
	for _, parte := range []struct{ tipo, contenido string }{{"text/plain", m.Texto}, {"text/html", m.HTML}} {
		w, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {parte.tipo + `; charset="utf-8"`},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, err
		}
		if err := escribirQP(w, parte.contenido); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func escribirQP(w interface{ Write([]byte) (int, error) }, s string) error {
	qp := quotedprintable.NewWriter(w)
	if _, err := qp.Write([]byte(s)); err != nil {
		return err
	}
	return qp.Close()
}

func nuevoID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b) + "@" + dominioID
}
