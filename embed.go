// Package mensajero embebe las plantillas y la configuración de sistemas en el binario:
// cada versión de plantillas es una versión de la imagen.
package mensajero

import "embed"

//go:generate go run ./cmd/tipos -o cliente/plantillas.d.ts

//go:embed plantillas sistemas.json
var Archivos embed.FS
