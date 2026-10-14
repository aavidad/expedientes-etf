package expediente

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"testing"
	"time"
)

const municipioPrueba = "ab60479d-b778-407f-a83e-012345678910"
const centroPrueba = "ab60479d-b778-407f-a83e-012345678911"
const equipoPrueba = "ab60479d-b778-407f-a83e-012345678912"

type confirmacionesPrueba struct {
	*entorno
	huellas map[string]string
	recibos map[string]dominio.ReciboDerivacion
}

func (e *confirmacionesPrueba) RepetirConfirmacion(_ context.Context, usuario, clave, huella string, _ time.Time) (dominio.ReciboDerivacion, bool, error) {
	k := usuario + clave
	if anterior, ok := e.huellas[k]; ok {
		if anterior != huella {
			return dominio.ReciboDerivacion{}, false, dominio.ErrIdempotencia
		}
		return e.recibos[k], true, nil
	}
	return dominio.ReciboDerivacion{}, false, nil
}
func (e *confirmacionesPrueba) BloquearLectura(_ context.Context, id string) error {
	for _, d := range e.datos {
		if d.Lectura.ID == id {
			return dominio.ErrEstado
		}
	}
	return nil
}
func (e *confirmacionesPrueba) RecordarConfirmacion(_ context.Context, usuario, clave, huella string, r dominio.ReciboDerivacion, _ time.Time) error {
	e.huellas[usuario+clave] = huella
	e.recibos[usuario+clave] = r
	return nil
}
func prepararConfirmacion(t *testing.T) (*Servicio, *confirmacionesPrueba, entrada.Actor, entrada.Confirmacion) {
	s, e, a, b := preparar(t)
	c := &confirmacionesPrueba{entorno: e, huellas: map[string]string{}, recibos: map[string]dominio.ReciboDerivacion{}}
	s.d.Confirmaciones = c
	e.municipios = []catalogo.Municipio{{ID: municipioPrueba, INE: "18057", ClaveTexto: "municipio.18057", AmbitoDiputacion: true, CentroID: ptr(centroPrueba), EquipoID: ptr(equipoPrueba)}}
	s.d.Catalogo = c
	d, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "" || len(e.datos) != 0 {
		t.Fatal("la subida ha persistido datos")
	}
	return s, c, a, entrada.Confirmacion{LecturaID: d.Lectura.ID, Tipo: d.Datos.Extraccion.Tipo, Campos: d.Datos.Extraccion.Campos, Ubicacion: dominio.UbicacionRevisada{MunicipioID: municipioPrueba, CentroID: centroPrueba, EquipoID: equipoPrueba}}
}
func ptr(s string) *string { return &s }
func (e *confirmacionesPrueba) Datos(ctx context.Context, v catalogo.Version) (catalogo.Datos, error) {
	d, err := e.entorno.Datos(ctx, v)
	d.Centros = []catalogo.Centro{{ID: centroPrueba}}
	d.Equipos = []catalogo.Equipo{{ID: equipoPrueba}}
	return d, err
}
func TestConfirmacionCorrigeGuardaAuditaYRepiteSinDuplicar(t *testing.T) {
	s, e, a, p := prepararConfirmacion(t)
	p.Campos[0].Valor = "Centro de Servicios Sociales de Alhama"
	clave := "ab60479d-b778-407f-a83e-012345678999"
	r, err := s.ConfirmarDerivacion(context.Background(), a, clave, p)
	if err != nil {
		t.Fatal(err)
	}
	d := e.datos[r.ID]
	if d.Estado != dominio.DerivacionRecibida || d.Numero != nil || d.EquipoID != nil || d.Datos.UbicacionConfirmada == nil || d.Datos.Extraccion.Campos[0].Valor != p.Campos[0].Valor {
		t.Fatal(d)
	}
	if _, err := s.d.Borradores.Obtener(context.Background(), a.UsuarioID, p.LecturaID); !errors.Is(err, dominio.ErrNoEncontrada) {
		t.Fatal("borrador confirmado retenido", err)
	}
	r2, err := s.ConfirmarDerivacion(context.Background(), a, clave, p)
	if err != nil || r2 != r || len(e.datos) != 1 || len(e.eventos) != 2 {
		t.Fatal(r2, err, e.eventos)
	}
	p.Campos[0].Valor = "Otro centro"
	if _, err := s.ConfirmarDerivacion(context.Background(), a, clave, p); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal(err)
	}
	t.Log("Análisis sin escritura; corrección confirmada; auditoría atómica; repetición sin duplicados; borrador liberado")
}
func TestConfirmacionDeniegaOtroActorDatosInvalidosYAuditoriaFallida(t *testing.T) {
	s, e, a, p := prepararConfirmacion(t)
	otro := a
	otro.UsuarioID = "ab60479d-b778-407f-a83e-012345678998"
	if _, err := s.ConfirmarDerivacion(context.Background(), otro, p.LecturaID, p); !errors.Is(err, dominio.ErrNoEncontrada) {
		t.Fatal(err)
	}
	q := p
	q.Ubicacion.CentroID = "ab60479d-b778-407f-a83e-012345678997"
	if _, err := s.ConfirmarDerivacion(context.Background(), a, p.LecturaID, q); !errors.Is(err, dominio.ErrObligatorios) {
		t.Fatal(err)
	}
	q = p
	q.Campos = []dominio.Campo{{Codigo: "actor.usuario_id", Valor: a.UsuarioID}}
	if _, err := s.ConfirmarDerivacion(context.Background(), a, p.LecturaID, q); !errors.Is(err, dominio.ErrPeticion) {
		t.Fatal(err)
	}
	e.falloAuditoria = true
	if _, err := s.ConfirmarDerivacion(context.Background(), a, p.LecturaID, p); err == nil || len(e.datos) != 0 || len(e.recibos) != 0 {
		t.Fatal("escritura sin auditoría", err)
	}
	e.falloAuditoria = false
	if _, err := s.ConfirmarDerivacion(context.Background(), a, p.LecturaID, p); err != nil {
		t.Fatal(err)
	}
}
func TestDescartarImpideConfirmacion(t *testing.T) {
	s, e, a, p := prepararConfirmacion(t)
	if err := s.Descartar(context.Background(), a, p.LecturaID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmarDerivacion(context.Background(), a, p.LecturaID, p); !errors.Is(err, dominio.ErrNoEncontrada) || len(e.datos) != 0 {
		t.Fatal(err)
	}
}
