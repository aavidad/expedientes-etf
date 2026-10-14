package estadistica

// MotivoDesdeCodigo proyecta los códigos de cierre publicados a las categorías
// estadísticas. Un código no reconocido permanece desconocido.
func MotivoDesdeCodigo(c string) Motivo {
	switch c {
	case "OBJ_TOTAL":
		return ObjetivosCumplidos
	case "OBJ_PARCIAL":
		return ObjetivosParciales
	case "ABANDONO", "SIN_CONSENTIMIENTO":
		return Rechazo
	case "SIN_ADHERENCIA":
		return NoAdherencia
	case "TRASLADO":
		return Traslado
	case "MAYORIA_EDAD":
		return MayoriaEdad
	case "DECLARACION_RIESGO":
		return PropuestaDSR
	case "MEDIDA_PROTECTORA":
		return MedidaProtectora
	case "INGRESO_CENTRO":
		return IngresoCentro
	case "OTROS":
		return OtrasCausas
	}
	return ""
}
