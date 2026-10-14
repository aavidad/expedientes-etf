package http

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	"io/fs"
	"log/slog"
	"net/http"
)

func modulosPredeterminados(o Opciones, sesion *sesionHTTP) []Modulo {
	return append([]Modulo{ModuloPlataforma{sesion: sesion}}, modulosNegocio(o)...)
}

func modulosNegocio(o Opciones) []Modulo {
	if o.Modulos != nil {
		return o.Modulos
	}
	return []Modulo{
		ModuloBandeja{Servicio: o.Bandeja}, ModuloSalidas{}, ModuloSMI{o.SMI}, ModuloUsuarios{o.Usuarios}, ModuloCatalogos{o.Catalogos},
		ModuloInstalacion{o.Instalacion}, ModuloDerivaciones{o.Derivaciones},
		ModuloExpedientes{o.Expedientes}, ModuloLecturaMemoria{}, ModuloDocumentos{o.Documentos}, ModuloActividad{o.Actividad}, ModuloSPAF{o.SPAF}, ModuloAnexo{}, ModuloDecision{}, ModuloPlantillas{},
		ModuloEquipo{o.Equipo}, ModuloFichaMenoresIntervencion{}, ModuloTraslado{}, ModuloCarga{}, ModuloConservacion{},
	}
}

func componerMux(paquetes entrada.Paquetes, web fs.FS, idioma, tema string, avisos []string, comprobarBase func(context.Context) error, o Opciones) (*muxHTTP, *sesionHTTP, error) {
	sesion := nuevaSesionHTTP(o.Sesion)
	plataforma := ModuloPlataforma{paquetes: paquetes, web: web, idioma: idioma, tema: tema, avisos: avisos, comprobarBase: comprobarBase, sesion: sesion}
	rutas := plataforma.Rutas()
	pagina := rutas[0].Handler
	for _, ruta := range rutas {
		if ruta.Patron == "/{$}" {
			pagina = ruta.Handler
		}
	}
	for _, modulo := range modulosNegocio(o) {
		if modulo == nil {
			return nil, nil, errors.New("http.registro_invalido")
		}
		propias := modulo.Rutas()
		if len(propias) == 0 {
			return nil, nil, errors.New("http.registro_invalido")
		}
		rutas = append(rutas, propias...)
	}
	if err := ValidarRutas(rutas); err != nil {
		return nil, nil, err
	}
	mux := nuevoMuxHTTP()
	porPatron := map[string][]Ruta{}
	for _, ruta := range rutas {
		if ruta.Pantalla && ruta.Handler == nil {
			ruta.Handler = pagina
		}
		mux.rutas = append(mux.rutas, ruta)
		sesion.rutas = append(sesion.rutas, ruta.ReglaRuta)
		porPatron[ruta.Patron] = append(porPatron[ruta.Patron], ruta)
	}
	for patron, entradas := range porPatron {
		mux.HandleFunc(patron, func(w http.ResponseWriter, r *http.Request) {
			for _, ruta := range entradas {
				for _, metodo := range ruta.Metodos {
					if metodo == r.Method {
						ruta.Handler.ServeHTTP(w, r)
						return
					}
				}
			}
			responderError(w, http.StatusMethodNotAllowed, "http.metodo_no_permitido")
		})
	}
	return mux, sesion, nil
}

// nuevoMux mantiene el punto de inspección de las pruebas anteriores.
func nuevoMux(paquetes entrada.Paquetes, web fs.FS, idioma, tema string, avisos []string, registro *slog.Logger, comprobarBase func(context.Context) error, opciones Opciones) (*muxHTTP, *sesionHTTP) {
	mux, sesion, err := componerMux(paquetes, web, idioma, tema, avisos, comprobarBase, opciones)
	if err != nil {
		panic(err)
	}
	return mux, sesion
}
