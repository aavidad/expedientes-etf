package estadistica

import "time"

const ErrCifrasNoPublicadasMemoria Error = "estadistica.memoria_cifras_no_publicadas"

// CifrasMemoria es la proyección agregada: no admite narrativas ni composición.
type CifrasMemoria struct {
	Anio         int                     `json:"anio"`
	EquipoID     string                  `json:"equipo_id"`
	EquipoCodigo string                  `json:"equipo_codigo"`
	Corte        time.Time               `json:"corte"`
	ConocidoEn   time.Time               `json:"conocido_en"`
	Apartados    []CifrasApartadoMemoria `json:"apartados"`
}

type CifrasApartadoMemoria struct {
	Codigo   Apartado              `json:"codigo"`
	Contrato ContratoCifrasMemoria `json:"contrato"`
	Total    TotalCifrasMemoria    `json:"total"`
	Celdas   []CeldaCifrasMemoria  `json:"celdas"`
	Avisos   map[string]int        `json:"avisos"`
}

type ContratoCifrasMemoria struct {
	Poblacion    string    `json:"poblacion"`
	Denominador  string    `json:"denominador"`
	Desconocidos string    `json:"desconocidos"`
	FechaDatos   string    `json:"fecha_datos"`
	Corte        time.Time `json:"corte"`
	FechaEdad    time.Time `json:"fecha_edad"`
	VersionRegla string    `json:"version_regla"`
	DivisionCero string    `json:"division_cero"`
}

type TotalCifrasMemoria struct {
	Familias                   int `json:"familias"`
	Menores                    int `json:"menores"`
	Adultos                    int `json:"adultos"`
	Comisiones                 int `json:"comisiones"`
	Informes                   int `json:"informes"`
	FamiliasSinRelacionMenores int `json:"familias_sin_relacion_menores"`
}

type GrupoCifrasMemoria struct {
	Subprograma Subprograma `json:"subprograma"`
	Origen      string      `json:"origen"`
	Sexo        Sexo        `json:"sexo"`
	Tramo       string      `json:"tramo"`
	Motivo      Motivo      `json:"motivo"`
	Papel       string      `json:"papel"`
	Estado      string      `json:"estado"`
	Causa       string      `json:"causa"`
	Organismo   string      `json:"organismo"`
}

type DenominadorCifrasMemoria struct {
	Conocido bool `json:"conocido"`
	Valor    int  `json:"valor"`
}

type PorcentajeCifrasMemoria struct {
	Centesimas int    `json:"centesimas"`
	Aviso      string `json:"aviso"`
}

type CeldaCifrasMemoria struct {
	Grupo        GrupoCifrasMemoria       `json:"grupo"`
	Unidad       Unidad                   `json:"unidad"`
	Numerador    int                      `json:"numerador"`
	Denominador  DenominadorCifrasMemoria `json:"denominador"`
	Porcentaje   *PorcentajeCifrasMemoria `json:"porcentaje"`
	SinDato      int                      `json:"sin_dato"`
	Poblacion    string                   `json:"poblacion"`
	VersionRegla string                   `json:"version_regla"`
}
