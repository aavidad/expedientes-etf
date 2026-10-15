package estadistica

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	actividad "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	actividadSalida "github.com/aavidad/expedientes-etf/internal/puertos/salida/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"

	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"slices"
	"time"
)

type FuenteMemoria struct {
	Expedientes expediente.LecturaMemoria
	Reglas      salida.ReglasMemoria
	Actividad   actividadSalida.Repositorio
	Narrativas  salida.NarrativasHistoricas
	Equipos     catalogo.EquiposEstadistica
}

var _ entrada.FuenteMemoria = FuenteMemoria{}

func (s FuenteMemoria) MemoriaEnTransaccion(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, ahora time.Time) (d.InformeMemoria, error) {
	return s.memoria(ctx, a, f, ahora, ahora, nil)
}

func (s FuenteMemoria) MemoriaFijadaEnTransaccion(ctx context.Context, a d.ActorMemoria, base d.InformeMemoria, conocido time.Time) (d.InformeMemoria, error) {
	return s.memoria(ctx, a, base.FiltroMemoria, base.Corte, conocido, &base)
}
func (s FuenteMemoria) EquiposMemoria(ctx context.Context) ([]string, error) {
	equipos, ok := s.Equipos.(catalogo.EquiposMemoria)
	if !ok {
		return nil, d.ErrDisponibleMemoria
	}
	return equipos.EquiposMemoria(ctx)
}
func (s FuenteMemoria) memoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, fechaCalculo, ahora time.Time, fijada *d.InformeMemoria) (d.InformeMemoria, error) {
	f, err := d.AutorizarMemoria(a, f)
	if err != nil {
		return d.InformeMemoria{}, err
	}
	if s.Expedientes == nil || s.Reglas == nil || s.Narrativas == nil {
		return d.InformeMemoria{}, d.ErrDisponibleMemoria
	}
	cifras, e, parametros, versionReglas, err := s.calcularCifras(ctx, f, fechaCalculo, ahora, fijada)
	if err != nil {
		return d.InformeMemoria{}, err
	}
	n, err := s.Narrativas.NarrativasConocidas(ctx, f, ahora)
	if err != nil {
		return d.InformeMemoria{}, err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return d.InformeMemoria{}, err
	}
	h := sha256.Sum256(b)
	clear(b)
	r := d.InformeMemoria{VersionReglas: versionReglas, Parametros: parametros.Valor, HuellaFuentes: hex.EncodeToString(h[:]), FiltroMemoria: f, Corte: cifras.Corte, ConocidoEn: ahora, Narrativas: n, Apartados: []d.ApartadoMemoria{}}
	if f.EquipoID != "" {
		if s.Equipos == nil {
			return d.InformeMemoria{}, d.ErrDisponibleMemoria
		}
		codigo, err := s.Equipos.CodigoEquipo(ctx, f.EquipoID)
		if err != nil {
			return d.InformeMemoria{}, err
		}
		r.EquipoCodigo = codigo
	}

	for codigo, a := range cifras.Apartados {
		ap := d.ApartadoMemoria{Codigo: codigo, Contrato: a.Contrato, Total: a.Total, Avisos: a.Avisos, Celdas: []d.CeldaMemoria{}}
		for k, c := range a.Celdas {
			ap.Celdas = append(ap.Celdas, d.CeldaMemoria{Grupo: k.Grupo, Unidad: k.Unidad, Celda: c})
		}
		slices.SortFunc(ap.Celdas, func(a, b d.CeldaMemoria) int {
			ab, _ := json.Marshal(a)
			bb, _ := json.Marshal(b)
			return cmp.Compare(string(ab), string(bb))
		})
		r.Apartados = append(r.Apartados, ap)
	}
	slices.SortFunc(r.Apartados, func(a, b d.ApartadoMemoria) int { return cmp.Compare(a.Codigo, b.Codigo) })
	r.Desglose, err = d.AportesMemoria(e)
	if err != nil {
		return d.InformeMemoria{}, err
	}
	r.VersionDesglose = 1
	r.Fuentes = e.Expedientes
	r.Composicion = []d.ComposicionMemoria{}
	composicion := map[string]*d.ComposicionMemoria{}
	for _, aporte := range r.Desglose {
		for _, celda := range aporte.Celdas {
			clave := d.ClaveCelda{Grupo: celda.Grupo, Unidad: celda.Unidad}
			k, _ := json.Marshal(struct {
				Apartado d.Apartado
				Clave    d.ClaveCelda
			}{aporte.Apartado, clave})
			v := composicion[string(k)]
			if v == nil {
				v = &d.ComposicionMemoria{Apartado: aporte.Apartado, Grupo: clave.Grupo, Unidad: clave.Unidad}
				composicion[string(k)] = v
			}
			v.Expedientes = append(v.Expedientes, aporte.ExpedienteID)
		}
	}
	claves := []string{}
	for k := range composicion {
		claves = append(claves, k)
	}
	slices.Sort(claves)
	porApartado := map[d.Apartado][]d.ComposicionMemoria{}
	for _, k := range claves {
		v := *composicion[k]
		slices.Sort(v.Expedientes)
		r.Composicion = append(r.Composicion, v)
		porApartado[v.Apartado] = append(porApartado[v.Apartado], v)
	}
	r.HuellasComposicion = map[d.Apartado]string{}
	for codigo, lista := range porApartado {
		b, err := json.Marshal(lista)
		if err != nil {
			return d.InformeMemoria{}, err
		}
		h := sha256.Sum256(b)
		clear(b)
		r.HuellasComposicion[codigo] = hex.EncodeToString(h[:])
	}

	return r, nil
}

