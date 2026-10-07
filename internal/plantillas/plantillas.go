// Package plantillas carga y renderiza las plantillas de mail de cada sistema.
package plantillas

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	htemplate "html/template"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"
	ttemplate "text/template"
	"text/template/parse"
)

// ErrDatos indica que los datos no coinciden con lo que pide la plantilla.
var ErrDatos = errors.New("datos inválidos")

type Plantilla struct {
	Sistema string
	Nombre  string
	Campos  []string

	asunto *ttemplate.Template
	html   *htemplate.Template
	texto  *ttemplate.Template // nil si no hay cuerpo.txt
}

type Renderizado struct {
	Asunto string
	HTML   string
	Texto  string
}

type Catalogo struct {
	plantillas map[string]*Plantilla
}

// Cargar lee raiz/<sistema>/base.html y raiz/<sistema>/<plantilla>/.
func Cargar(fsys fs.FS, raiz string) (*Catalogo, error) {
	c := &Catalogo{plantillas: map[string]*Plantilla{}}
	sistemas, err := fs.ReadDir(fsys, raiz)
	if err != nil {
		return nil, err
	}
	for _, s := range sistemas {
		if !s.IsDir() {
			continue
		}
		dir := path.Join(raiz, s.Name())
		base, err := fs.ReadFile(fsys, path.Join(dir, "base.html"))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.Name(), err)
		}
		entradas, err := fs.ReadDir(fsys, dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entradas {
			if !e.IsDir() {
				continue
			}
			p, err := cargarPlantilla(fsys, path.Join(dir, e.Name()), string(base))
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", s.Name(), e.Name(), err)
			}
			p.Sistema, p.Nombre = s.Name(), e.Name()
			c.plantillas[s.Name()+"/"+e.Name()] = p
		}
	}
	return c, nil
}

func cargarPlantilla(fsys fs.FS, dir, base string) (*Plantilla, error) {
	leer := func(nombre string) (string, error) {
		b, err := fs.ReadFile(fsys, path.Join(dir, nombre))
		return string(b), err
	}
	asunto, err := leer("asunto.txt")
	if err != nil {
		return nil, err
	}
	cuerpo, err := leer("cuerpo.html")
	if err != nil {
		return nil, err
	}

	p := &Plantilla{}
	if p.asunto, err = ttemplate.New("asunto").Option("missingkey=error").Parse(strings.TrimSpace(asunto)); err != nil {
		return nil, err
	}
	if p.html, err = htemplate.New("base").Option("missingkey=error").Parse(base); err != nil {
		return nil, err
	}
	if _, err = p.html.New("cuerpo").Parse(cuerpo); err != nil {
		return nil, err
	}
	texto, err := leer("cuerpo.txt")
	switch {
	case err == nil:
		if p.texto, err = ttemplate.New("texto").Option("missingkey=error").Parse(texto); err != nil {
			return nil, err
		}
	case !errors.Is(err, fs.ErrNotExist):
		return nil, err
	}

	var arboles []*parse.Tree
	for _, t := range p.asunto.Templates() {
		arboles = append(arboles, t.Tree)
	}
	for _, t := range p.html.Templates() {
		arboles = append(arboles, t.Tree)
	}
	if p.texto != nil {
		for _, t := range p.texto.Templates() {
			arboles = append(arboles, t.Tree)
		}
	}
	campos := map[string]bool{}
	for _, a := range arboles {
		if a == nil {
			continue
		}
		if err := juntarCampos(a.Root, campos); err != nil {
			return nil, err
		}
	}
	for c := range campos {
		p.Campos = append(p.Campos, c)
	}
	sort.Strings(p.Campos)
	return p, nil
}

