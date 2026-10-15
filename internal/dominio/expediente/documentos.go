package expediente

// DatosParaDocumento conserva la derivación, su decisión y la recepción
// leídas bajo el mismo bloqueo y dentro de la transacción de emisión.
type DatosParaDocumento struct {
	Derivacion
	// Anexo6 recoge la ficha vigente; nil conserva compatibilidad con la lectura inicial.
	Anexo6       *FichaParaAnexo6
	Finalizacion *FinalizacionGuardada
	ActaCierre   *ActaCierre
	// CopiaTraslado es la fotografía inmutable tomada antes del cierre de origen.
	CopiaTraslado *CopiaTraslado
	// FechaRecepcion es la fecha civil de Madrid, en formato AAAA-MM-DD.
	FechaRecepcion string
}

// FichaParaAnexo6 distingue los campos aún no recogidos de los vacíos confirmados.
type FichaParaAnexo6 struct {
	Familia *Familia
	Adultos *[]Adulto
	Menores []Menor
}
