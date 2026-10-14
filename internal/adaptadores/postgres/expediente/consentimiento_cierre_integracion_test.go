package expediente_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestConsentimientoPermiteFinalizacionPropuestaYRetirada(t *testing.T) {
	b, ctx, deps, coordinacion, profesional, id := prepararHitos(t)
	defer b.Cerrar()
	deps.Propuestas = &repo.Repositorio{}
	deps.MotivosCierre = &catalogo.Repositorio{}
	s := aplicacion.Nuevo(deps)

	firma := dominio.ConfirmacionHito{Tipo: "consentimiento", Fecha: "2026-02-05", Version: 2}
	firmado, err := s.ConfirmarHito(ctx, profesional, id, "firma-cierre-"+id, firma)
	if err != nil || firmado.Version != 3 || firmado.Estado != dominio.EnIntervencion {
		t.Fatal(firmado, err)
	}
	if repetido, err := s.ConfirmarHito(ctx, profesional, id, "firma-cierre-"+id, firma); err != nil || repetido != firmado {
		t.Fatal("repetición de firma", repetido, err)
	}

	finalizacion := dominio.Finalizacion{Version: 3, Motivo: "OBJ_TOTAL", ObjetivosAbordados: "La familia mantiene los cuidados y la asistencia escolar de forma autónoma."}
	guardada, err := s.GuardarFinalizacion(ctx, profesional, id, "finalizacion-firmada-"+id, finalizacion)
	if err != nil || guardada.Version != 4 || guardada.Estado != "borrador" {
		t.Fatal(guardada, err)
	}
	if repetida, err := s.GuardarFinalizacion(ctx, profesional, id, "finalizacion-firmada-"+id, finalizacion); err != nil || repetida != guardada {
		t.Fatal("repetición de finalización", repetida, err)
	}

	propuesta, err := s.ProponerCierre(ctx, profesional, id, "propuesta-firmada-"+id, dominio.VersionCierre{Version: 4})
	if err != nil || propuesta.Version != 5 || propuesta.Estado != "pendiente" || propuesta.Motivo != "OBJ_TOTAL" {
		t.Fatal(propuesta, err)
	}
	if repetida, err := s.ProponerCierre(ctx, profesional, id, "propuesta-firmada-"+id, dominio.VersionCierre{Version: 4}); err != nil || repetida != propuesta {
		t.Fatal("repetición de propuesta", repetida, err)
	}

	retirada, err := s.RetirarPropuestaCierre(ctx, profesional, id, "retirada-firmada-"+id, dominio.VersionCierre{Version: 5})
	if err != nil || retirada.ID != propuesta.ID || retirada.Version != 6 || retirada.Estado != "retirada" {
		t.Fatal(retirada, err)
	}
	if repetida, err := s.RetirarPropuestaCierre(ctx, profesional, id, "retirada-firmada-"+id, dominio.VersionCierre{Version: 5}); err != nil || !mismoReciboCierre(t, repetida, retirada) {
		t.Fatal("repetición de retirada", repetida, err)
	}

	ficha, err := s.Consultar(ctx, profesional, id)
	if err != nil || ficha.Version != 6 || ficha.Estado != dominio.EnIntervencion || ficha.Aceptacion == nil || ficha.Aceptacion.FechaActa != "2026-02-03" {
		t.Fatal(ficha, err)
	}
	hitos, err := s.ConsultarHitos(ctx, profesional, id)
	if err != nil || len(hitos) != 2 {
		t.Fatal(hitos, err)
	}
	if hitos[1].Tipo != "consentimiento" || hitos[1].Fecha != "2026-02-05" || hitos[1].ConfirmadoPor != profesional.UsuarioID {
		t.Fatal("el circuito de cierre cambió el consentimiento", hitos[1])
	}
	comprobarHistoriaDecision(t, b, ctx, coordinacion, id, 2, 2, 1)

	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: profesional.UsuarioID, Perfiles: profesional.Perfiles, EquipoID: profesional.EquipoID}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var total, retiradas int
		if err := e.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE estado='retirada') FROM expediente.propuesta_cierre WHERE expediente_id=$1`, id).Scan(&total, &retiradas); err != nil {
			return err
		}
		if total != 1 || retiradas != 1 {
			t.Fatalf("propuestas=%d, retiradas=%d", total, retiradas)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	exportacion, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: coordinacion.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	acciones := map[string]int{
		"expediente.confirmar_hito":           0,
		"expediente.finalizacion":             0,
		"expediente.proponer_cierre":          0,
		"expediente.retirar_propuesta_cierre": 0,
	}
	for _, evento := range exportacion.Eventos {
		if evento.ExpedienteID == id {
			if _, corresponde := acciones[evento.Accion]; corresponde {
				acciones[evento.Accion]++
			}
		}
	}
	for accion, total := range acciones {
		if total != 1 {
			t.Fatalf("%s: eventos=%d", accion, total)
		}
	}
	t.Log("Firma, finalización, propuesta y retirada: versiones 3/4/5/6; repetición conserva recibos; estado en intervención; fecha de consentimiento intacta; un evento por escritura")
}

// El contrato de repetición compara el recibo JSON, no el puntero interno de la zona horaria.
func mismoReciboCierre(t *testing.T, a, b any) bool {
	t.Helper()
	ja, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	jb, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Equal(ja, jb)
}
