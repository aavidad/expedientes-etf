#!/usr/bin/env python3
"""Genera el corpus inventado de actas, PDF y firma con certificado local."""
import copy
import json
import os
from pathlib import Path
import subprocess
import tempfile
from xml.etree import ElementTree as E
from zipfile import ZipFile

RAIZ = Path(__file__).resolve().parents[2]
DESTINO = RAIZ / "internal/adaptadores/anexos/actas/testdata"
W = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"


def fechar(ruta, fecha, etiqueta, subprograma):
    with ZipFile(ruta) as z:
        partes = [(i, z.read(i)) for i in z.infolist()]
    with ZipFile(ruta, "w") as z:
        for info, b in partes:
            if info.filename == "word/document.xml":
                r = E.fromstring(b)
                for p in list(r.iter(f"{{{W}}}p")):
                    textos = p.findall(f".//{{{W}}}t")
                    valor = "".join(t.text or "" for t in textos)
                    if "a la fecha de la firma electrónica" in valor.lower():
                        textos[0].text = etiqueta + fecha
                        for t in textos[1:]:
                            t.text = ""
                    if valor.startswith("Descripción de la situación familiar:"):
                        # Dato conocido en este corpus; no altera la plantilla oficial.
                        nuevo = copy.deepcopy(p)
                        nt = nuevo.findall(f".//{{{W}}}t")
                        nt[0].text = "Subprograma: " + subprograma
                        for t in nt[1:]:
                            t.text = ""
                        for padre in r.iter():
                            if p in list(padre):
                                padre.insert(list(padre).index(p) + 1, nuevo)
                                break
                b = E.tostring(r, encoding="utf-8", xml_declaration=True)
            z.writestr(info, b)


def main():
    from pyhanko.pdf_utils.incremental_writer import IncrementalPdfFileWriter
    from pyhanko.sign import signers
    from pyhanko.sign.validation import validate_pdf_signature
    from pyhanko.pdf_utils.reader import PdfFileReader
    from pyhanko_certvalidator import ValidationContext

    subprocess.run(["go", "run", "./scripts/actas"], cwd=RAIZ, check=True)
    casos = json.loads((DESTINO / "datos.json").read_text())
    with tempfile.TemporaryDirectory(prefix="resp2-actas-") as temporal:
        tmp = Path(temporal)
        for c in casos:
            ruta = DESTINO / (c["Nombre"] + ".docx")
            fechar(ruta, c["Fecha"], c["EtiquetaFecha"], c["Subprograma"])
        entorno = dict(os.environ)
        entorno.pop("LD_LIBRARY_PATH", None)
        subprocess.run(["libreoffice", "-env:UserInstallation=" + (tmp / "lo").as_uri(),
                        "--headless", "--convert-to", "pdf", "--outdir", str(DESTINO),
                        *[str(DESTINO / (c["Nombre"] + ".docx")) for c in casos]], check=True, env=entorno)
        clave = tmp / "clave.pem"
        certificado = tmp / "certificado.pem"
        subprocess.run(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
                        "-keyout", str(clave), "-out", str(certificado), "-days", "3650",
                        "-addext", "keyUsage=digitalSignature,nonRepudiation",
                        "-subj", "/CN=Certificado local de prueba ETF"], check=True,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        firmante = signers.SimpleSigner.load(str(clave), str(certificado))
        with (DESTINO / "aceptacion-2022.pdf").open("rb") as entrada:
            salida = DESTINO / "aceptacion-2022-firmada.pdf"
            with salida.open("wb") as f:
                signers.PdfSigner(signers.PdfSignatureMetadata(field_name="FirmaPrueba"),
                                  signer=firmante).sign_pdf(IncrementalPdfFileWriter(entrada), output=f)
        with salida.open("rb") as f:
            firma = PdfFileReader(f).embedded_signatures[0]
            estado = validate_pdf_signature(firma, signer_validation_context=ValidationContext(
                trust_roots=[firmante.signing_cert], allow_fetching=False))
            if not estado.intact or not estado.valid or not estado.trusted:
                raise RuntimeError("La firma local no conserva la integridad del PDF.")
            print("RESP2_FIRMA_LOCAL: intacta y criptográficamente válida; certificado autofirmado de prueba.")
        # El certificado es público. La clave desaparece con el temporal.
        (DESTINO / "certificado-prueba.pem").write_bytes(certificado.read_bytes())


if __name__ == "__main__":
    main()
