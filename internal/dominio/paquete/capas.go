package paquete

import "path"

// CapasTextos devuelve, por orden de prioridad, las carpetas de idioma cuyos
// textos se combinan: la versión activa, la distribución del mismo idioma y
// la distribución del castellano. Una versión publicada antes de una
// actualización no conoce las claves nuevas; las capas siguientes sólo
// aportan las que faltan, nunca sustituyen las de una capa anterior.
func CapasTextos(activa string) []string {
	codigo := path.Base(activa)
	capas := []string{activa}
	for _, c := range []string{codigo, IdiomaPredeterminado} {
		repetida := false
		for _, existente := range capas {
			repetida = repetida || existente == c
		}
		if !repetida {
			capas = append(capas, c)
		}
	}
	return capas
}

// CompletarTextos añade a destino sólo las claves que todavía no tiene.
func CompletarTextos(destino, capa map[string]string) {
	for clave, valor := range capa {
		if _, existe := destino[clave]; !existe {
			destino[clave] = valor
		}
	}
}
