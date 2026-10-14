package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	estadistica "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
)

// WordCifras comparte plantilla y formato numérico, con entrada exclusivamente
// numérica. No tiene una fuente del informe completo ni accede a sus narrativas.
type WordCifras struct {
	Metadatos   estadistica.TextosConsulta
	Repositorio salida.RepositorioMemoria
	Idiomas     salida.Idiomas
	Ficheros    salida.FicherosPlantilla
	Textos      salida.TextosMemoria
	Motor       salida.GeneradorCifras
}

func (s WordCifras) GenerarCifras(ctx context.Context, cifras e.CifrasConsulta, idioma string) ([]byte, error) {
	if s.Repositorio == nil || s.Idiomas == nil || s.Ficheros == nil || s.Textos == nil || s.Motor == nil || s.Metadatos == nil {
		return nil, d.ErrNoDisponible
	}
	paquete, err := s.Idiomas.Resolver(ctx, idioma)
	if err != nil {
		return nil, err
	}
	plantilla, err := s.Repositorio.PlantillaMemoria(ctx, idioma)
	if err != nil {
		return nil, err
	}
	textos, err := s.Textos.LeerMemoria(ctx, paquete, plantilla)
	if err != nil {
		return nil, err
	}
	metadatos, err := s.Metadatos.LeerTextosCifras(ctx, paquete.Ruta)
	if err != nil {
		return nil, err
	}
	for k, v := range metadatos {
		textos.Campos[k] = v
	}
	campos, err := s.Motor.CamposCifras(ctx, cifras, textos)
	if err != nil {
		return nil, err
	}
	b, err := s.Ficheros.Leer(ctx, paquete, plantilla)
	if err != nil {
		return nil, err
	}
	defer clear(b)
	return s.Motor.Generar(ctx, b, d.Relleno{Campos: campos})
}
