package ficheros

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
)

type ExportacionesAuditoria struct{ plazas chan struct{} }

func NuevasExportacionesAuditoria() *ExportacionesAuditoria {
	return &ExportacionesAuditoria{make(chan struct{}, 2)}
}

type archivoAuditoria struct {
	*os.File
	carpeta string
	filas   int
	tamano  int64
	liberar func()
}

func (a *archivoAuditoria) Tamano() int64 { return a.tamano }
func (a *archivoAuditoria) Filas() int    { return a.filas }
func (a *archivoAuditoria) Close() error {
	err := a.File.Close()
	_ = os.RemoveAll(a.carpeta)
	if a.liberar != nil {
		a.liberar()
		a.liberar = nil
	}
	return err
}

type limiteAuditoria struct {
	w        io.Writer
	restante *int64
	ctx      context.Context
}

func (l limiteAuditoria) Write(p []byte) (int, error) {
	if err := l.ctx.Err(); err != nil {
		return 0, err
	}
	if int64(len(p)) > *l.restante {
		return 0, errors.New("auditoria.exportacion_demasiado_grande")
	}
	n, err := l.w.Write(p)
	*l.restante -= int64(n)
	return n, err
}

// La escritura canónica del registro filtrado permite comprobar también la
// selección contra la prueba contigua, sin cargar el conjunto de eventos.
//
//etf:tecnico
func escribirRegistro(w io.Writer, formato string, recorrer func(func(auditoria.Evento) error) error) error {
	if formato == "csv" {
		c := csv.NewWriter(w)
		if err := c.Write([]string{"id", "ocurrido_en", "usuario_id", "perfiles", "accion", "entidad", "entidad_id", "expediente_id", "resultado", "huella_anterior", "huella"}); err != nil {
			return err
		}
		err := recorrer(func(e auditoria.Evento) error {
			p, _ := json.Marshal(e.Perfiles)
			err := c.Write([]string{strconv.FormatUint(e.ID, 10), e.OcurridoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), e.UsuarioID, string(p), e.Accion, e.Entidad, e.EntidadID, e.ExpedienteID, e.Resultado, e.HuellaAnterior, e.Huella})
			c.Flush()
			if err != nil {
				return err
			}
			return c.Error()
		})
		c.Flush()
		if err != nil {
			return err
		}
		return c.Error()
	}
	if formato != "json" {
		return errors.New("auditoria.formato_invalido")
	}
	if _, err := io.WriteString(w, "["); err != nil {
		return err
	}
	primero := true
	err := recorrer(func(e auditoria.Evento) error {
		if !primero {
			if _, err := io.WriteString(w, ","); err != nil {
				return err
			}
		}
		primero = false
		return json.NewEncoder(w).Encode(e)
	})
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "]\n")
	return err
}

//etf:tecnico
func (p *ExportacionesAuditoria) Preparar(ctx context.Context, t auditoria.TramoExportacion, f auditoria.FiltroExportacion, formato, instrucciones string, recorrer func(func(auditoria.Evento) error) error) (resultado salida.ArchivoExportacion, fallo error) {
	select {
	case p.plazas <- struct{}{}:
	default:
		return nil, errors.New("auditoria.ocupada")
	}
	dir, err := os.MkdirTemp("", "etf-auditoria-")
	if err != nil {
		<-p.plazas
		return nil, errors.New("auditoria.no_disponible")
	}
	defer func() {
		if resultado == nil {
			_ = os.RemoveAll(dir)
			<-p.plazas
		}
	}()
	crear := func(nombre string) (*os.File, error) {
		return os.OpenFile(dir+"/"+nombre, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	}
	prueba, err := crear("cadena.json")
	if err != nil {
		return nil, errors.New("auditoria.no_disponible")
	}
	defer prueba.Close()
	registro, err := crear("registro." + formato)
	if err != nil {
		return nil, errors.New("auditoria.no_disponible")
	}
	defer registro.Close()
	presupuesto := int64(auditoria.MaxBytesExportacion)
	hp, hr := sha256.New(), sha256.New()
	wp := limiteAuditoria{io.MultiWriter(prueba, hp), &presupuesto, ctx}
	wr := limiteAuditoria{io.MultiWriter(registro, hr), &presupuesto, ctx}
	cabecera := auditoria.Exportacion{Version: auditoria.VersionExportacion, Algoritmo: "sha256", Canonizacion: "etf.auditoria.binaria.v1", Orden: "id_ascendente", Anterior: t.Anterior, Final: t.Final, Eventos: []auditoria.Evento{}}
	datos, _ := json.Marshal(cabecera)
	if _, err = wp.Write(datos[:len(datos)-2]); err != nil {
		return nil, err
	}
	filas, eventos := 0, 0
	err = escribirRegistro(wr, formato, func(seleccionar func(auditoria.Evento) error) error {
		return recorrer(func(e auditoria.Evento) error {
			if eventos > 0 {
				if _, err := io.WriteString(wp, ","); err != nil {
					return err
				}
			}
			if err := json.NewEncoder(wp).Encode(e); err != nil {
				return err
			}
			eventos++
			if f.Incluye(e) {
				filas++
				return seleccionar(e)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	if _, err = io.WriteString(wp, "]}\n"); err != nil {
		return nil, err
	}
	m := auditoria.ManifiestoExportacion{Version: "etf.auditoria.paquete.v1", Formato: formato, Filtro: f, Tramo: t, Filas: filas, EventosCadena: eventos, HuellaRegistro: hex.EncodeToString(hr.Sum(nil)), HuellaCadena: hex.EncodeToString(hp.Sum(nil))}
	paquete, err := crear("auditoria.zip")
	if err != nil {
		return nil, errors.New("auditoria.no_disponible")
	}
	correcto := false
	defer func() {
		if !correcto {
			_ = paquete.Close()
		}
	}()
	limiteZIP := int64(auditoria.MaxBytesExportacion)
	z := zip.NewWriter(limiteAuditoria{paquete, &limiteZIP, ctx})
	agregar := func(nombre string, r io.Reader) error {
		w, err := z.Create(nombre)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		return err
	}
	if _, err = registro.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if err = agregar("registro."+formato, registro); err != nil {
		return nil, err
	}
	if _, err = prueba.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if err = agregar("cadena.json", prueba); err != nil {
		return nil, err
	}
	w, err := z.Create("manifiesto.json")
	if err != nil {
		return nil, err
	}
	if err = json.NewEncoder(w).Encode(m); err != nil {
		return nil, err
	}
	w, err = z.Create("LEEME.txt")
	if err != nil {
		return nil, err
	}
	if len(instrucciones) > 32<<10 {
		return nil, errors.New("auditoria.formato_invalido")
	}
	if _, err = fmt.Fprintln(w, instrucciones); err != nil {
		return nil, err
	}
	if err = z.Close(); err != nil {
		return nil, err
	}
	tamano, err := paquete.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if _, err = paquete.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	correcto = true
	return &archivoAuditoria{File: paquete, carpeta: dir, filas: filas, tamano: tamano, liberar: func() { <-p.plazas }}, nil
}
