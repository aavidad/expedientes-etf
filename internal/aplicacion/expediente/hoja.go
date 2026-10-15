package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"regexp"
	"strings"
	"unicode"
)

// ServicioHoja convierte la hoja en el contrato de F2.08a. No almacena una
// vista previa: la confirmación recibe de nuevo los mismos bytes y los revisa.
type ServicioHoja struct {
	Historicos entrada.CargasHistoricas
	Lector     salida.LectorHoja
	Antivirus  plataforma.Antivirus
	Reloj      salida.Reloj
}

var _ entrada.HojasHistoricas = (*ServicioHoja)(nil)

func (s *ServicioHoja) RechazarCargaHistorica(ctx context.Context, a d.Actor) error {
	if s.Historicos == nil {
		return d.ErrNoDisponible
	}
	return s.Historicos.RechazarCargaHistorica(ctx, a)
}

func (s *ServicioHoja) leer(ctx context.Context, a d.Actor, u entrada.SubidaHoja) (p d.CargaHistorica, filas []d.FilaHoja, informe d.InformeHoja, fallo error) {
	informe.Incidencias = []d.IncidenciaHoja{}
	defer func() {
		if fallo != nil && s.RechazarCargaHistorica(ctx, a) != nil {
			fallo = d.ErrNoDisponible
		}
	}()
	if err := autorizarCarga(a, u.EquipoID); err != nil {
		return p, nil, informe, err
	}
	if s.Historicos == nil || s.Lector == nil || s.Antivirus == nil || s.Reloj == nil {
		return p, nil, informe, d.ErrNoDisponible
	}
	if len(u.Contenido) == 0 || len(u.Contenido) > d.MaxBytes {
		return p, nil, informe, d.ErrHojaLimites
	}
	if u.Formato != "csv" && u.Formato != "xlsx" {
		return p, nil, informe, d.ErrHojaFormato
	}
	estado, err := s.Antivirus.Revisar(ctx, u.Contenido)
	if estado == plataforma.AntivirusInfectado {
		return p, nil, informe, d.ErrInfectado
	}
	if err != nil || estado != plataforma.AntivirusLimpio {
		return p, nil, informe, d.ErrHojaAntivirus
	}
	filas, err = s.Lector.LeerHoja(ctx, u.Formato, u.Contenido)
	if err != nil {
		return p, nil, informe, err
	}
	h := sha256.Sum256(u.Contenido)
	informe.Huella = hex.EncodeToString(h[:])
	informe.Casos = len(filas)
	informe.Filas = []d.ResumenFilaHoja{}
	for _, fila := range filas {
		resumen := d.ResumenFilaHoja{Fila: fila.Numero}
		var iniciales strings.Builder
		if regexp.MustCompile(`^[0-9]{2}-[0-9]{2}-[0-9]{2}$`).MatchString(fila.Caso.Numero) {
			resumen.Numero = fila.Caso.Numero
		}
		switch fila.Caso.Estado {
		case d.Aceptada, d.EnIntervencion, d.Cerrada, d.NoAceptada:
			resumen.Estado = fila.Caso.Estado
		}
		for _, campo := range fila.Caso.Datos.Extraccion.Campos {
			if campo.Codigo == "menores.persona.nombre" {
				for _, parte := range strings.Fields(campo.Valor) {
					for _, letra := range parte {
						if unicode.IsLetter(letra) {
							iniciales.WriteRune(unicode.ToUpper(letra))
							iniciales.WriteRune('.')
							iniciales.WriteRune(' ')
							break
						}
					}
				}
				break
			}
		}
		resumen.Iniciales = strings.TrimSpace(iniciales.String())
		informe.Filas = append(informe.Filas, resumen)
	}
	p.EquipoID = u.EquipoID
	p.ConfirmarDuplicados = u.ConfirmarDuplicados
	p.FuenteHoja = &d.FuenteHoja{Formato: u.Formato, Huella: informe.Huella, Bytes: len(u.Contenido)}
	if len(filas) < 1 || len(filas) > 100 {
		return p, filas, informe, d.ErrHojaLimites
	}
	numeros := map[string]int{}
	for _, f := range filas {
		if f.Caso.Aceptacion != nil {
			f.Caso.Aceptacion.EquipoID = u.EquipoID
		}
		p.Casos = append(p.Casos, f.Caso)
		informe.Incidencias = append(informe.Incidencias, f.Incidencias...)
		if len(f.Incidencias) != 0 {
			continue
		}
		if err := (d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{f.Caso}}).Validar(s.Reloj.Ahora()); err != nil {
			informe.Incidencias = append(informe.Incidencias, incidencia(f.Numero, err))
		}
		if f.Caso.Numero != "" {
			if previa, existe := numeros[f.Caso.Numero]; existe {
				informe.Incidencias = append(informe.Incidencias, incidencia(previa, d.ErrNumeroDuplicado), incidencia(f.Numero, d.ErrNumeroDuplicado))
			} else {
				numeros[f.Caso.Numero] = f.Numero
			}
		}
	}
	// Identificador estable de usuario, ámbito y bytes. La autorización y toda
	// la validación se repiten al confirmar; este UUID no concede acceso.
	l := sha256.Sum256([]byte(a.UsuarioID + ":" + u.EquipoID + ":" + u.Formato + ":" + informe.Huella))
	l[6] = l[6]&15 | 80
	l[8] = l[8]&63 | 128
	x := hex.EncodeToString(l[:16])
	informe.LoteID = x[:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:]
	return p, filas, informe, nil
}

