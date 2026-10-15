-- Cambios inventados ocurridos después de la copia, mientras AD sigue activo.
BEGIN;
SELECT set_config('app.usuario_id',(SELECT id::text FROM identidad.usuario WHERE cuenta_ad='elena.salas'),true),set_config('app.perfiles','administracion',true);
SET ROLE etf_app;
DO $$ DECLARE u record; ahora timestamptz:=clock_timestamp(); equipo uuid; BEGIN
 SELECT id INTO equipo FROM catalogo.equipo WHERE codigo='04';
 SELECT id,version INTO u FROM identidad.usuario WHERE cuenta_ad='javier.medina';
 PERFORM identidad.cambiar_perfiles(u.id,u.version,'baja','[]',NULL);
 SELECT id,version INTO u FROM identidad.usuario WHERE cuenta_ad='lucia.navarro';
 PERFORM identidad.cambiar_perfiles(u.id,u.version,'activo',jsonb_build_array(jsonb_build_object('perfil','coordinacion','desde',ahora,'hasta',NULL)),NULL);
 SELECT id,version INTO u FROM identidad.usuario WHERE cuenta_ad='marta.cabrera';
 PERFORM identidad.cambiar_perfiles(u.id,u.version,'activo',jsonb_build_array(jsonb_build_object('perfil','profesional_etf','desde',ahora,'hasta',NULL)),jsonb_build_object('equipo_id',equipo,'puesto','ts','desde',ahora,'hasta',NULL));
END $$;
RESET ROLE;
INSERT INTO identidad.usuario(object_guid_ad,cuenta_ad,nombre_mostrado,primer_acceso_en,ultimo_acceso_en)
VALUES(gen_random_uuid(),'raquel.santos','Raquel Santos López',clock_timestamp(),clock_timestamp());
COMMIT;