// juntarCampos anota los {{.campo}} que usa la plantilla y rechaza lo que el cliente TS no puede tipar.
func juntarCampos(n parse.Node, campos map[string]bool) error {
	switch n := n.(type) {
	case *parse.ListNode:
		if n == nil {
			return nil
		}
		for _, h := range n.Nodes {
			if err := juntarCampos(h, campos); err != nil {
				return err
			}
		}
	case *parse.ActionNode:
		return juntarCampos(n.Pipe, campos)
	case *parse.PipeNode:
		if n == nil {
			return nil
		}
		for _, c := range n.Cmds {
			if err := juntarCampos(c, campos); err != nil {
				return err
			}
		}
	case *parse.CommandNode:
		for _, a := range n.Args {
			if err := juntarCampos(a, campos); err != nil {
				return err
			}
		}
	case *parse.FieldNode:
		if len(n.Ident) != 1 {
			return fmt.Errorf("campo anidado no soportado: .%s", strings.Join(n.Ident, "."))
		}
		campos[n.Ident[0]] = true
	case *parse.IfNode:
		for _, h := range []parse.Node{n.Pipe, n.List, n.ElseList} {
			if err := juntarCampos(h, campos); err != nil {
				return err
			}
		}
	case *parse.TemplateNode:
		// {{template "x" .}} pasa todos los datos tal cual: es el único uso permitido de {{.}}.
		if n.Pipe != nil && len(n.Pipe.Cmds) == 1 && len(n.Pipe.Cmds[0].Args) == 1 {
			if _, ok := n.Pipe.Cmds[0].Args[0].(*parse.DotNode); ok {
				return nil
			}
		}
		return juntarCampos(n.Pipe, campos)
	case *parse.RangeNode:
		return errors.New("range no soportado: los datos son strings")
	case *parse.WithNode:
		return errors.New("with no soportado")
	case *parse.TextNode, *parse.CommentNode, *parse.IdentifierNode,
		*parse.StringNode, *parse.NumberNode, *parse.BoolNode:
	default:
		return fmt.Errorf("construcción no soportada: %s", n)
	}
	return nil
}

func (c *Catalogo) Buscar(sistema, nombre string) (*Plantilla, bool) {
	p, ok := c.plantillas[sistema+"/"+nombre]
	return p, ok
}

func (c *Catalogo) Todas() []*Plantilla {
	claves := make([]string, 0, len(c.plantillas))
	for k := range c.plantillas {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	todas := make([]*Plantilla, len(claves))
	for i, k := range claves {
		todas[i] = c.plantillas[k]
	}
	return todas
}

// Renderizar exige exactamente los campos de la plantilla, ni uno más ni uno menos.
func (p *Plantilla) Renderizar(datos map[string]string) (Renderizado, error) {
	for _, c := range p.Campos {
		if _, ok := datos[c]; !ok {
			return Renderizado{}, fmt.Errorf("%w: falta %q", ErrDatos, c)
		}
	}
	for k := range datos {
		if !slices.Contains(p.Campos, k) {
			return Renderizado{}, fmt.Errorf("%w: sobra %q", ErrDatos, k)
		}
	}

	var asunto, html, texto bytes.Buffer
	if err := p.asunto.Execute(&asunto, datos); err != nil {
		return Renderizado{}, err
	}
	a := strings.TrimSpace(asunto.String())
	if strings.ContainsAny(a, "\r\n") {
		return Renderizado{}, fmt.Errorf("%w: el asunto queda con saltos de línea", ErrDatos)
	}
	if err := p.html.ExecuteTemplate(&html, "base", datos); err != nil {
		return Renderizado{}, err
	}
	if p.texto != nil {
		if err := p.texto.Execute(&texto, datos); err != nil {
			return Renderizado{}, err
		}
	}
	return Renderizado{Asunto: a, HTML: html.String(), Texto: texto.String()}, nil
}

// Ejemplo lee los datos de ejemplo de una plantilla: alimentan la vista previa y los tests.
func Ejemplo(fsys fs.FS, raiz, sistema, nombre string) (map[string]string, error) {
	b, err := fs.ReadFile(fsys, path.Join(raiz, sistema, nombre, "ejemplo.json"))
	if err != nil {
		return nil, err
	}
	var d map[string]string
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("%s/%s/ejemplo.json: %w", sistema, nombre, err)
	}
	return d, nil
}
