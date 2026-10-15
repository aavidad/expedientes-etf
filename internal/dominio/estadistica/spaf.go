package estadistica

import (
	"regexp"
	"slices"
	"sort"
	"time"
)

const (
	ErrPermisoSPAF    Error = "estadistica.spaf_acceso_denegado"
	ErrDisponibleSPAF Error = "estadistica.spaf_no_disponible"
)

type ActorSPAF struct {
	UsuarioID string
	Perfiles  []string
	EquipoID  string
}

func UUIDValidoSPAF(s string) bool {
	return regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`).MatchString(s)
}

type FiltroSPAF struct {
	Anio     int    `json:"anio"`
	Mes      int    `json:"mes"`
	EquipoID string `json:"equipo_id"`
}

// Autorizar no permite una tabla provincial: Coordinación elige un equipo;
// el profesional puede omitirlo para consultar el de su sesión.
func (a ActorSPAF) AutorizarSPAF(f FiltroSPAF) (FiltroSPAF, error) {
	if !UUIDValidoSPAF(a.UsuarioID) {
		return f, ErrPermisoSPAF
	}
	if !slices.Contains(a.Perfiles, "coordinacion") {
		if !slices.Contains(a.Perfiles, "profesional_etf") || !UUIDValidoSPAF(a.EquipoID) || (f.EquipoID != "" && f.EquipoID != a.EquipoID) {
			return f, ErrPermisoSPAF
		}
		f.EquipoID = a.EquipoID
	}
	if f.Anio < 1 || f.Anio > 9998 || f.Mes < 1 || f.Mes > 12 || !UUIDValidoSPAF(f.EquipoID) {
		return f, ErrEntrada
	}
	return f, nil
}

type MenorSPAF struct {
	ID               string          `json:"-"`
	Apellidos        string          `json:"apellidos"`
	Nombre           string          `json:"nombre"`
	Nacimiento       Dato[time.Time] `json:"nacimiento"`
	CentroProteccion Dato[string]    `json:"centro_proteccion"`
}

type GuardadorSPAF struct {
	Nombre     string `json:"nombre"`
	Parentesco string `json:"parentesco"`
}

type AnexoSPAF struct {
	Anexo int       `json:"anexo"`
	Fecha time.Time `json:"fecha"`
}

// DatosSPAF contiene los valores históricos de las columnas de la plantilla.
// Los campos categóricos son códigos, nunca etiquetas de catálogo.
type DatosSPAF struct {
	Menores                Dato[[]MenorSPAF]     `json:"menores"`
	Guardadores            Dato[[]GuardadorSPAF] `json:"guardadores"`
	Municipio              Dato[string]          `json:"municipio"`
	CaracteristicasFamilia Dato[[]string]        `json:"caracteristicas_familia"`
	Derivaciones           []AnexoSPAF           `json:"-"`
	ServicioDerivante      Dato[string]          `json:"servicio_derivante"`
	CoordinadorCaso        Dato[string]          `json:"coordinador_caso"`
	FechasAnexos           []AnexoSPAF           `json:"fechas_anexos"`
	FaseIntervencion       Dato[string]          `json:"fase_intervencion"`
	TemporalizacionMeses   Dato[int]             `json:"temporalizacion_meses"`
	Finalidad              Dato[string]          `json:"finalidad"`
	Pronostico             Dato[string]          `json:"pronostico"`
	Instrumentos           Dato[[]string]        `json:"instrumentos"`
}

// ExpedienteSPAF es una proyección distinta de la Memoria, con datos personales.
// El productor entrega episodios confirmados conocidos en el instante solicitado
// y conserva los expedientes de origen y destino de cada traslado por separado.
type ExpedienteSPAF struct {
	ID, EquipoID, Numero string
	Episodios            []Episodio
	Ficha                Historia[DatosSPAF]
}

// ReglasSPAF recibe las equivalencias versionadas de motivos del catálogo.
// No incluye porcentajes: la plantilla SPAF sólo pide cuatro recuentos.
type ReglasSPAF struct{ MotivosCierre map[Motivo]string }

type EntradaSPAF struct {
	Filtro                   FiltroSPAF
	FechaCalculo, ConocidoEn time.Time
	Reglas                   Historia[ReglasSPAF]
	Expedientes              []ExpedienteSPAF
}

type CierreSPAF struct {
	EpisodioID string       `json:"episodio_id"`
	Fecha      time.Time    `json:"fecha"`
	Motivo     Dato[string] `json:"motivo"`
}

type FilaSPAF struct {
	ExpedienteID string `json:"expediente_id"`
	Numero       string `json:"numero"`
	DatosSPAF
	NumeroMenores Dato[int]       `json:"numero_menores"`
	Derivacion    Dato[AnexoSPAF] `json:"derivacion"`
	Aceptacion    Dato[time.Time] `json:"aceptacion"`
	Cierres       []CierreSPAF    `json:"cierres"`
	Alta          bool            `json:"alta"`
	Baja          bool            `json:"baja"`
	VersionFicha  string          `json:"version_ficha"`
}

type ResumenSPAF struct {
	Altas               int `json:"altas"`
	Bajas               int `json:"bajas"`
	Casos               int `json:"casos"`
	Menores             int `json:"menores"`
	CasosSinDatoMenores int `json:"casos_sin_dato_menores"`
}

type TablaSPAF struct {
	FiltroSPAF
	Corte        time.Time      `json:"corte"`
	ConocidoEn   time.Time      `json:"conocido_en"`
	VersionRegla string         `json:"version_regla"`
	Filas        []FilaSPAF     `json:"filas"`
	Resumen      ResumenSPAF    `json:"resumen"`
	Avisos       map[string]int `json:"avisos"`
}

type EquipoSPAF struct {
	ID         string `json:"id"`
	ClaveTexto string `json:"clave_texto"`
}

type OpcionesSPAF struct {
	Equipos []EquipoSPAF                 `json:"equipos"`
	Textos  map[string]map[string]string `json:"textos"`
}

func finMesSPAF(f FiltroSPAF) time.Time {
	return time.Date(f.Anio, time.Month(f.Mes)+1, 0, 0, 0, 0, 0, time.UTC)
}

// CalcularSPAF cuenta expedientes, no episodios. Un cierre durante el mes
// sigue siendo baja aunque después se reabra; ambos hechos se conservan.
func CalcularSPAF(e EntradaSPAF) (TablaSPAF, error) {
	f := e.Filtro
	if f.Anio < 1 || f.Anio > 9998 || f.Mes < 1 || f.Mes > 12 || f.EquipoID == "" || e.FechaCalculo.IsZero() || e.ConocidoEn.IsZero() {
		return TablaSPAF{}, ErrEntrada
	}
	fin := finMesSPAF(f)
	if !fecha(e.FechaCalculo).After(fin) || e.ConocidoEn.After(e.FechaCalculo) {
		return TablaSPAF{}, ErrEntrada
	}
	inicio := time.Date(f.Anio, time.Month(f.Mes), 1, 0, 0, 0, 0, time.UTC)
	reglas, version, err := e.Reglas.ValorEn(fin, e.ConocidoEn)
	if err != nil {
		return TablaSPAF{}, err
	}
	if !reglas.Conocido {
		return TablaSPAF{}, ErrReglas
	}
	t := TablaSPAF{FiltroSPAF: f, Corte: fin, ConocidoEn: e.ConocidoEn, VersionRegla: version, Filas: []FilaSPAF{}, Avisos: map[string]int{}}
	ids := map[string]bool{}
	for _, exp := range e.Expedientes {
		if exp.ID == "" || exp.EquipoID != f.EquipoID || ids[exp.ID] {
			return TablaSPAF{}, ErrEntrada
		}
		ids[exp.ID] = true
		proyeccion := Expediente{Episodios: exp.Episodios}
		if err := validarExpediente(proyeccion, e.ConocidoEn); err != nil {
			return TablaSPAF{}, err
		}
		incluido := false
		for _, ep := range exp.Episodios {
			if !ep.Aceptacion.IsZero() && !fecha(ep.Aceptacion).After(fin) && !fecha(ep.Apertura).After(fin) && (ep.Cierre.IsZero() || !fecha(ep.Cierre).Before(inicio)) {
				incluido = true
			}
		}
		if !incluido {
			if len(exp.Episodios) > 0 && aceptacionInicial(proyeccion).IsZero() {
				t.Avisos["estadistica.aceptacion_sin_dato"]++
			}
			continue
		}
		datos, vf, err := exp.Ficha.ValorEn(fin, e.ConocidoEn)
		if err != nil {
			return TablaSPAF{}, err
		}
		fila := FilaSPAF{ExpedienteID: exp.ID, Numero: exp.Numero, VersionFicha: vf, Cierres: []CierreSPAF{}}
		if datos.Conocido {
			fila.DatosSPAF = datos.Valor
		} else {
			t.Avisos["estadistica.spaf_ficha_sin_dato"]++
		}
		if err := prepararDatosSPAF(&fila, fin, t.Avisos); err != nil {
			return TablaSPAF{}, err
		}
		aceptacion := aceptacionInicial(proyeccion)
		fila.Aceptacion = Dato[time.Time]{Valor: fecha(aceptacion), Conocido: !aceptacion.IsZero()}
		fila.Alta = !aceptacion.IsZero() && !fecha(aceptacion).Before(inicio) && !fecha(aceptacion).After(fin)
		for _, ep := range exp.Episodios {
			if ep.Cierre.IsZero() || fecha(ep.Cierre).Before(inicio) || fecha(ep.Cierre).After(fin) {
				continue
			}
			motivo := reglas.Valor.MotivosCierre[ep.Motivo]
			fila.Cierres = append(fila.Cierres, CierreSPAF{EpisodioID: ep.ID, Fecha: fecha(ep.Cierre), Motivo: Dato[string]{Valor: motivo, Conocido: motivo != ""}})
			if motivo == "" {
				t.Avisos["estadistica.spaf_motivo_sin_equivalencia"]++
			}
		}
		sort.Slice(fila.Cierres, func(i, j int) bool {
			if fila.Cierres[i].Fecha.Equal(fila.Cierres[j].Fecha) {
				return fila.Cierres[i].EpisodioID < fila.Cierres[j].EpisodioID
			}
			return fila.Cierres[i].Fecha.Before(fila.Cierres[j].Fecha)
		})
		fila.Baja = len(fila.Cierres) > 0
		t.Resumen.Casos++
		if fila.Alta {
			t.Resumen.Altas++
		}
		if fila.Baja {
			t.Resumen.Bajas++
		}
		if fila.NumeroMenores.Conocido {
			t.Resumen.Menores += fila.NumeroMenores.Valor
		} else {
			t.Resumen.CasosSinDatoMenores++
			t.Avisos["estadistica.spaf_menores_sin_dato"]++
		}
		t.Filas = append(t.Filas, fila)
	}
	sort.Slice(t.Filas, func(i, j int) bool {
		if t.Filas[i].Numero == t.Filas[j].Numero {
			return t.Filas[i].ExpedienteID < t.Filas[j].ExpedienteID
		}
		return t.Filas[i].Numero < t.Filas[j].Numero
	})
	return t, nil
}

func prepararDatosSPAF(f *FilaSPAF, fin time.Time, avisos map[string]int) error {
	// Nunca se devuelve el valor residual de un dato declarado desconocido.
	f.Menores = limpiarDatoSPAF(f.Menores)
	f.Guardadores = limpiarDatoSPAF(f.Guardadores)
	f.Municipio = limpiarDatoSPAF(f.Municipio)
	f.CaracteristicasFamilia = limpiarDatoSPAF(f.CaracteristicasFamilia)
	f.ServicioDerivante = limpiarDatoSPAF(f.ServicioDerivante)
	f.CoordinadorCaso = limpiarDatoSPAF(f.CoordinadorCaso)
	f.FaseIntervencion = limpiarDatoSPAF(f.FaseIntervencion)
	f.TemporalizacionMeses = limpiarDatoSPAF(f.TemporalizacionMeses)
	f.Finalidad = limpiarDatoSPAF(f.Finalidad)
	f.Pronostico = limpiarDatoSPAF(f.Pronostico)
	f.Instrumentos = limpiarDatoSPAF(f.Instrumentos)
	f.Menores.Valor = slices.Clone(f.Menores.Valor)
	for i := range f.Menores.Valor {
		f.Menores.Valor[i].Nacimiento = limpiarDatoSPAF(f.Menores.Valor[i].Nacimiento)
		f.Menores.Valor[i].CentroProteccion = limpiarDatoSPAF(f.Menores.Valor[i].CentroProteccion)
	}
	if f.TemporalizacionMeses.Conocido && f.TemporalizacionMeses.Valor < 0 {
		return ErrEntrada
	}
	vistos := map[string]bool{}
	for _, m := range f.Menores.Valor {
		if m.ID == "" || vistos[m.ID] || (m.Nacimiento.Conocido && (m.Nacimiento.Valor.IsZero() || fecha(m.Nacimiento.Valor).After(fin))) {
			return ErrEntrada
		}
		vistos[m.ID] = true
	}
	f.NumeroMenores = Dato[int]{Valor: len(f.Menores.Valor), Conocido: f.Menores.Conocido}
	var ultima AnexoSPAF
	ambigua := false
	for _, a := range f.Derivaciones {
		if a.Anexo < 3 || a.Anexo > 6 || a.Fecha.IsZero() {
			return ErrEntrada
		}
		a.Fecha = fecha(a.Fecha)
		if a.Fecha.After(fin) {
			continue
		}
		if ultima.Fecha.IsZero() || a.Fecha.After(ultima.Fecha) {
			ultima, ambigua = a, false
		} else if a.Fecha.Equal(ultima.Fecha) && a.Anexo != ultima.Anexo {
			ambigua = true
		}
	}
	f.Derivacion = Dato[AnexoSPAF]{Valor: ultima, Conocido: !ultima.Fecha.IsZero() && !ambigua}
	if ambigua {
		f.Derivacion.Valor = AnexoSPAF{}
		avisos["estadistica.spaf_derivacion_ambigua"]++
	}
	// Las fechas posteriores al mes no forman parte de la tabla, aunque el
	// productor las haya entregado en una revisión conocida más tarde.
	filtradas := []AnexoSPAF{}
	for _, a := range f.FechasAnexos {
		if (a.Anexo != 1 && a.Anexo != 2 && a.Anexo != 7 && a.Anexo != 8) || a.Fecha.IsZero() {
			return ErrEntrada
		}
		a.Fecha = fecha(a.Fecha)
		if !a.Fecha.After(fin) {
			filtradas = append(filtradas, a)
		}
	}
	f.FechasAnexos = filtradas
	return nil
}

func limpiarDatoSPAF[T any](d Dato[T]) Dato[T] {
	if !d.Conocido {
		return Dato[T]{}
	}
	return d
}
