package documento_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	firmantes "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/catalogo"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	expedienteRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paqueteRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	expedienteApp "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	sesionEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	exp "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type relojPropuestaSalida struct{ dia time.Time }

func (r *relojPropuestaSalida) Ahora() time.Time { return r.dia }

type sesionesSalida struct {
	sesiones
	perfil identidad.Perfil
	equipo string
}

func (s sesionesSalida) Consultar(context.Context, string) (sesionEntrada.Acceso, error) {
	return sesionEntrada.Acceso{Estado: identidad.Activo, Usuario: identidad.Usuario{ID: usuario, Estado: identidad.Activo, Perfiles: []identidad.Perfil{s.perfil}, EquipoID: s.equipo}}, nil
}

func TestAnexo6PostgresHTTPDescargaHitosPermisos(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 110*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	r := postgres.NuevoRegistroAuditoria(b)
	er := &expedienteRepo.Repositorio{}
	relojSalida := &relojPropuestaSalida{dia: time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)}
	salida := expedienteApp.Nuevo(expedienteApp.Dependencias{SalidasServicio: er, Repositorio: er, Catalogo: &catalogo.Repositorio{}, Transacciones: b, Registro: r, RegistroTransaccional: r, Reloj: relojSalida, Identificadores: ids{}})
	archivos := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	propuesta := &relojPropuestaSalida{dia: time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)}
	datos := app.DatosEntrada{Salidas: salida, Catalogos: &catalogo.Repositorio{}, Referencias: &paqueteRepo.Referencias{}, Presentacion: archivos, Reloj: propuesta}
	deps := app.Dependencias{Repositorio: repo.Repositorio{}, Datos: datos, Firmantes: firmantes.Firmantes{Catalogo: &catalogo.Repositorio{}}, Motor: word.Motor{}, Idiomas: archivos, Ficheros: archivos, Transacciones: b, Registro: r, RegistroTransaccional: r, Reloj: reloj{}, Identificadores: ids{}}
	documentos := app.Nuevo(deps)
	a := entrada.Actor{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}
	var equipo, ajeno string
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT e.id::text,q.id::text FROM catalogo.equipo e,catalogo.equipo q WHERE e.codigo='06' AND q.codigo='08'`).Scan(&equipo, &ajeno) //etf:tecnico
	})
	if err != nil {
		t.Fatal(err)
	}
	servidor := func(perfil identidad.Perfil, equipo string) *httptest.Server {
		return httptest.NewServer(adaptadorHTTP.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: sesionesSalida{perfil: perfil, equipo: equipo}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Documentos: adaptadorHTTP.ConfiguracionDocumentos{Servicio: documentos}, Expedientes: adaptadorHTTP.ConfiguracionAceptaciones{SalidasServicio: salida}}))
	}
	hacer := func(s *httptest.Server, metodo, ruta, clave string, p any, csrf bool) (int, http.Header, []byte) {
		t.Helper()
		cookie := &http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"}
		inicio, _ := http.NewRequestWithContext(ctx, "GET", s.URL+"/api/sesion", nil)
		inicio.AddCookie(cookie)
		res, err := s.Client().Do(inicio)
		if err != nil {
			t.Fatal(err)
		}
		var sesion struct {
			CSRF string `json:"csrf"`
		}
		err = json.NewDecoder(res.Body).Decode(&sesion)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		cuerpo, _ := json.Marshal(p)
		req, _ := http.NewRequestWithContext(ctx, metodo, s.URL+ruta, bytes.NewReader(cuerpo))
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://etf.interno")
		req.Header.Set("Idempotency-Key", clave)
		if csrf {
			req.Header.Set("X-CSRF-Token", sesion.CSRF)
		}
		res, err = s.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		contenido, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, res.Header, contenido
	}
	s := servidor(identidad.ProfesionalETF, equipo)
	defer s.Close()
	profesional := entrada.Actor{UsuarioID: usuario, Perfiles: []string{"profesional_etf"}, EquipoID: equipo}
	for _, n := range []int{1, 6, 15} {
		id := fmt.Sprintf("aaaaaaaa-7777-4777-8777-%012d", n)
		ruta := "/api/expedientes/" + id
		p := entrada.NuevaSalidaServicio{Tipo: "otro_servicio", Version: 2, Servicio: "SALUD", Motivo: "Coordinación de la atención familiar"}
		codigo, _, contenido := hacer(s, "POST", ruta+"/derivaciones-salida", fmt.Sprintf("salida-%d", n), p, true)
		if codigo != 200 {
			t.Fatalf("crear %d: %s", codigo, contenido)
		}
		var creada entrada.SalidaServicio
		if json.Unmarshal(contenido, &creada) != nil || creada.VersionExpediente != 3 || creada.Fecha != "" {
			t.Fatalf("recibo: %s", contenido)
		}
		codigo, _, repetida := hacer(s, "POST", ruta+"/derivaciones-salida", fmt.Sprintf("salida-%d", n), p, true)
		if codigo != 200 || !bytes.Equal(contenido, repetida) {
			t.Fatalf("repetición: %d %s", codigo, repetida)
		}
		p.Motivo = "Coordinación del seguimiento sanitario"
		codigo, _, _ = hacer(s, "POST", ruta+"/derivaciones-salida", fmt.Sprintf("salida-%d", n), p, true)
		if codigo != 422 {
			t.Fatal("clave alterada", codigo)
		}
		codigo, _, _ = hacer(s, "POST", ruta+"/derivaciones-salida", fmt.Sprintf("salida-conflicto-%d", n), p, true)
		if codigo != 409 {
			t.Fatal("versión expediente", codigo)
		}
		for _, idioma := range []string{"es", "en"} {
			clave := fmt.Sprintf("anexo6-%d-%s", n, idioma)
			solicitud := map[string]string{"idioma": idioma, "salida_id": creada.ID}
			codigo, cabeceras, contenido := hacer(s, "POST", ruta+"/documentos/anexo6", clave, solicitud, true)
			if codigo != 201 {
				t.Fatalf("emitir %d: %s", codigo, contenido)
			}
			if !strings.Contains(cabeceras.Get("Content-Type"), "wordprocessingml") || cabeceras.Get("X-Emision-ID") == "" {
				t.Fatal("descarga Word")
			}
			xml := xmlDocumento(t, contenido)
			for _, nombre := range []string{"Nerea", "Álvaro", "Celia", "Mateo", "Irene", "Hugo", "Paula", "Bruno", "Aitana", "Leo", "Claudia", "Mario", "Vera", "Lucas", "Eva"}[:n] {
				if !strings.Contains(xml, nombre+" Ortega Salas") {
					t.Fatal("menor ausente", nombre)
				}
			}
			if strings.Contains(xml, "{{") || strings.Contains(xml, "Haga clic") || strings.Contains(xml, p.Motivo) || strings.Contains(xml, "Coordinación de la atención familiar") || !strings.Contains(xml, "Lucía") {
				t.Fatal("relleno o apartados narrativos")
			}
			propuesta.dia = propuesta.dia.Add(24 * time.Hour)
			codigo, _, descarga := hacer(s, "POST", ruta+"/documentos/anexo6", clave, solicitud, true)
			propuesta.dia = propuesta.dia.Add(-24 * time.Hour)
			if codigo != 200 || !bytes.Equal(contenido, descarga) {
				t.Fatal("idempotencia Word", codigo)
			}
		}
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: profesional.Perfiles, EquipoID: equipo}, func(ctx context.Context) error {
			e, err := er.Obtener(ctx, id)
			if err != nil {
				return err
			}
			if e.Version != 3 || e.Estado != "aceptada" {
				t.Fatal("emisión altera expediente")
			}
			tx, _ := postgres.DesdeContexto(ctx)
			var hitos int
			err = tx.QueryRow(ctx, `SELECT count(*) FROM expediente.salida_servicio_hito WHERE expediente_id=$1`, id).Scan(&hitos)
			if hitos != 0 {
				t.Fatal("emisión crea hito")
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if n == 1 {
			relojSalida.dia = time.Date(2026, 10, 12, 22, 30, 0, 0, time.UTC)
			hoy := entrada.ConfirmarSalidaServicio{Tipo: "derivacion_otro_servicio", SalidaID: creada.ID, Version: 3, VersionSalida: 1, Fecha: "2026-10-13"}
			// Forzamos conflicto de salida para comprobar que hoy Madrid supera la validación de fecha.
			hoy.VersionSalida = 99
			codigo, _, _ = hacer(s, "POST", ruta+"/hitos", "hito-hoy-madrid", hoy, true)
			if codigo != 409 {
				t.Fatal("fecha civil de Madrid rechazada", codigo)
			}
			hoy.Fecha = "2026-10-14"
			codigo, _, _ = hacer(s, "POST", ruta+"/hitos", "hito-futuro", hoy, true)
			if codigo != 400 {
				t.Fatal("fecha futura admitida", codigo)
			}
			relojSalida.dia = time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
		}
		h := entrada.ConfirmarSalidaServicio{Tipo: "derivacion_otro_servicio", SalidaID: creada.ID, Version: 3, VersionSalida: 1, Fecha: "2026-10-12"}
		codigo, _, contenido = hacer(s, "POST", ruta+"/hitos", fmt.Sprintf("hito-%d", n), h, true)
		if codigo != 200 {
			t.Fatalf("hito %d: %s", codigo, contenido)
		}
		codigo, _, repetida = hacer(s, "POST", ruta+"/hitos", fmt.Sprintf("hito-%d", n), h, true)
		if codigo != 200 || !bytes.Equal(contenido, repetida) {
			t.Fatal("repetición hito", codigo)
		}
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: profesional.Perfiles, EquipoID: equipo}, func(ctx context.Context) error {
			e, err := er.Obtener(ctx, id)
			if err != nil {
				return err
			}
			r, err := er.LeerSalida(ctx, id, creada.ID)
			if err != nil {
				return err
			}
			if e.Version != 4 || e.Estado != "aceptada" || r.Estado != "emitida" || r.Fecha != "2026-10-12" || r.Version != 2 {
				t.Fatal("hito incorrecto")
			}
			tx, _ := postgres.DesdeContexto(ctx)
			var total int
			err = tx.QueryRow(ctx, `SELECT count(*) FROM expediente.salida_servicio_hito WHERE expediente_id=$1`, id).Scan(&total)
			if total != 1 {
				t.Fatal("hito duplicado")
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		h.Version = 4
		h.VersionSalida = 1
		codigo, _, _ = hacer(s, "POST", ruta+"/hitos", fmt.Sprintf("hito-version-%d", n), h, true)
		if codigo != 409 {
			t.Fatal("versión salida", codigo)
		}
		// Una nueva emisión puede usar los datos confirmados sin cambiar la fecha real.
		codigo, _, _ = hacer(s, "POST", ruta+"/documentos/anexo6", fmt.Sprintf("anexo6-posterior-%d", n), map[string]string{"idioma": "es", "salida_id": creada.ID}, true)
		if codigo != 201 {
			t.Fatal("regeneración", codigo)
		}
		for _, perfil := range []identidad.Perfil{identidad.Consulta, identidad.Administracion} {
			denegado := servidor(perfil, equipo)
			for _, endpoint := range []string{"/derivaciones-salida", "/hitos", "/documentos/anexo6"} {
				codigo, _, _ = hacer(denegado, "POST", ruta+endpoint, "denegado", p, true)
				if codigo != 403 {
					t.Fatal("perfil", perfil, codigo)
				}
			}
			denegado.Close()
		}
		otro := servidor(identidad.ProfesionalETF, ajeno)
		codigo, _, _ = hacer(otro, "POST", ruta+"/derivaciones-salida", "otro", p, true)
		if codigo != 404 {
			t.Fatal("equipo ajeno", codigo)
		}
		codigo, _, _ = hacer(otro, "POST", ruta+"/documentos/anexo6", "otro-word", map[string]string{"idioma": "es", "salida_id": creada.ID}, true)
		if codigo != 403 {
			t.Fatal("Word ajeno", codigo)
		}
		otro.Close()
		codigo, _, _ = hacer(s, "POST", ruta+"/hitos", "sin-csrf", h, false)
		if codigo != 403 {
			t.Fatal("CSRF", codigo)
		}
		codigo, _, _ = hacer(s, "GET", ruta+"/documentos/anexo6", "get", nil, true)
		if codigo != 405 {
			t.Fatal("método", codigo)
		}

		if n == 6 {
			// La ficha editada debe sustituir los valores iniciales, incluso al retirar menores.
			err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: profesional.Perfiles, EquipoID: equipo}, func(ctx context.Context) error {
				familia := exp.Familia{Direccion: "Calle del Almendro 24", Telefono: "600987321"}
				if err := er.GuardarFamilia(ctx, id, exp.CambioFamilia{Version: 4, Desde: "2026-10-13", Familia: &familia}, usuario, reloj{}.Ahora()); err != nil {
					return err
				}
				ficha, err := er.ObtenerMenores(ctx, id)
				if err != nil {
					return err
				}
				if len(ficha.Menores) == 0 {
					return errors.New("expediente.menores_invalidos")
				}
				menores := ficha.Menores[:1]
				menores[0].Nombre = "Marina"
				menores[0].Apellidos = "Serrano Vega"
				menores[0].CentroEducativo = "Colegio del Olivar"
				if err := er.GuardarMenores(ctx, id, exp.CambioMenores{Version: 5, Desde: "2026-10-13", Menores: &menores}, usuario, reloj{}.Ahora()); err != nil {
					return err
				}
				_, err = r.RegistrarEnTransaccion(ctx, auditoria.Datos{UsuarioID: usuario, Perfiles: profesional.Perfiles, Accion: "expediente.editar_menores", Entidad: "expediente", EntidadID: id, Resultado: "ok", OcurridoEn: reloj{}.Ahora()})
				return err
			})
			if err != nil {
				t.Fatal("edición de ficha", err)
			}
			codigo, _, contenido = hacer(s, "POST", ruta+"/documentos/anexo6", "anexo6-ficha-vigente", map[string]string{"idioma": "es", "salida_id": creada.ID}, true)
			if codigo != 201 {
				t.Fatalf("Word vigente: %d %s", codigo, contenido)
			}
			xml := xmlDocumento(t, contenido)
			if !strings.Contains(xml, "Calle del Almendro 24") || !strings.Contains(xml, "600987321") || !strings.Contains(xml, "Marina Serrano Vega") || !strings.Contains(xml, "Colegio del Olivar") || strings.Contains(xml, "Calle del Laurel 18") || strings.Contains(xml, "Álvaro Ortega Salas") {
				t.Fatal("Word conserva datos antiguos")
			}
			t.Log("AN6-FICHA-OK: dirección y colección de menores vigentes; emisión sin alterar el hito")
			err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: profesional.Perfiles, EquipoID: equipo}, func(ctx context.Context) error {
				vacios := []exp.Menor{}
				return er.GuardarMenores(ctx, id, exp.CambioMenores{Version: 6, Desde: "2026-10-13", Menores: &vacios}, usuario, reloj{}.Ahora())
			})
			if err != nil {
				t.Fatal(err)
			}
			codigo, _, contenido = hacer(s, "POST", ruta+"/documentos/anexo6", "anexo6-sin-menores", map[string]string{"idioma": "es", "salida_id": creada.ID}, true)
			if codigo != 201 || strings.Contains(xmlDocumento(t, contenido), "Marina Serrano Vega") {
				t.Fatalf("colección vacía no respetada %d: %s", codigo, contenido)
			}
			t.Log("AN6-VACIOS-OK: menores retirados no se recuperan del anexo original")

		}
		// Segunda barrera: las tablas nuevas tampoco muestran filas al perfil incorrecto.
		for _, actor := range []plataforma.Identidad{{UsuarioID: usuario, Perfiles: []string{"consulta"}}, {UsuarioID: usuario, Perfiles: []string{"administracion"}}, {UsuarioID: usuario, Perfiles: []string{"profesional_etf"}, EquipoID: ajeno}} {
			err = b.Ejecutar(ctx, actor, func(ctx context.Context) error {
				tx, _ := postgres.DesdeContexto(ctx)
				for _, tabla := range []string{"salida_servicio", "salida_servicio_hito", "salida_servicio_peticion"} {
					var total int
					if err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.`+tabla+` WHERE expediente_id=$1`, id).Scan(&total); err != nil {
						return err
					}
					if total != 0 {
						t.Fatal("RLS", tabla, total)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			err = b.Ejecutar(ctx, actor, func(ctx context.Context) error {
				tx, _ := postgres.DesdeContexto(ctx)
				var actual int
				var coincide bool
				return tx.QueryRow(ctx, `SELECT actual,coincide FROM expediente.version_salida_servicio($1,4,true)`, id).Scan(&actual, &coincide)
			})
			if err == nil {
				t.Fatal("función permite principal ajeno")
			}
		}
	}
	// Si falla el apunte, no persisten salida, versión ni respuesta idempotente.
	fallo := expedienteApp.Nuevo(expedienteApp.Dependencias{SalidasServicio: er, Catalogo: &catalogo.Repositorio{}, Transacciones: b, Registro: r, RegistroTransaccional: registroFallido{}, Reloj: reloj{}, Identificadores: ids{}})
	id := "aaaaaaaa-7777-4777-8777-000000000001"
	_, err = fallo.CrearSalidaServicio(ctx, profesional, id, "salida-auditoria-fallida", entrada.NuevaSalidaServicio{Tipo: "otro_servicio", Version: 4, Servicio: "SALUD", Motivo: "Coordinación familiar"})
	if !errors.Is(err, entrada.ErrNoDisponible) {
		t.Fatal("auditoría fallida", err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: profesional.Perfiles, EquipoID: equipo}, func(ctx context.Context) error {
		e, err := er.Obtener(ctx, id)
		if err != nil {
			return err
		}
		if e.Version != 4 {
			t.Fatal("versión sin auditoría")
		}
		tx, _ := postgres.DesdeContexto(ctx)
		var salidas, recibos int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM expediente.salida_servicio WHERE expediente_id=$1`, id).Scan(&salidas); err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM expediente.salida_servicio_peticion WHERE clave='salida-auditoria-fallida'`).Scan(&recibos); err != nil {
			return err
		}
		if salidas != 1 || recibos != 0 {
			t.Fatal("escritura sin auditoría", salidas, recibos)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	coordinacion := servidor(identidad.Coordinacion, "")
	defer coordinacion.Close()
	p := entrada.NuevaSalidaServicio{Tipo: "otro_servicio", Version: 4, Servicio: "SALUD", Motivo: "Seguimiento de la atención familiar"}
	codigo, _, contenido := hacer(coordinacion, "POST", "/api/expedientes/"+id+"/derivaciones-salida", "coordinacion-salida", p, true)
	if codigo != 200 {
		t.Fatalf("coordinación %d: %s", codigo, contenido)
	}
	var creada entrada.SalidaServicio
	if err := json.Unmarshal(contenido, &creada); err != nil {
		t.Fatal(err)
	}
	codigo, _, contenido = hacer(coordinacion, "POST", "/api/expedientes/"+id+"/documentos/anexo6", "coordinacion-word", map[string]string{"idioma": "es", "salida_id": creada.ID}, true)
	if codigo != 201 || !strings.Contains(xmlDocumento(t, contenido), "Nerea") {
		t.Fatal("Word coordinación", codigo)
	}
	h := entrada.ConfirmarSalidaServicio{Tipo: "derivacion_otro_servicio", SalidaID: creada.ID, Version: 5, VersionSalida: 1, Fecha: "2026-10-13"}
	codigo, _, contenido = hacer(coordinacion, "POST", "/api/expedientes/"+id+"/hitos", "coordinacion-hito", h, true)
	if codigo != 200 {
		t.Fatalf("hito coordinación %d: %s", codigo, contenido)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var total int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.salida_servicio_hito WHERE expediente_id=$1`, id).Scan(&total)
		if total != 2 {
			t.Fatal("varias derivaciones", total)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ANEXO6-OK: descarga HTTP es/en, 1/6/15 menores, hitos separados, idempotencia tras cambio de día, versiones 409, Coordinación, perfiles, equipo ajeno, RLS y rollback de auditoría")
}
