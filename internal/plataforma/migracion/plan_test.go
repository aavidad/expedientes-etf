package migracion

import (
	"strings"
	"testing"
	"testing/fstest"
)

func archivos(manifiesto string) fstest.MapFS {
	return fstest.MapFS{
		"plataforma/manifiesto.json":  {Data: []byte(manifiesto)},
		"plataforma/0001_control.sql": {Data: []byte("SELECT 1;")},
	}
}

func TestPlanCausal(t *testing.T) {
	disco := archivos(`[{"numero":"0001","archivo":"0001_control.sql","depende_de":[]}]`)
	disco["identidad/manifiesto.json"] = &fstest.MapFile{Data: []byte(`[{"numero":"0001","archivo":"0001_identidad.sql","depende_de":[{"modulo":"catalogo","numero":"0001"}]}]`)}
	disco["identidad/0001_identidad.sql"] = &fstest.MapFile{Data: []byte("SELECT 3;")}
	disco["catalogo/manifiesto.json"] = &fstest.MapFile{Data: []byte(`[{"numero":"0001","archivo":"0001_catalogo.sql","depende_de":[{"modulo":"plataforma","numero":"0001"}]}]`)}
	disco["catalogo/0001_catalogo.sql"] = &fstest.MapFile{Data: []byte("SELECT 2;")}
	plan, err := LeerPlan(disco)
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"plataforma/0001", "catalogo/0001", "identidad/0001"} {
		if plan[i].ID() != id {
			t.Fatalf("orden: %v", plan)
		}
	}
	// Cambiar la dependencia altera la huella incluso con el mismo SQL.
	huella := plan[2].Huella
	disco["identidad/manifiesto.json"].Data = []byte(`[{"numero":"0001","archivo":"0001_identidad.sql","depende_de":[{"modulo":"plataforma","numero":"0001"}]}]`)
	plan, err = LeerPlan(disco)
	if err != nil {
		t.Fatal(err)
	}
	if plan[2].Huella == huella {
		t.Fatal("dependencia sin huella")
	}
}

func TestPlanRechazaErrores(t *testing.T) {
	casos := []struct{ nombre, manifiesto, clave string }{
		{"autodependencia", `[{"numero":"0001","archivo":"0001_control.sql","depende_de":[{"modulo":"plataforma","numero":"0001"}]}]`, "migracion.dependencia"},
		{"ausente", `[{"numero":"0001","archivo":"0001_control.sql","depende_de":[{"modulo":"catalogo","numero":"0001"}]}]`, "migracion.dependencia_ausente"},
		{"duplicada", `[{"numero":"0001","archivo":"0001_control.sql","depende_de":[]},{"numero":"0001","archivo":"0001_control.sql","depende_de":[]}]`, "migracion.duplicada"},
		{"ruta", `[{"numero":"0001","archivo":"../0001_control.sql","depende_de":[]}]`, "migracion.declaracion"},
		{"sin_dependencias", `[{"numero":"0001","archivo":"0001_control.sql"}]`, "migracion.declaracion"},
		{"json_extra", `[{"numero":"0001","archivo":"0001_control.sql","depende_de":[]}] {}`, "migracion.manifiesto"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := LeerPlan(archivos(caso.manifiesto))
			if err == nil || err.Error() != caso.clave {
				t.Fatalf("error: %v", err)
			}
		})
	}
	disco := archivos(`[{"numero":"0001","archivo":"0001_control.sql","depende_de":[]}]`)
	disco["plataforma/0002_huerfana.sql"] = &fstest.MapFile{Data: []byte("SELECT 2;")}
	if _, err := LeerPlan(disco); err == nil || err.Error() != "migracion.sin_declarar" {
		t.Fatalf("SQL omitido: %v", err)
	}
}

func TestRechazaNumeroPosteriorYCicloEntreModulos(t *testing.T) {
	disco := archivos(`[{"numero":"0001","archivo":"0001_control.sql","depende_de":[{"modulo":"plataforma","numero":"0002"}]}]`)
	if _, err := LeerPlan(disco); err == nil || err.Error() != "migracion.dependencia" {
		t.Fatalf("número posterior: %v", err)
	}
	disco = archivos(`[{"numero":"0001","archivo":"0001_control.sql","depende_de":[{"modulo":"catalogo","numero":"0001"}]}]`)
	disco["catalogo/manifiesto.json"] = &fstest.MapFile{Data: []byte(`[{"numero":"0001","archivo":"0001_catalogo.sql","depende_de":[{"modulo":"plataforma","numero":"0001"}]}]`)}
	disco["catalogo/0001_catalogo.sql"] = &fstest.MapFile{Data: []byte("SELECT 1;")}
	if _, err := LeerPlan(disco); err == nil || err.Error() != "migracion.ciclo" {
		t.Fatalf("ciclo: %v", err)
	}
}

func TestHuellaSQLYGuion(t *testing.T) {
	disco := archivos(`[{"numero":"0001","archivo":"0001_control.sql","depende_de":[]}]`)
	antes, _ := LeerPlan(disco)
	disco["plataforma/0001_control.sql"].Data = []byte("SELECT 2;")
	despues, _ := LeerPlan(disco)
	if antes[0].Huella == despues[0].Huella {
		t.Fatal("SQL sin huella")
	}
	guion := guion(despues)
	if strings.Index(guion, "migracion.huella") > strings.Index(guion, "SELECT 2;") {
		t.Fatal("validación después de escribir")
	}
}
