#!/usr/bin/env python3
"""Regenera las muestras sanitizadas y las firmas de prueba, sin conexión de datos."""
import datetime
import io
import os
from pathlib import Path
import re
import subprocess
import tempfile
import zipfile

from asn1crypto import keys, x509 as asn1x509
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.x509.oid import NameOID
from pypdf import PdfReader, PdfWriter
from pyhanko.pdf_utils.incremental_writer import IncrementalPdfFileWriter
from pyhanko.sign import fields, signers
from pyhanko.pdf_utils.reader import PdfFileReader
from pyhanko.sign.validation import validate_pdf_signature
from pyhanko_certvalidator import ValidationContext
from pyhanko_certvalidator.registry import SimpleCertificateStore

DESTINO = Path(__file__).resolve().parent
RAIZ = DESTINO.parents[4]


def generar():
    with tempfile.TemporaryDirectory(prefix="resp4-", dir=os.environ["TMPDIR"]) as zona:
        temporal = Path(zona)
        for numero in (3, 4):
            fuente = DESTINO / f"anexo{numero}.odt"
            with zipfile.ZipFile(io.BytesIO(fuente.read_bytes())) as original, zipfile.ZipFile(fuente, "w") as limpio:
                for parte in original.infolist():
                    contenido = original.read(parte)
                    if parte.filename.endswith(".xml"):
                        contenido = re.sub(rb"@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}", b"@servicios.invalid", contenido)
                    if parte.filename == "meta.xml":
                        texto = re.sub(r"<(dc:creator|meta:initial-creator)>.*?</\1>",
                                       r"<\1>Expedientes ETF</\1>", contenido.decode())
                        contenido = texto.encode()
                    limpio.writestr(parte, contenido)
        entorno = dict(os.environ, LD_LIBRARY_PATH="/usr/lib/libreoffice/program", SAL_USE_VCLPLUGIN="svp")
        subprocess.run(["libreoffice", f"-env:UserInstallation={temporal.as_uri()}/lo", "--headless",
                        "--convert-to", "pdf", "--outdir", str(temporal),
                        *[str(DESTINO / f"anexo{n}.odt") for n in (3, 4)]], env=entorno, check=True)
        clave = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        nombre = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "Expedientes ETF")])
        ahora = datetime.datetime.now(datetime.timezone.utc)
        certificado = (x509.CertificateBuilder().subject_name(nombre).issuer_name(nombre)
                       .public_key(clave.public_key()).serial_number(x509.random_serial_number())
                       .not_valid_before(ahora - datetime.timedelta(days=1))
                       .not_valid_after(ahora + datetime.timedelta(days=3650))
                       .add_extension(x509.BasicConstraints(ca=True, path_length=None), critical=True)
                       .add_extension(x509.KeyUsage(digital_signature=True, content_commitment=True, key_encipherment=False, data_encipherment=False, key_agreement=False, key_cert_sign=True, crl_sign=True, encipher_only=None, decipher_only=None), critical=True)
                       .sign(clave, hashes.SHA256()))
        firmante = signers.SimpleSigner(
            signing_cert=asn1x509.Certificate.load(certificado.public_bytes(serialization.Encoding.DER)),
            signing_key=keys.PrivateKeyInfo.load(clave.private_bytes(serialization.Encoding.DER,
                    serialization.PrivateFormat.PKCS8, serialization.NoEncryption())),
            cert_registry=SimpleCertificateStore())
        for numero in (3, 4):
            lector = PdfReader(temporal / f"anexo{numero}.pdf")
            escritor = PdfWriter(clone_from=lector)
            if "/Metadata" in escritor.root_object:
                del escritor.root_object["/Metadata"]
            escritor.metadata.clear()
            escritor.add_metadata({"/Author": "Expedientes ETF", "/Creator": "Expedientes ETF"})
            limpio = io.BytesIO()
            escritor.write(limpio)
            (DESTINO / f"anexo{numero}.pdf").write_bytes(limpio.getvalue())
            limpio.seek(0)
            firmado = signers.sign_pdf(IncrementalPdfFileWriter(limpio),
                signature_meta=signers.PdfSignatureMetadata(field_name="FirmaETF"), signer=firmante,
                new_field_spec=fields.SigFieldSpec(sig_field_name="FirmaETF"))
            (DESTINO / f"anexo{numero}-firmado.pdf").write_bytes(firmado.getvalue())
            firmado.seek(0)
            firma = PdfFileReader(firmado).embedded_signatures[0]
            validacion = validate_pdf_signature(firma, signer_validation_context=ValidationContext(
                trust_roots=[firma.signer_cert], allow_fetching=False))
            assert validacion.intact and validacion.valid and validacion.trusted
            print(f"anexo{numero}: firma íntegra y válida; certificado autofirmado local; autor limpio")


if __name__ == "__main__":
    generar()
