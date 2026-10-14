package catalogo_test

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	refs "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	paquete "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type auditoriaFallida struct{}

func (auditoriaFallida) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("auditoria.registro_fallido")
}
func TestIntegracionEdicionAtomicaHistoriaYCarrera(t *testing.T) {
	b, ctx := base(t)
	// La misma base conserva paquetes semver de otros idiomas. La limpieza
	// del catálogo debe consultar sólo sus instantáneas SHA-256.
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		huella := strings.Repeat("b", 64)
		ruta := ".versiones/it/1.0.0/" + huella + "/" + strings.Repeat("c", 32) + "/it"
		_, err = tx.Exec(ctx, `SELECT paquete.activar('idioma','it','1.0.0',$1,$2,0)`, huella, ruta)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, cod := range []string{"es", "en"} {
		destino := filepath.Join(dir, cod, "catalogos")
		if err := os.MkdirAll(destino, 0700); err != nil {
			t.Fatal(err)
		}
		for _, nombre := range []string{"listas.json", "ubicaciones.json"} {
			contenido, err := os.ReadFile("../../../../idiomas/" + cod + "/catalogos/" + nombre)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(destino, nombre), contenido, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	disco := ficheros.NuevosPaquetes(dir, "")
	publicacion := paquetes.NuevaPublicacion(disco, &refs.Referencias{})
	a := postgres.NuevoRegistroAuditoriaTransaccional(b)
	editor := app.NuevaEdicion(&repo.Repositorio{}, publicacion, b, a, time.Now)
	actor := entrada.Actor{UsuarioID: actorID, Perfiles: []string{"administracion"}}
	fecha := func(anio int) time.Time { return time.Date(anio, 1, 1, 0, 0, 0, 0, time.UTC) }
	estado, err := editor.Consultar(ctx, actor, "listas")
	if err != nil {
		t.Fatal(err)
	}
	c := entrada.Cambio{VersionEsperada: estado.Datos.Version.Numero, ReferenciasEsperadas: estado.Referencias, VigenteDesde: fecha(2030), Opcion: &dominio.Opcion{Lista: "pronostico", Codigo: "ESTABLE", Orden: 90, Activa: true}, Textos: map[string]string{"es": "Evolución estable", "en": "Stable progress"}}
	s1, err := editor.Guardar(ctx, actor, "listas", c)
	if err != nil {
		t.Fatal(err)
	}
	if len(s1.Datos.Opciones) != len(estado.Datos.Opciones)+1 || s1.Datos.Version.Numero != estado.Datos.Version.Numero+1 || s1.Referencias["es"] != 1 || s1.Textos["en"]["pronostico.ESTABLE"] != "Stable progress" {
		t.Fatal("opción, copia o referencia incorrecta", s1)
	}
	var referenciaAnterior paquete.ReferenciaCatalogo
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: actor.Perfiles}, func(ctx context.Context) error {
		var err error
		referenciaAnterior, err = (&refs.Referencias{}).Consultar(ctx, "es")
		if err != nil {
			return err
		}
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var valida bool
		err = tx.QueryRow(ctx, `SELECT i.id IS NOT NULL AND i.estado='verificada'
 FROM paquete.referencia_activa r JOIN paquete.idioma_instalado i ON i.id=r.idioma_id
 WHERE r.tipo='catalogo' AND r.codigo='es' AND i.version=$1 AND i.huella_sha256=$1`, referenciaAnterior.Contenido).Scan(&valida)
		if err == nil && !valida {
			t.Fatal("referencia de catálogo sin versión verificada")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	c.VersionEsperada = s1.Datos.Version.Numero
	c.ReferenciasEsperadas = s1.Referencias
	c.VigenteDesde = fecha(2031)
	c.Textos["es"] = "Evolución sostenida"
	s2, err := editor.Guardar(ctx, actor, "listas", c)
	if err != nil {
		t.Fatal(err)
	}
	textosAnteriores, err := disco.LeerCatalogo(ctx, referenciaAnterior)
	if err != nil || textosAnteriores["pronostico.ESTABLE"] != "Evolución estable" {
		t.Fatal("texto anterior perdido", err)
	}
	antiguo, err := app.NuevaConsulta(&repo.Repositorio{}, b).Datos(ctx, actor, "listas", fecha(2030))
	if err != nil || antiguo.Version.Numero != s1.Datos.Version.Numero || antiguo.Version.VigenteHasta == nil || !antiguo.Version.VigenteHasta.Equal(fecha(2031)) {
		t.Fatal("historia de vigencia", err)
	}
	// Una activación ya ejecutada en SQL se revierte si no se puede auditar.
	fallo := app.NuevaEdicion(&repo.Repositorio{}, publicacion, b, auditoriaFallida{}, time.Now)
	c.VersionEsperada = s2.Datos.Version.Numero
	c.ReferenciasEsperadas = s2.Referencias
	c.VigenteDesde = fecha(2032)
	if _, err := fallo.Guardar(ctx, actor, "listas", c); err == nil {
		t.Fatal("escritura sin auditoría")
	}
	trasFallo, err := editor.Consultar(ctx, actor, "listas")
	if err != nil || trasFallo.Datos.Version.Numero != s2.Datos.Version.Numero || trasFallo.Referencias["es"] != 2 {
		t.Fatal("escritura parcial", err)
	}
	eventosCatalogo := func() int {
		t.Helper()
		exportacion, err := a.Exportar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: actor.Perfiles}, 1, math.MaxInt64)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, evento := range exportacion.Eventos {
			if strings.HasPrefix(evento.Accion, "catalogo.") {
				n++
			}
		}
		return n
	}
	nEventos := eventosCatalogo()
	c.VersionEsperada = s1.Datos.Version.Numero
	if _, err := editor.Guardar(ctx, actor, "listas", c); err == nil {
		t.Fatal("versión antigua aceptada")
	}
	if eventosCatalogo() != nEventos {
		t.Fatal("auditoría de escritura revertida")
	}
	huerfano, err := disco.EscribirCatalogo(ctx, "es", map[string]string{"cargo.HUERFANO": "Cargo pendiente"})
	if err != nil {
		t.Fatal(err)
	}
	antigua := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(dir, huerfano.Ruta), antigua, antigua); err != nil {
		t.Fatal(err)
	}
	plazos, err := editor.Consultar(ctx, actor, "plazos")
	if err != nil {
		t.Fatal(err)
	}
	inicial := plazos.Datos.Version.Numero
	p := entrada.Cambio{VersionEsperada: inicial, VigenteDesde: fecha(2030), Plazo: &dominio.Plazo{Clave: "plazo_consentimiento", Valor: 30, Unidad: "dias_naturales"}}
	p1, err := editor.Guardar(ctx, actor, "plazos", p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, huerfano.Ruta)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("huérfano no limpiado al publicar", err)
	}
	if p1.Datos.Version.Numero != inicial+1 {
		t.Fatal(p1.Datos.Version)
	}
	p.VersionEsperada = inicial + 1
	p.VigenteDesde = fecha(2031)
	p.Plazo.Valor = 45
	if _, err := editor.Guardar(ctx, actor, "plazos", p); err != nil {
		t.Fatal(err)
	}
	plazoAnterior, err := app.NuevaConsulta(&repo.Repositorio{}, b).Datos(ctx, actor, "plazos", fecha(2030))
	if err != nil || plazoAnterior.Plazos[0].Valor != 30 {
		t.Fatal("plazo histórico", err)
	}
	// Dos instancias simulan procesos distintos: la exclusión debe estar en PostgreSQL.
	var wg sync.WaitGroup
	resultados := make(chan error, 2)
	inicio := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			otro := postgres.NuevoRegistroAuditoriaTransaccional(b)
			e := app.NuevaEdicion(&repo.Repositorio{}, publicacion, b, otro, time.Now)
			copia := p
			copia.VersionEsperada = inicial + 2
			copia.VigenteDesde = fecha(2032 + i)
			<-inicio
			_, err := e.Guardar(ctx, actor, "plazos", copia)
			resultados <- err
		}(i)
	}
	close(inicio)
	wg.Wait()
	close(resultados)
	ok, conflictos := 0, 0
	for err := range resultados {
		var c *dominio.Conflicto
		if err == nil {
			ok++
		} else if errors.As(err, &c) && c.Actual == inicial+3 {
			conflictos++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflictos != 1 {
		t.Fatal(ok, conflictos)
	}
	// Misma carrera para referencias de idioma, incluso sin catálogo.
	var rNueva paquete.ReferenciaCatalogo
	rNueva, err = disco.EscribirCatalogo(ctx, "es", s2.Textos["es"])
	if err != nil {
		t.Fatal(err)
	}
	resultados = make(chan error, 2)
	inicio = make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-inicio
			resultados <- b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: actor.Perfiles}, func(ctx context.Context) error { return (&refs.Referencias{}).Activar(ctx, rNueva, 2) })
		}()
	}
	close(inicio)
	wg.Wait()
	close(resultados)
	ok, conflictos = 0, 0
	for err := range resultados {
		var c *paquete.Conflicto
		if err == nil {
			ok++
		} else if errors.As(err, &c) && c.Actual == 3 {
			conflictos++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflictos != 1 {
		t.Fatal("carrera de referencias", ok, conflictos)
	}
	// Tramos: copia, límite abierto y rechazo de solapamiento.
	tramo := entrada.Cambio{VigenteDesde: fecha(2030), Tramo: &dominio.Tramo{Tipo: "edad_menor", Orden: 0, Desde: 0, Unidad: "anos"}}
	tr, err := editor.Guardar(ctx, actor, "tramos", tramo)
	if err != nil || len(tr.Datos.Tramos) != 1 {
		t.Fatal(err)
	}
	tramo.VersionEsperada = 1
	tramo.VigenteDesde = fecha(2031)
	tramo.Tramo = &dominio.Tramo{Tipo: "edad_menor", Orden: 1, Desde: 6, Unidad: "anos"}
	if _, err := editor.Guardar(ctx, actor, "tramos", tramo); !errors.Is(err, dominio.ErrEdicion) {
		t.Fatal("solapamiento permitido", err)
	}
	estado, err = editor.Consultar(ctx, actor, "firmantes")
	if err != nil {
		t.Fatal(err)
	}
	f := entrada.Cambio{VigenteDesde: fecha(2030), ReferenciasEsperadas: estado.Referencias, Firmante: &dominio.Firmante{Tipo: "firmante", Rol: "coordinacion_programa", Nombre: "Marta Serrano Ruiz", CargoCodigo: "COORDINACION", OrganismoCodigo: "DIPUTACION", Direccion: "Calle del Almez, 18"}, Textos: map[string]string{"es": "Coordinadora del Programa", "en": "Programme coordinator"}}
	f1, err := editor.Guardar(ctx, actor, "firmantes", f)
	if err != nil {
		t.Fatal(err)
	}
	f.Firmante.ID = f1.Firmantes[0].ID
	f.Firmante.Nombre = "Clara Molina Vega"
	f.Textos["es"] = "Coordinadora provincial"
	f.VersionEsperada = 1
	f.VigenteDesde = fecha(2031)
	f.ReferenciasEsperadas = f1.Referencias
	f2, err := editor.Guardar(ctx, actor, "firmantes", f)
	if err != nil || len(f2.Firmantes) != 2 || f2.Firmantes[1].VigenteHasta == nil {
		t.Fatal("firmante histórico", err)
	}
	if f2.Firmantes[1].CargoTextos["es"] != "Coordinadora del Programa" || f2.Firmantes[0].CargoTextos["es"] != "Coordinadora provincial" {
		t.Fatal("cargo histórico sustituido")
	}
	duplicado := f
	duplicado.Firmante = &dominio.Firmante{Tipo: "firmante", Rol: "coordinacion_programa", Nombre: "Nuria Medina Alba", CargoCodigo: "COORDINACION", OrganismoCodigo: "DIPUTACION"}
	duplicado.VersionEsperada = 0
	duplicado.VigenteDesde = fecha(2032)
	duplicado.ReferenciasEsperadas = f2.Referencias
	var conflictoRol *dominio.Conflicto
	if _, err := editor.Guardar(ctx, actor, "firmantes", duplicado); !errors.As(err, &conflictoRol) || conflictoRol.Actual != 2 {
		t.Fatal("rol duplicado sin 409", err)
	}
	f.VersionEsperada = 0
	f.VigenteDesde = fecha(2030)
	f.Firmante = &dominio.Firmante{Tipo: "destinatario", Rol: "servicio_menores", Nombre: "Álvaro Rueda Cano", CargoCodigo: "JEFATURA", OrganismoCodigo: "SPM", Direccion: "Avenida de la Vega, 27"}
	f.ReferenciasEsperadas = f2.Referencias
	f.Textos = map[string]string{"es": "Jefe de Servicio", "en": "Head of service"}
	hastaOffset := time.Date(2030, 1, 2, 0, 0, 0, 0, time.FixedZone("oriente", 14*60*60))
	f.Firmante.VigenteHasta = &hastaOffset
	destinatarios, err := editor.Guardar(ctx, actor, "firmantes", f)
	if err != nil {
		t.Fatal(err)
	}
	encontrado := false
	for _, d := range destinatarios.Firmantes {
		if d.Tipo == "destinatario" {
			encontrado = true
			if d.VigenteHasta == nil || !d.VigenteHasta.Equal(time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC)) {
				t.Fatal("fecha civil alterada", d.VigenteHasta)
			}
		}
	}
	if !encontrado {
		t.Fatal("destinatario ausente")
	}
	tx := migrador(t, ctx)
	rechaza(t, ctx, tx, `UPDATE catalogo.firmante SET nombre='Cambio indebido' WHERE id=$1`, "P0001", f1.Firmantes[0].ID)
	// Las funciones de escritura deniegan también con un contexto no administrador.
	for _, perfil := range []string{"equipo", "consulta", "coordinacion", "sistema", ""} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{perfil}}, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `SELECT catalogo.editar('plazos',3,DATE '2035-01-01','{}')`)
			return err
		})
		if err == nil {
			t.Fatal("función sin permiso", perfil)
		}
	}
	if roto, err := a.Verificar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: actor.Perfiles}, 1, math.MaxInt64); err != nil || roto != nil {
		t.Fatal("cadena inválida")
	}
	t.Log("EDICION-OK opciones, plazos, tramos, firmantes y destinatarios; historia y textos inmutables; auditoría fallida revierte datos y referencias")
	t.Log("CARRERA-OK PostgreSQL: catálogo y referencias, un éxito y un conflicto con versión actual")
}
