-- RLS y reintentos reales sobre el PostgreSQL desechable restaurado.
BEGIN;
DO $$ DECLARE r uuid; v bigint; actor uuid; referencia jsonb; fecha timestamptz; nuevo uuid; propia uuid; datos_propios jsonb; BEGIN
 IF EXISTS(SELECT FROM identidad.sesion) THEN RAISE EXCEPTION 'seg4.sesion_restaurada'; END IF;
 SELECT id,version,copia_en,revision_restauracion.referencia INTO r,v,fecha,referencia FROM identidad.revision_restauracion WHERE confirmada_en IS NULL AND anulada_en IS NULL;
 SELECT id INTO actor FROM identidad.usuario WHERE cuenta_ad='elena.salas';
 PERFORM set_config('app.usuario_id',actor::text,true);PERFORM set_config('app.perfiles','administracion',true);
 BEGIN
  PERFORM identidad.confirmar_revision_restauracion(r,v);
  RAISE EXCEPTION 'seg4.confirmacion_con_pendientes';
 EXCEPTION WHEN raise_exception THEN
  IF SQLERRM<>'restauracion.cambios_pendientes' THEN RAISE; END IF;
 END;
 BEGIN
  PERFORM identidad.comprobar_revision_restauracion((SELECT id FROM identidad.usuario WHERE cuenta_ad='javier.medina'));
  RAISE EXCEPTION 'seg4.usuario_restaurado_admitido';
 EXCEPTION WHEN raise_exception THEN
  IF SQLERRM<>'restauracion.acceso_bloqueado' THEN RAISE; END IF;
 END;
 -- Una propuesta propia nunca se anuncia como aplicada tras la regla SEG2.
 SELECT x INTO datos_propios FROM jsonb_array_elements(referencia) x
 WHERE x->>'cuenta'='elena.salas';
 INSERT INTO identidad.cambio_restauracion(revision_id,guid,anterior,propuesto)
 VALUES(r,(datos_propios->>'guid')::uuid,datos_propios,
 jsonb_set(datos_propios,'{perfiles}',(datos_propios->'perfiles') || jsonb_build_array(jsonb_build_object('perfil','coordinacion','desde',clock_timestamp(),'hasta',NULL)))) RETURNING id INTO propia;
 BEGIN
  PERFORM identidad.resolver_cambio_restauracion(r,propia,v,'aplicar');
  RAISE EXCEPTION 'seg4.cambio_propio_aplicado';
 EXCEPTION WHEN raise_exception THEN
  IF SQLERRM<>'restauracion.revision_propia' THEN RAISE; END IF;
 END;
 IF EXISTS(SELECT FROM identidad.cambio_restauracion WHERE id=propia AND decision IS NOT NULL)
 OR EXISTS(SELECT FROM auditoria.evento WHERE entidad_id=propia AND accion='identidad.restauracion_aplicar') THEN
  RAISE EXCEPTION 'seg4.cambio_propio_confirmado';
 END IF;
 DELETE FROM identidad.cambio_restauracion WHERE id=propia;
 nuevo:=identidad.iniciar_revision_restauracion(fecha,referencia);
 IF nuevo=r OR EXISTS(SELECT FROM identidad.revision_restauracion WHERE id=r AND anulada_en IS NULL) THEN RAISE EXCEPTION 'seg4.revision_reutilizada'; END IF;
 PERFORM set_config('app.usuario_id',actor::text,true);PERFORM set_config('app.perfiles','administracion',true);
 BEGIN
  PERFORM identidad.confirmar_revision_restauracion(r,v);
  RAISE EXCEPTION 'seg4.confirmacion_antigua';
 EXCEPTION WHEN raise_exception THEN
  IF SQLERRM<>'identidad.conflicto' THEN RAISE; END IF;
 END;
END $$;
SET ROLE etf_app;
DO $$ BEGIN
 IF (SELECT count(*) FROM identidad.revision_restauracion WHERE confirmada_en IS NULL AND anulada_en IS NULL)<>1 THEN RAISE EXCEPTION 'seg4.rls_admin'; END IF;
 BEGIN
  UPDATE identidad.revision_restauracion SET confirmada_en=clock_timestamp();
  RAISE EXCEPTION 'seg4.mutacion_directa';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM identidad.iniciar_revision_restauracion(clock_timestamp(),'[]');
  RAISE EXCEPTION 'seg4.inicio_desde_app';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $$;
SELECT set_config('app.usuario_id','',true),set_config('app.perfiles','',true);
DO $$ BEGIN
 IF EXISTS(SELECT FROM identidad.cambio_restauracion) THEN RAISE EXCEPTION 'seg4.rls_anonimo'; END IF;
END $$;
RESET ROLE;
COMMIT;

EXPLAIN SELECT id FROM identidad.cambio_restauracion
 WHERE revision_id=(SELECT id FROM identidad.revision_restauracion WHERE confirmada_en IS NULL AND anulada_en IS NULL);