func incidencia(fila int, err error) d.IncidenciaHoja {
	clave, columna := "expediente.carga_historica_invalida", ""
	for _, c := range []struct {
		err     error
		columna string
	}{
		{d.ErrNumeroDuplicado, "numero"}, {d.ErrNumeroHistorico, "numero"}, {d.ErrFamilia, "familia"},
		{d.ErrMenores, "menores"}, {d.ErrProfesional, "profesional_referencia_puesto"}, {d.ErrSexo, "sexos_menores"},
		{d.ErrObligatorios, ""},
	} {
		if errors.Is(err, c.err) {
			clave, columna = c.err.Error(), c.columna
			break
		}
	}
	return d.IncidenciaHoja{Fila: fila, Columna: columna, Clave: clave}
}

func (s *ServicioHoja) ValidarHoja(ctx context.Context, a d.Actor, u entrada.SubidaHoja) (d.InformeHoja, error) {
	p, filas, informe, err := s.leer(ctx, a, u)
	if err != nil {
		return d.InformeHoja{}, err
	}
	if len(informe.Incidencias) != 0 {
		if err := s.RechazarCargaHistorica(ctx, a); err != nil {
			return d.InformeHoja{}, err
		}
		informe.LoteID = ""
		return informe, nil
	}
	vista, err := s.Historicos.ValidarCargaHistorica(ctx, a, p)
	informe.Avisos = vista.Avisos
	if err != nil {
		if errors.Is(err, d.ErrNoDisponible) || errors.Is(err, d.ErrAcceso) || ctx.Err() != nil {
			return d.InformeHoja{}, err
		}
		// Sólo en un lote rechazado se ensaya cada fila por separado para que el
		// informe incluya también los errores de catálogo y vigencia posteriores.
		for _, f := range filas {
			individual := d.CargaHistorica{EquipoID: p.EquipoID, FuenteHoja: p.FuenteHoja, Casos: []d.CasoHistorico{f.Caso}}
			if _, fallo := s.Historicos.ValidarCargaHistorica(ctx, a, individual); fallo != nil {
				if errors.Is(fallo, d.ErrNoDisponible) || errors.Is(fallo, d.ErrAcceso) || ctx.Err() != nil {
					return d.InformeHoja{}, fallo
				}
				informe.Incidencias = append(informe.Incidencias, incidencia(f.Numero, fallo))
			}
		}
		if len(informe.Incidencias) == 0 {
			var fila d.ErrorFilaHistorica
			if errors.As(err, &fila) && fila.Indice >= 0 && fila.Indice < len(filas) {
				informe.Incidencias = append(informe.Incidencias, incidencia(filas[fila.Indice].Numero, err))
			} else {
				informe.Incidencias = append(informe.Incidencias, incidencia(0, err))
			}
		}
		informe.LoteID = ""
		return informe, nil
	}
	informe.Valida = true
	return informe, nil
}

func (s *ServicioHoja) ConfirmarHoja(ctx context.Context, a d.Actor, lote, clave, huella string, u entrada.SubidaHoja) (d.ResultadoCargaHistorica, d.InformeHoja, error) {
	if !d.ClaveValida(clave) {
		return d.ResultadoCargaHistorica{}, d.InformeHoja{}, s.rechazar(ctx, a, d.ErrClave)
	}
	p, filas, informe, err := s.leer(ctx, a, u)
	if err != nil {
		return d.ResultadoCargaHistorica{}, d.InformeHoja{}, err
	}
	if informe.LoteID != lote || informe.Huella != huella {
		return d.ResultadoCargaHistorica{}, d.InformeHoja{}, s.rechazar(ctx, a, d.ErrHojaCambiada)
	}
	if len(informe.Incidencias) != 0 {
		if err := s.RechazarCargaHistorica(ctx, a); err != nil {
			return d.ResultadoCargaHistorica{}, d.InformeHoja{}, err
		}
		informe.LoteID = ""
		return d.ResultadoCargaHistorica{}, informe, nil
	}
	r, err := s.Historicos.CargarHistoricos(ctx, a, lote, clave, p)
	if err != nil {
		var fila d.ErrorFilaHistorica
		if errors.As(err, &fila) && !errors.Is(err, d.ErrNoDisponible) && !errors.Is(err, d.ErrAcceso) && fila.Indice >= 0 && fila.Indice < len(filas) {
			informe.Incidencias = append(informe.Incidencias, incidencia(filas[fila.Indice].Numero, err))
			informe.LoteID = ""
			return d.ResultadoCargaHistorica{}, informe, nil
		}
		return d.ResultadoCargaHistorica{}, d.InformeHoja{}, err
	}
	informe.Valida = true
	return r, informe, nil
}

func (s *ServicioHoja) rechazar(ctx context.Context, a d.Actor, err error) error {
	if fallo := s.RechazarCargaHistorica(ctx, a); fallo != nil {
		return fallo
	}
	return err
}
