package estadistica

import "time"

// InformeMemoria conserva cifras y sus fuentes sin nombres ni contactos.
// Sólo Coordinación y el equipo autorizado pueden recibirlo. La representación
// en lista permite una huella canónica de las cifras y de su composición.
type CeldaMemoria struct {
	Grupo  Grupo
	Unidad Unidad
	Celda  Celda
}
type ApartadoMemoria struct {
	Codigo   Apartado
	Contrato Contrato
	Total    Recuento
	Celdas   []CeldaMemoria
	Avisos   map[string]int
}
type InformeMemoria struct {
	EquipoCodigo string
	FiltroMemoria
	Corte, ConocidoEn  time.Time
	Narrativas         Narrativas
	Apartados          []ApartadoMemoria
	VersionReglas      string
	Parametros         Parametros
	HuellaFuentes      string
	Composicion        []ComposicionMemoria
	HuellasComposicion map[Apartado]string
	Fuentes            []Expediente
	VersionDesglose    int
	Desglose           []AporteMemoria
}

type ComposicionMemoria struct {
	Apartado    Apartado
	Grupo       Grupo
	Unidad      Unidad
	Expedientes []string
}
