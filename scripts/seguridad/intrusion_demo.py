#!/usr/bin/env python3
"""Comprueba fronteras HTTP sólo en la demo propia, con datos inventados."""

import argparse
import io
import json
import re
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
import zipfile
from http.cookies import SimpleCookie
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
RESULTADOS = []


class Cliente:
    def __init__(self, base):
        self.base, self.cookies, self.csrf = base, {}, ""
        self.http = urllib.request.build_opener(urllib.request.ProxyHandler({}))

    def pedir(self, metodo, ruta, datos=None, cabeceras=None, csrf=True):
        headers = {"Origin": self.base}
        if self.cookies:
            headers["Cookie"] = "; ".join(f"{k}={v}" for k, v in self.cookies.items())
        if csrf and metodo != "GET":
            headers["X-CSRF-Token"] = self.csrf
        if isinstance(datos, dict):
            datos = json.dumps(datos).encode()
            headers["Content-Type"] = "application/json"
        headers.update(cabeceras or {})
        req = urllib.request.Request(self.base + ruta, datos, headers, method=metodo)
        try:
            respuesta = self.http.open(req, timeout=30)
        except urllib.error.HTTPError as error:
            respuesta = error
        with respuesta:
            cuerpo = respuesta.read(12_000_000)
            for valor in respuesta.headers.get_all("Set-Cookie", []):
                for k, v in SimpleCookie(valor).items():
                    if v.value:
                        self.cookies[k] = v.value
                    else:
                        self.cookies.pop(k, None)
            if cuerpo and "application/json" in respuesta.headers.get("Content-Type", ""):
                objeto = json.loads(cuerpo)
                if isinstance(objeto, dict):
                    self.csrf = objeto.get("csrf", self.csrf)
            return respuesta.status, cuerpo, respuesta.headers

    def entrar(self, cuenta, claves):
        comprobar("pre_sesion", self.pedir("GET", "/api/sesion"), {200})
        respuesta = self.pedir("POST", "/api/sesion", {"cuenta": cuenta, "contrasena": claves[cuenta]})
        comprobar("entrada_" + cuenta, respuesta, {200})
        return json.loads(respuesta[1])


def comprobar(caso, respuesta, estados):
    estado, cuerpo, _ = respuesta
    # Se conservan estados y tamaño, nunca cookies, credenciales o datos de negocio.
    correcto = estado in estados
    RESULTADOS.append({"caso": caso, "estado": estado, "bytes": len(cuerpo), "correcto": correcto})
    if not correcto:
        raise AssertionError(f"{caso}: HTTP {estado}; esperado {sorted(estados)}")
    return cuerpo


def archivo_zip(partes):
    b = io.BytesIO()
    with zipfile.ZipFile(b, "w", zipfile.ZIP_DEFLATED) as z:
        for nombre, contenido in partes.items():
            z.writestr(nombre, contenido)
    return b.getvalue()


def anexo(cliente, nombre, contenido):
    limite = "etf-" + uuid.uuid4().hex
    cuerpo = (f'--{limite}\r\nContent-Disposition: form-data; name="anexo"; filename="{nombre}"\r\n'
              'Content-Type: application/octet-stream\r\n\r\n').encode() + contenido + f"\r\n--{limite}--\r\n".encode()
    return cliente.pedir("POST", "/api/coordinacion/derivaciones", cuerpo,
                         {"Content-Type": "multipart/form-data; boundary=" + limite})


