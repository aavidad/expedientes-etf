// Package migracion es infraestructura exclusiva del ejecutor de despliegue.
package migracion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
)

type Referencia struct {
	Modulo string `json:"modulo"`
	Numero string `json:"numero"`
}

type Declaracion struct {
	Numero    string       `json:"numero"`
	Archivo   string       `json:"archivo"`
	DependeDe []Referencia `json:"depende_de"`
}

type Migracion struct {
	Referencia
	Nombre       string
	Dependencias []Referencia
	SQL          string
	Huella       string
}

func (r Referencia) ID() string { return r.Modulo + "/" + r.Numero }

// LeerPlan comprueba manifiestos completos y ordena un grafo causal, no carpetas.
func LeerPlan(disco fs.FS) ([]Migracion, error) {
	modulos := map[string]bool{"plataforma": true, "identidad": true, "catalogo": true, "paquete": true, "auditoria": true, "expediente": true, "evaluacion": true, "documento": true, "actividad": true, "estadistica": true, "carga": true}
	numero := regexp.MustCompile(`^[0-9]{4}$`)
	nombre := regexp.MustCompile(`^[0-9]{4}_[a-z0-9_]+\.sql$`)
	todos := map[string]Migracion{}
	err := fs.WalkDir(disco, ".", func(ruta string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.Type()&fs.ModeSymlink != 0 {
			return errors.New("migracion.enlace")
		}
		if entrada.IsDir() || path.Base(ruta) != "manifiesto.json" {
			return nil
		}
		modulo := path.Dir(ruta)
		if !modulos[modulo] {
			return errors.New("migracion.modulo")
		}
		contenido, err := fs.ReadFile(disco, ruta)
		if err != nil {
			return err
		}
		var declaraciones []Declaracion
		decoder := json.NewDecoder(strings.NewReader(string(contenido)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&declaraciones); err != nil {
			return errors.New("migracion.manifiesto")
		}
		// Un manifiesto no admite otro documento JSON detrás.
		if err := decoder.Decode(new(any)); err != io.EOF {
			return errors.New("migracion.manifiesto")
		}
		for _, d := range declaraciones {
			if !numero.MatchString(d.Numero) || d.Numero == "0000" || !nombre.MatchString(d.Archivo) || !strings.HasPrefix(d.Archivo, d.Numero+"_") || d.DependeDe == nil {
				return errors.New("migracion.declaracion")
			}
			ref := Referencia{modulo, d.Numero}
			if _, existe := todos[ref.ID()]; existe {
				return errors.New("migracion.duplicada")
			}
			vistas := map[string]bool{}
			for _, dep := range d.DependeDe {
				if !modulos[dep.Modulo] || !numero.MatchString(dep.Numero) || vistas[dep.ID()] || (dep.Modulo == modulo && dep.Numero >= d.Numero) {
					return errors.New("migracion.dependencia")
				}
				vistas[dep.ID()] = true
			}
			sort.Slice(d.DependeDe, func(i, j int) bool { return d.DependeDe[i].ID() < d.DependeDe[j].ID() })
			sql, err := fs.ReadFile(disco, path.Join(modulo, d.Archivo))
			if err != nil {
				return errors.New("migracion.archivo")
			}
			if len(sql) == 0 {
				return errors.New("migracion.archivo")
			}
			deps, _ := json.Marshal(d.DependeDe)
			suma := sha256.Sum256(append(append(sql, 0), deps...))
			todos[ref.ID()] = Migracion{ref, d.Archivo, d.DependeDe, string(sql), hex.EncodeToString(suma[:])}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Ningún SQL puede quedar fuera del manifiesto.
	err = fs.WalkDir(disco, ".", func(ruta string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() || !strings.HasSuffix(ruta, ".sql") {
			return nil
		}
		for _, m := range todos {
			if ruta == path.Join(m.Modulo, m.Nombre) {
				return nil
			}
		}
		return errors.New("migracion.sin_declarar")
	})
	if err != nil {
		return nil, err
	}
	raiz := "plataforma/0001"
	if _, existe := todos[raiz]; !existe {
		return nil, errors.New("migracion.sin_control")
	}
	estado := map[string]int{}
	var orden []Migracion
	var visitar func(string) error
	visitar = func(id string) error {
		if estado[id] == 1 {
			return errors.New("migracion.ciclo")
		}
		if estado[id] == 2 {
			return nil
		}
		m, existe := todos[id]
		if !existe {
			return errors.New("migracion.dependencia_ausente")
		}
		if id != raiz && len(m.Dependencias) == 0 {
			return errors.New("migracion.sin_dependencia_control")
		}
		estado[id] = 1
		for _, dep := range m.Dependencias {
			if err := visitar(dep.ID()); err != nil {
				return err
			}
		}
		estado[id] = 2
		orden = append(orden, m)
		return nil
	}
	ids := make([]string, 0, len(todos))
	for id := range todos {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visitar(id); err != nil {
			return nil, err
		}
	}
	if len(orden) == 0 || orden[0].ID() != raiz {
		return nil, errors.New("migracion.orden_control")
	}
	return orden, nil
}
