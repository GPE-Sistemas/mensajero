package sistemas

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// Claves guarda el SHA-256 de la apikey de cada sistema. Nunca la apikey en claro.
type Claves map[string][32]byte

// CargarClaves lee {"<sistema>": "<sha256 en hex>"}, el valor de la env CLAVES.
func CargarClaves(j string) (Claves, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(j), &m); err != nil {
		return nil, fmt.Errorf("CLAVES: %w", err)
	}
	if len(m) == 0 {
		return nil, errors.New("CLAVES: no hay ninguna")
	}
	c := Claves{}
	dueño := map[[32]byte]string{}
	for s, h := range m {
		b, err := hex.DecodeString(h)
		if err != nil || len(b) != sha256.Size {
			return nil, fmt.Errorf("CLAVES: el hash de %q no es un SHA-256 en hex", s)
		}
		k := [32]byte(b)
		if otro, ok := dueño[k]; ok {
			a, z := min(s, otro), max(s, otro)
			return nil, fmt.Errorf("CLAVES: %q y %q tienen el mismo hash", a, z)
		}
		dueño[k] = s
		c[s] = k
	}
	return c, nil
}

// Sistema devuelve a qué sistema pertenece la apikey.
func (c Claves) Sistema(apikey string) (string, bool) {
	h := sha256.Sum256([]byte(apikey))
	encontrado := ""
	for s, ref := range c {
		if subtle.ConstantTimeCompare(h[:], ref[:]) == 1 {
			encontrado = s
		}
	}
	return encontrado, encontrado != ""
}