// calcularCifras es el único camino al motor para la Memoria completa y Consulta.
func (s FuenteMemoria) calcularCifras(ctx context.Context, f d.FiltroMemoria, fechaCalculo, ahora time.Time, fijada *d.InformeMemoria) (d.Memoria, d.Entrada, d.Dato[d.Parametros], string, error) {
	var err error
	corte := time.Date(f.Anio, 12, 31, 0, 0, 0, 0, time.UTC)
	if fechaCalculo.Before(corte) {
		corte = fechaCalculo
	}
	var reglas d.Historia[d.Parametros]
	if fijada != nil {
		reglas = d.Historia[d.Parametros]{{ID: fijada.VersionReglas, Vigencia: d.Vigencia{Desde: time.Date(f.Anio, 1, 1, 0, 0, 0, 0, time.UTC)}, RegistradoEn: time.Time{}.Add(time.Microsecond), Dato: d.Dato[d.Parametros]{Conocido: true, Valor: fijada.Parametros}}}
	} else {
		reglas, err = s.Reglas.ReglasMemoria(ctx, f.Anio, ahora)
		if err != nil {
			return d.Memoria{}, d.Entrada{}, d.Dato[d.Parametros]{}, "", err
		}
	}
	expedientes, err := s.Expedientes.ConsultarMemoria(ctx, f.EquipoID, corte, ahora)
	if err != nil {
		return d.Memoria{}, d.Entrada{}, d.Dato[d.Parametros]{}, "", err
	}
	for _, exp := range expedientes {
		if f.EquipoID != "" && exp.EquipoID != f.EquipoID {
			return d.Memoria{}, d.Entrada{}, d.Dato[d.Parametros]{}, "", d.ErrPermisoMemoria
		}
	}
	e := d.Entrada{Anio: f.Anio, FechaCalculo: fechaCalculo, ConocidoEn: ahora, Reglas: reglas, Expedientes: expedientes}
	if s.Actividad != nil {
		r, err := s.Actividad.Recontar(ctx, actividad.Filtro{Anio: f.Anio, EquipoID: f.EquipoID})
		if err != nil {
			return d.Memoria{}, d.Entrada{}, d.Dato[d.Parametros]{}, "", err
		}
		e.Actividad = &d.RecuentoActividad{Anio: r.Anio, Comisiones: r.Comisiones, Informes: r.Informes, PorOrganismo: r.PorOrganismo}
	}
	cifras, err := d.Calcular(e)
	if err != nil {
		return d.Memoria{}, d.Entrada{}, d.Dato[d.Parametros]{}, "", err
	}

	cifras.Apartados[d.NoAceptados].Avisos["estadistica.no_aceptados_sin_equipo"]++
	// Una relación parental desconocida no equivale a cero personas en las
	// otras dimensiones; el motor sólo crea su celda de ausencia en el 19.
	for k, c := range cifras.Apartados[d.EdadParental].Celdas {
		if k.Grupo.Papel == "" && k.Grupo.Sexo == "" && c.SinDato > 0 {
			for _, codigo := range []d.Apartado{d.NivelEducativoParental, d.SituacionLaboralParental} {
				cifras.Apartados[codigo].Avisos["estadistica.poblacion_parental_sin_dato"] += c.SinDato
			}
		}
	}

	parametros, versionReglas, err := reglas.ValorEn(time.Date(f.Anio, 12, 31, 0, 0, 0, 0, time.UTC), ahora)
	if err != nil {
		return d.Memoria{}, d.Entrada{}, d.Dato[d.Parametros]{}, "", err
	}
	for _, dimension := range []struct {
		codigo   d.Apartado
		opciones []string
	}{{d.CoordinacionRecursos, parametros.Valor.RecursosCoordinacion}, {d.FactoresFamilia, parametros.Valor.FactoresFamilia}, {d.FactoresMenores, parametros.Valor.FactoresMenores}, {d.TipologiasDesproteccion, parametros.Valor.TipologiasDesproteccion}, {d.ResponsablesDesproteccion, parametros.Valor.ResponsablesDesproteccion}} {
		if len(dimension.opciones) == 0 {
			cifras.Apartados[dimension.codigo].Avisos["estadistica.catalogo_dimension_sin_dato"]++
		}
	}
	return cifras, e, parametros, versionReglas, nil
}

// CalcularCifrasEnTransaccion no carga narrativas ni construye composición.
// Su adaptador ejecuta esta lectura con la credencial técnica de cálculo.
func (s FuenteMemoria) CalcularCifrasEnTransaccion(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, ahora time.Time) (d.CifrasConsulta, error) {
	if err := d.AutorizarConsultaMemoria(a, f); err != nil {
		return d.CifrasConsulta{}, err
	}
	if s.Expedientes == nil || s.Reglas == nil {
		return d.CifrasConsulta{}, d.ErrDisponibleMemoria
	}
	cifras, _, _, _, err := s.calcularCifras(ctx, f, ahora, ahora, nil)
	if err != nil {
		return d.CifrasConsulta{}, err
	}
	r := d.ProyectarCifras(cifras, f.EquipoID)
	if f.EquipoID != "" {
		if s.Equipos == nil {
			return d.CifrasConsulta{}, d.ErrDisponibleMemoria
		}
		r.EquipoCodigo, err = s.Equipos.CodigoEquipo(ctx, f.EquipoID)
		if err != nil {
			return d.CifrasConsulta{}, err
		}
	}
	return r, nil
}
