export function etiquetaCifra(t: (clave: string) => string, apartado: string, dimension: string, valor: string) {
    const contextos: Record<string, string> = { '3': 'antecedentes', '4': 'servicio_derivante', '5': 'servicio_derivante', '10': 'derivacion_sscc', '11.1': 'propuesta_dsr', '11.2': 'pif_derivado_dsr', '12.1': 'medida_protectora', '12.2': 'respuesta_spm', '14': 'coordinacion', '14.1': 'atencion_temprana', '15': 'comision', '17': 'composicion_familiar', '18': 'familia_numerosa', '20': 'nivel_educativo_parental', '21': 'situacion_laboral_parental', '22': 'situacion_economica', '24': 'factor_familia', '26': 'factor_menor', '27': 'tipologia_desproteccion', '28': 'responsable_desproteccion' };
    const tramos: Record<string, string> = { '13.1': 'tramo_duracion_riesgo', '13.2': 'tramo_duracion_reunificacion', '19': 'tramo_edad_parental', '23': 'tramo_ingresos_smi', '25': 'tramo_edad_menor' };
    const dimensiones: Record<string, string> = { papel: 'papel_parental', motivo: 'motivo_cierre', causa: 'causa_no_aceptacion' };
    const prefijo = dimension === 'sexo' && ['19', '20', '21'].includes(apartado) ? 'sexo_parental' : dimension === 'estado' ? contextos[apartado] : dimension === 'tramo' ? tramos[apartado] : dimensiones[dimension] ?? dimension;
    const codigo = valor.toLowerCase();
    return t(`memoria.valor.${prefijo}.${codigo}`) || t(`memoria.valor.${codigo}`) || t(codigo) || t('memoria.etiqueta_pendiente');
  }
