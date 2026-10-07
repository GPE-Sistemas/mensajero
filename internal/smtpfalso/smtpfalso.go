// Package smtpfalso es un servidor SMTP mínimo para tests: acepta lo que le llega y lo guarda.
package smtpfalso

import (
	"net"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type Recibido struct {
	De    string
	Para  []string
	Datos string
}

type Servidor struct {
	Addr string

	ln        net.Listener
	rechazar  atomic.Bool
	mu        sync.Mutex
	recibidos []Recibido
}

// Rechazar hace que el servidor responda 550 a todo RCPT TO (seguro entre goroutines).
func (s *Servidor) Rechazar(b bool) { s.rechazar.Store(b) }

func Iniciar(t testing.TB) *Servidor {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &Servidor{Addr: ln.Addr().String(), ln: ln}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.atender(c)
		}
	}()
	return s
}

func (s *Servidor) Recibidos() []Recibido {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Recibido(nil), s.recibidos...)
}

func (s *Servidor) atender(c net.Conn) {
	defer c.Close()
	tp := textproto.NewConn(c)
	tp.PrintfLine("220 smtpfalso")
	var r Recibido
	for {
		linea, err := tp.ReadLine()
		if err != nil {
			return
		}
		cmd := strings.ToUpper(linea)
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			tp.PrintfLine("250 smtpfalso")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			r = Recibido{De: entreAngulos(linea)}
			tp.PrintfLine("250 ok")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			if s.rechazar.Load() {
				tp.PrintfLine("550 rechazado")
				continue
			}
			r.Para = append(r.Para, entreAngulos(linea))
			tp.PrintfLine("250 ok")
		case cmd == "DATA":
			tp.PrintfLine("354 dale")
			datos, err := tp.ReadDotBytes()
			if err != nil {
				return
			}
			r.Datos = string(datos)
			s.mu.Lock()
			s.recibidos = append(s.recibidos, r)
			s.mu.Unlock()
			tp.PrintfLine("250 ok")
		case cmd == "QUIT":
			tp.PrintfLine("221 chau")
			return
		default:
			tp.PrintfLine("250 ok")
		}
	}
}

func entreAngulos(s string) string {
	i, j := strings.Index(s, "<"), strings.LastIndex(s, ">")
	if i < 0 || j < i {
		return ""
	}
	return s[i+1 : j]
}
