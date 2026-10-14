package expediente

// DatosParaDocumento conserva la derivación, su decisión y la recepción
// leídas bajo el mismo bloqueo y dentro de la transacción de emisión.
type DatosParaDocumento struct {
	Derivacion
	Finalizacion *FinalizacionGuardada
	ActaCierre   *ActaCierre
	// CopiaTraslado es la fotografía inmutable tomada antes del cierre de origen.
	CopiaTraslado *CopiaTraslado
	// FechaRecepcion es la fecha civil de Madrid, en formato AAAA-MM-DD.
	FechaRecepcion string
}
