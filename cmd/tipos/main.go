// tipos genera cliente/plantillas.d.ts desde las plantillas y sistemas.json.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/GPE-Sistemas/mensajero"
	"github.com/GPE-Sistemas/mensajero/internal/plantillas"
	"github.com/GPE-Sistemas/mensajero/internal/sistemas"
)

func main() {
	salidaPath := flag.String("o", "cliente/plantillas.d.ts", "archivo a generar")
	flag.Parse()
	b, err := salida()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*salidaPath, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func salida() ([]byte, error) {
	cat, err := plantillas.Cargar(mensajero.Archivos, "plantillas")
	if err != nil {
		return nil, err
	}
	f, err := mensajero.Archivos.Open("sistemas.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sis, err := sistemas.Cargar(f)
	if err != nil {
		return nil, err
	}
	return generar(cat, sis), nil
}

func generar(cat *plantillas.Catalogo, sis map[string]sistemas.Sistema) []byte {
	var b bytes.Buffer
	b.WriteString("// Generado por `make generar` desde plantillas/ y sistemas.json. No editar.\n\n")

	b.WriteString("export interface Plantillas {\n")
	actual := ""
	for _, p := range cat.Todas() { // ordenadas por sistema/nombre
		if p.Sistema != actual {
			if actual != "" {
				b.WriteString("  };\n")
			}
			fmt.Fprintf(&b, "  '%s': {\n", p.Sistema)
			actual = p.Sistema
		}
		campos := make([]string, len(p.Campos))
		for i, c := range p.Campos {
			campos[i] = c + ": string"
		}
		fmt.Fprintf(&b, "    '%s': { %s };\n", p.Nombre, strings.Join(campos, "; "))
	}
	if actual != "" {
		b.WriteString("  };\n")
	}
	b.WriteString("}\n\n")

	b.WriteString("export interface Remitentes {\n")
	nombres := make([]string, 0, len(sis))
	for n := range sis {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	for _, n := range nombres {
		var rs []string
		for r := range sis[n].Remitentes {
			rs = append(rs, "'"+r+"'")
		}
		sort.Strings(rs)
		fmt.Fprintf(&b, "  '%s': %s;\n", n, strings.Join(rs, " | "))
	}
	b.WriteString("}\n")
	return b.Bytes()
}
