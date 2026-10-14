#!/usr/bin/env python3
"""Ensayo local desechable; requiere Compose levantado y ninguna base real."""
import json
import pathlib
import shutil
import subprocess
import tempfile

RAIZ = pathlib.Path(__file__).resolve().parents[2]
BASE = "etf_ensayo_f003"
COMPOSE = ["docker", "compose", "--project-directory", str(RAIZ)]


def orden(args, entrada=None, correcto=True):
    resultado = subprocess.run(args, input=entrada, text=True, capture_output=True, timeout=90)
    if correcto and resultado.returncode:
        # No volcar errores SQL que pudieran llevar contenido de una fila.
        raise RuntimeError("ensayo.orden_fallida")
    return resultado


def sql(texto, base=BASE):
    return orden(COMPOSE + ["exec", "-T", "postgres", "psql", "-X", "-qAt", "-v", "ON_ERROR_STOP=1",
                            "-U", "etf_instalador", "-d", base], texto).stdout.strip()


def migrar(ruta=None, correcto=True, extra=None):
    args = COMPOSE + ["run", "--rm", "--no-deps", "-e", "ETF_BD_NOMBRE=" + BASE]
    if ruta:
        args += ["-v", str(ruta) + ":/opt/etf/migraciones:ro"]
    args += extra or []
    return orden(args + ["migrar"], correcto=correcto)


def declaracion(numero, dependencias):
    return {"numero": numero, "archivo": numero + "_ensayo.sql", "depende_de": dependencias}


def escribir(ruta, declaraciones, archivos):
    (ruta / "plataforma" / "manifiesto.json").write_text(json.dumps(declaraciones))
    for nombre, contenido in archivos.items():
        (ruta / "plataforma" / nombre).write_text(contenido)


def main():
    # No destruir una base preexistente: podría pertenecer a otro ensayo.
    assert sql("SELECT count(*) FROM pg_database WHERE datname='" + BASE + "';", "etf") == "0"
    sql("CREATE DATABASE " + BASE + " OWNER etf_propietario TEMPLATE template0;", "etf")
    try:
        assert "migracion.aplicada.plataforma.0001" in migrar().stdout
        assert "migracion.existente.plataforma.0001" in migrar().stdout
        assert sql("SELECT count(*) FROM plataforma.migracion_aplicada;") == "1"
        print("ENSAYO-OK limpia y repetida: 1 migración, etf_migrador")

        assert sql("SELECT rolcanlogin,rolsuper,rolcreatedb,rolcreaterole,rolbypassrls FROM pg_roles WHERE rolname='etf_app';") == "t|f|f|f|f"
        assert sql("SELECT pg_has_role('etf_app','etf_propietario','MEMBER'),has_schema_privilege('etf_app','plataforma','CREATE'),has_table_privilege('etf_app','plataforma.migracion_aplicada','INSERT');") == "f|f|f"
        assert sql("SELECT relrowsecurity,relforcerowsecurity,pg_get_userbyid(relowner) FROM pg_class WHERE oid='plataforma.migracion_aplicada'::regclass;") == "t|t|etf_propietario"
        assert migrar(correcto=False, extra=["-e", "ETF_MIGRADOR_CONTRASENA=credencial_invalida"]).returncode != 0
        assert sql("SELECT rolcanlogin,rolsuper,rolpassword IS NULL FROM pg_authid WHERE rolname='etf_instalador';") == "t|t|t"
        acceso = orden(COMPOSE + ["exec", "-T", "postgres", "sh", "-c",
                       'PGPASSWORD="$POSTGRES_PASSWORD" psql -X --no-password -h 127.0.0.1 -U etf_instalador -d etf -c "SELECT 1;"'], correcto=False)
        assert acceso.returncode != 0
        assert "scram-sha-256" in sql("SELECT auth_method FROM pg_hba_file_rules WHERE type='host';")
        print("ENSAYO-OK propietario, RLS forzada, app sin CREATE/escritura y contraseña incorrecta rechazada")
        print("ENSAYO-OK contraseña del instalador retirada y acceso TCP rechazado")

        with tempfile.TemporaryDirectory(prefix="etf-f003-") as temporal:
            ruta = pathlib.Path(temporal) / "migraciones"
            shutil.copytree(RAIZ / "migraciones", ruta)
            # Docker ejecuta con uid 70; sólo se exponen ficheros de ensayo.
            pathlib.Path(temporal).chmod(0o755)
            manifiesto = json.loads((ruta / "plataforma" / "manifiesto.json").read_text())
            primera = ruta / "plataforma" / "0001_control.sql"
            contenido = primera.read_text()
            primera.write_text(contenido + "\n-- huella modificada\n")
            fallo = migrar(ruta, correcto=False)
            assert fallo.returncode != 0 and 'migracion.huella' in fallo.stdout
            assert sql("SELECT count(*) FROM plataforma.migracion_aplicada;") == "1"
            primera.write_text(contenido)
            print("ENSAYO-OK huella alterada rechazada antes de escribir")

            dep = [{"modulo": "plataforma", "numero": "0001"}]
            segunda = declaracion("0002", dep)
            escribir(ruta, manifiesto + [segunda], {"0002_ensayo.sql": "CREATE TABLE plataforma.ensayo(valor integer); SELECT 1/0;"})
            assert migrar(ruta, correcto=False).returncode != 0
            assert sql("SELECT to_regclass('plataforma.ensayo') IS NULL;") == "t"
            assert sql("SELECT count(*) FROM plataforma.migracion_aplicada;") == "1"
            print("ENSAYO-OK fallo SQL revierte DDL y control en la misma transacción")

            # Declaración desordenada: 0003 depende de 0002, no del orden JSON.
            tercera = declaracion("0003", [{"modulo": "plataforma", "numero": "0002"}])
            escribir(ruta, manifiesto + [tercera, segunda], {
                "0002_ensayo.sql": "SELECT pg_sleep(1);",
                "0003_ensayo.sql": "DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM plataforma.migracion_aplicada WHERE numero='0002') THEN RAISE EXCEPTION 'ensayo.orden'; END IF; END $$;",
            })
            args = COMPOSE + ["run", "--rm", "--no-deps", "-e", "ETF_BD_NOMBRE=" + BASE,
                              "-v", str(ruta) + ":/opt/etf/migraciones:ro", "migrar"]
            procesos = [subprocess.Popen(args, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE) for _ in range(2)]
            for proceso in procesos:
                proceso.communicate(timeout=90)
                assert proceso.returncode == 0
            assert sql("SELECT count(*) FROM plataforma.migracion_aplicada;") == "3"
            print("ENSAYO-OK dependencias desordenadas y dos ejecutores concurrentes: 3 filas, sin duplicados")

            escribir(ruta, manifiesto + [tercera, segunda], {})
            # Quitar una versión ya aplicada se rechaza, aunque permanezca el SQL.
            sql("DELETE FROM plataforma.migracion_aplicada WHERE numero='0002';")
            fallo = migrar(ruta, correcto=False)
            assert fallo.returncode != 0 and 'migracion.retroceso' in fallo.stdout
            print("ENSAYO-OK historial con retroceso rechazado")
    finally:
        sql("DROP DATABASE " + BASE + " WITH (FORCE);", "etf")
    print("ENSAYO-OK base desechable eliminada")


if __name__ == "__main__":
    main()