def ejecutar(base):
    claves = json.loads((RAIZ / "despliegue/demo/.local/usuarios.json").read_text())
    clientes = {}
    for cuenta in claves:
        c = clientes[cuenta] = Cliente(base)
        sesion = c.entrar(cuenta, claves)
        if cuenta == "sergio.campos":
            assert sesion["estado"] == "pendiente_alta" and "usuario" not in sesion
    coord, admin = clientes["ines.moreno"], clientes["elena.salas"]
    filas = json.loads(comprobar("bandeja_coordinacion", coord.pedir("GET", "/api/coordinacion/bandeja?limite=100"), {200}))["filas"]
    profesionales = [clientes["marta.cabrera"], clientes["pablo.ruiz"]]
    sesiones = [json.loads(comprobar("sesion_profesional", c.pedir("GET", "/api/sesion"), {200})) for c in profesionales]
    equipos = [s["equipo_id"] for s in sesiones]
    ajeno = next(f["id"] for f in filas if f["equipo_id"] == equipos[1] and f["estado"] in ("aceptada", "en_intervencion"))
    for cuenta, c in clientes.items():
        comprobar("vertical_admin_" + cuenta, c.pedir("GET", "/api/admin/usuarios"), {200} if cuenta == "elena.salas" else {403})
        comprobar("vertical_coordinacion_" + cuenta, c.pedir("GET", "/api/coordinacion/bandeja"),
                   {200} if cuenta in ("ines.moreno", "javier.medina") else {403})
    comprobar("horizontal_ficha_ajena", profesionales[0].pedir("GET", "/api/expedientes/" + ajeno), {403, 404})
    comprobar("horizontal_ficha_propia", profesionales[1].pedir("GET", "/api/expedientes/" + ajeno), {200})
    comprobar("horizontal_word_ajeno", profesionales[0].pedir("POST", f"/api/expedientes/{ajeno}/documentos/anexo1",
               {"idioma": "es"}, {"Idempotency-Key": str(uuid.uuid4())}), {403, 404})
    for sub in ["familia", "menores", "intervencion", "evaluaciones", "hitos", "avisos"]:
        comprobar("horizontal_" + sub, profesionales[0].pedir("GET", f"/api/expedientes/{ajeno}/{sub}"), {403, 404})
    comprobar("horizontal_avisos_equipo", profesionales[0].pedir("GET", f"/api/equipos/{equipos[1]}/avisos/recuento"), {403, 404})
    comprobar("horizontal_spaf_equipo", profesionales[0].pedir("POST", f"/api/equipos/{equipos[1]}/tabla-spaf/2026/1.docx",
               {"idioma": "es"}, {"Idempotency-Key": str(uuid.uuid4())}), {400, 403, 404, 422})
    comprobar("horizontal_carga_otro_equipo", profesionales[0].pedir("POST", "/api/carga-2026",
               {"equipo_id": equipos[1], "casos": [{}]}), {400, 403, 422})
    consulta = clientes["lucia.navarro"]
    estado, cuerpo, _ = consulta.pedir("GET", "/api/consulta/memoria/2026?equipo=" + equipos[0])
    comprobar("consulta_cifras_propias", (estado, cuerpo, None), {200, 404, 409})
    texto = cuerpo.decode(errors="replace")
    assert not re.search(r'"(numero|composicion|nombre|apellidos|expediente_id|narrativas?)"', texto), "Consulta recibe datos no numéricos"
    comprobar("consulta_parametro_extra", consulta.pedir("GET", "/api/consulta/memoria/2026?equipo=" + equipos[0] + "&vista=completa"), {400})
    comprobar("consulta_vista_completa", consulta.pedir("GET", "/api/memoria/2026/cifras/1"), {403})
    for ruta in ["/api/equipo/expedientes", f"/api/expedientes/{ajeno}/menores", "/api/coordinacion/bandeja", "/api/estadistica/spaf", "/api/admin/usuarios"]:
        comprobar("consulta_" + ruta.split("/")[-1], consulta.pedir("GET", ruta), {403})
    comprobar("consulta_ficha_api", consulta.pedir("GET", "/api/expedientes/" + ajeno), {403})
    comprobar("consulta_word_api", consulta.pedir("POST", "/api/documentos/memoria", {"anio": 2026}), {403, 404})
    comprobar("anonimo_ficha", Cliente(base).pedir("GET", "/api/expedientes/" + ajeno), {401})
    comprobar("csrf_ausente", coord.pedir("POST", "/api/coordinacion/bandeja/buscar", {}, csrf=False), {403})
    comprobar("csrf_otro_origen", coord.pedir("POST", "/api/coordinacion/bandeja/buscar", {}, {"Origin": "https://ajeno.invalid"}), {403})
    comprobar("csrf_cross_site", coord.pedir("POST", "/api/coordinacion/bandeja/buscar", {}, {"Sec-Fetch-Site": "cross-site"}), {403})
    for campo in ["equipo_id", "municipio_id", "estado", "limite"]:
        ruta = "/api/coordinacion/bandeja?" + urllib.parse.urlencode({campo: "' OR '1'='1"})
        comprobar("inyeccion_filtro_" + campo, coord.pedir("GET", ruta), {400})
    for i, texto in enumerate(["' OR 1=1 --", "'; SELECT pg_sleep(2); --", "%_\\' OR TRUE --"]):
        inicio = time.monotonic()
        b = comprobar("inyeccion_busqueda_" + str(i), coord.pedir("POST", "/api/coordinacion/bandeja/buscar", {"busqueda": texto, "limite": 20}), {200})
        assert json.loads(b)["total"] == 0 and time.monotonic() - inicio < 2
    comprobar("json_desconocido", coord.pedir("POST", "/api/coordinacion/bandeja/buscar", {"identidad": "otra"}), {400})
    comprobar("json_tamano", coord.pedir("POST", "/api/coordinacion/bandeja/buscar", {"busqueda": "x" * 5000}), {400})
    docx = {"[Content_Types].xml": '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>',
            "word/document.xml": '<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body/></w:document>'}
    casos = {
        "macro": {**docx, "word/vbaProject.bin": b"contenido inerte"},
        "ole": {**docx, "word/embeddings/oleObject1.bin": b"contenido inerte"},
        "rutas": {**docx, "../fuera.xml": b"contenido inerte"},
        "xxe": {**docx, "word/document.xml": '<!DOCTYPE w [<!ENTITY dato SYSTEM "file:///no-existe-etf">]><w/>'},
        "zip_bomba": {**docx, "word/document.xml": b"x" * 34_000_000},
    }
    for nombre, partes in casos.items():
        comprobar("docx_" + nombre, anexo(coord, nombre + ".docx", archivo_zip(partes)), {400, 413, 422})
    comprobar("pdf_formato_invalido", anexo(coord, "Documento.pdf", b"%PDF-1.7\n/JS (contenido inerte)\n%%EOF"), {400, 422})
    comprobar("anexo_tamano", anexo(coord, "Documento.docx", b"x" * 10_000_001), {413})
    for nombre, partes in casos.items():
        contenido = archivo_zip({k.replace("word/", "xl/"): v for k, v in partes.items()})
        comprobar("xlsx_" + nombre, profesionales[0].pedir("POST", "/api/carga-2026/hoja", contenido,
                   {"Content-Type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "X-Equipo-Carga": equipos[0]}), {413, 422})
    # F5.02: amplificación de memoria con cuerpos pequeños (corregida).
    masivo = "[" + "{}," * 100000 + "{}]"
    hoja = {"[Content_Types].xml": '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/></Types>',
            "_rels/.rels": '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>',
            "xl/workbook.xml": '<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Casos" sheetId="1" r:id="rId1"/></sheets></workbook>',
            "xl/_rels/workbook.xml.rels": '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/></Relationships>',
            "xl/sharedStrings.xml": '<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>' + masivo + '</t></si></sst>',
            "xl/worksheets/sheet1.xml": '<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1">'
            + "".join(f'<c r="{c}1" t="inlineStr"><is><t>{v}</t></is></c>' for c, v in zip("ABCDEFG", ["fecha_derivacion", "estado", "tipo_anexo", "campos_derivacion", "episodios", "familia", "menores"]))
            + "</row>" + "".join(f'<row r="{i}"><c r="A{i}" t="inlineStr"><is><t>2026-02-01</t></is></c><c r="B{i}" t="inlineStr"><is><t>aceptada</t></is></c><c r="C{i}" t="inlineStr"><is><t>anexo3</t></is></c>'
                                  + "".join(f'<c r="{c}{i}" t="s"><v>0</v></c>' for c in "DEFG") + "</row>" for i in range(2, 12)) + "</sheetData></worksheet>"}
    comprobar("xlsx_json_masivo", profesionales[0].pedir("POST", "/api/carga-2026/hoja", archivo_zip(hoja),
               {"Content-Type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "X-Equipo-Carga": equipos[0]}), {413, 422})
    cuerpo_masivo = ('{"equipo_id":"' + equipos[0] + '","casos":[' + "{}," * 699000 + "{}]}").encode()
    comprobar("carga_json_masiva", profesionales[0].pedir("POST", "/api/carga-2026", cuerpo_masivo, {"Content-Type": "application/json"}), {400, 413})
    b = io.BytesIO()
    with zipfile.ZipFile(b, "w", zipfile.ZIP_STORED) as z:
        z.writestr("[Content_Types].xml", '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>')
        z.writestr("word/document.xml", '<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body' + ' x=""' * 1590000 + '/></w:document>')
    comprobar("docx_atributos_masivos", anexo(coord, "Atributos.docx", b.getvalue()), {400, 413, 422})
    comprobar("app_sigue_viva", Cliente(base).pedir("GET", "/salud"), {200})
    respuesta = coord.pedir("GET", "/api/coordinacion/bandeja?limite=-1")
    cuerpo = comprobar("error_sin_detalles", respuesta, {400}).decode()
    assert not re.search(r"SELECT|postgres|panic|goroutine|/home/|password", cuerpo, re.I)
    headers = respuesta[2]
    for k, v in {"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer", "X-Frame-Options": "DENY"}.items():
        assert headers[k] == v, k
    assert "frame-ancestors 'none'" in headers["Content-Security-Policy"]
    assert "Strict-Transport-Security" in headers
    fijacion = Cliente(base)
    fijacion.entrar("javier.medina", claves)
    anterior = dict(fijacion.cookies)
    respuesta = fijacion.pedir("POST", "/api/sesion", {"cuenta": "javier.medina", "contrasena": claves["javier.medina"]})
    comprobar("fijacion_rotacion", respuesta, {200})
    assert fijacion.cookies["__Host-etf-sesion"] != anterior["__Host-etf-sesion"]
    for valor in respuesta[2].get_all("Set-Cookie", []):
        if "__Host-etf-sesion=" in valor:
            assert "Secure" in valor and "HttpOnly" in valor and "SameSite=Strict" in valor and "Path=/" in valor and "Domain=" not in valor
    vieja = Cliente(base)
    vieja.cookies = anterior
    comprobar("fijacion_anterior_revocada", vieja.pedir("GET", "/api/sesion"), {401})
    copiar = dict(fijacion.cookies)
    comprobar("salida", fijacion.pedir("DELETE", "/api/sesion"), {204})
    vieja.cookies = copiar
    comprobar("salida_revocada", vieja.pedir("GET", "/api/sesion"), {401})
    # Una X-Forwarded-For enviada por el cliente no renueva el cupo: nginx la sustituye.
    falsa = Cliente(base)
    comprobar("xff_pre_sesion", falsa.pedir("GET", "/api/sesion"), {200})
    vistos = set()
    for i in range(32):
        estado = falsa.pedir("POST", "/api/sesion", {"cuenta": f"inventada{i}", "contrasena": "mala"}, {"X-Forwarded-For": f"203.0.113.{i}"})[0]
        vistos.add(estado)
    assert 429 in vistos, "X-Forwarded-For del cliente eludió el límite de intentos"
    RESULTADOS.append({"caso": "xff_falsificada_no_elude_limite", "estado": 429, "bytes": 0, "correcto": True})
    for i in range(105):
        respuesta = Cliente(base).pedir("GET", "/api/sesion")
        if respuesta[0] == 429:
            assert respuesta[2]["Retry-After"] == "60"
            comprobar("limite_tasa", respuesta, {429})
            break
    else:
        raise AssertionError("no se observó el límite de peticiones de sesión")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--salida", type=Path, required=True)
    args = parser.parse_args()
    puerto = (RAIZ / "despliegue/demo/.local/puerto").read_text().strip()
    if not puerto.isdecimal() or not 1024 <= int(puerto) <= 65535:
        raise SystemExit("Puerto de demo no válido")
    fallo = None
    try:
        ejecutar("http://127.0.0.1:" + puerto)
    except Exception as error:
        fallo = type(error).__name__ + ": " + str(error)
    args.salida.write_text(json.dumps({"resultados": RESULTADOS, "fallo": fallo}, ensure_ascii=False, indent=2) + "\n")
    print(f"F5.02 HTTP: {len(RESULTADOS)} comprobaciones; " + ("OK" if fallo is None else fallo))
    if fallo:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
