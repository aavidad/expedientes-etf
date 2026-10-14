// Comprobador de literales y claves en Go, basado en el AST estándar.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	claves := []string{}
	raiz := "."
	if len(os.Args) > 1 {
		raiz = os.Args[1]
	}
	fallos := false
	conjunto := regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	prosa := regexp.MustCompile(`[[:alpha:]ÁÉÍÓÚáéíóúñÑ]+\s+[[:alpha:]ÁÉÍÓÚáéíóúñÑ]+`)
	err := filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (strings.HasPrefix(d.Name(), ".") && ruta != "." || d.Name() == "node_modules" || d.Name() == "scripts" || d.Name() == "dist") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		posiciones := token.NewFileSet()
		archivo, err := parser.ParseFile(posiciones, ruta, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		esTecnico := marcasTecnicas(archivo, posiciones)
		ast.Inspect(archivo, func(n ast.Node) bool {
			literal, ok := n.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING || esTecnico(literal) {
				return true
			}
			valor, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			// Importaciones, gramáticas, MIME, CSS y políticas HTTP son datos técnicos.
			tecnico := strings.ContainsAny(valor, "{}[]\\;\n") || strings.Contains(valor, "/") || strings.HasPrefix(valor, "GET") || strings.HasPrefix(valor, "HEAD")
			if !tecnico && prosa.MatchString(valor) {
				fmt.Fprintln(os.Stderr, posiciones.Position(literal.Pos()), "literal visible:", valor)
				fallos = true
			}
			return true
		})
		ast.Inspect(archivo, func(n ast.Node) bool {
			informar := func(lit *ast.BasicLit) {
				if esTecnico(lit) {
					return
				}
				valor, _ := strconv.Unquote(lit.Value)
				if valor != "" && !conjunto.MatchString(valor) {
					fmt.Fprintln(os.Stderr, posiciones.Position(lit.Pos()), "respuesta visible sin clave:", valor)
					fallos = true
				}
			}
			if llamada, ok := n.(*ast.CallExpr); ok {
				if sel, ok := llamada.Fun.(*ast.SelectorExpr); ok {
					nombre := sel.Sel.Name
					if !esTecnico(llamada) && (nombre == "Write" || nombre == "WriteString" || nombre == "Fprint" || nombre == "Fprintln" || nombre == "Fprintf") {
						// El único generador CSS de la app recibe valores de tokens validados.
						css, ok := sel.X.(*ast.Ident)
						if !ok || css.Name != "css" {
							argumentos := llamada.Args
							if strings.HasPrefix(nombre, "Fprint") && len(argumentos) > 0 {
								argumentos = argumentos[1:]
							}
							for _, expr := range argumentos {
								literalesVisibles(expr, map[*ast.Object]bool{}, informar)
							}
						}
					}
				}
			}
			if campo, ok := n.(*ast.KeyValueExpr); ok {
				nombre := ""
				switch clave := campo.Key.(type) {
				case *ast.Ident:
					nombre = clave.Name
				case *ast.BasicLit:
					if clave.Kind == token.STRING {
						nombre, _ = strconv.Unquote(clave.Value)
					}
				}
				switch strings.ToLower(nombre) {
				case "mensaje", "texto", "titulo", "etiqueta", "descripcion", "ayuda":
					literalesVisibles(campo.Value, map[*ast.Object]bool{}, informar)
				}
			}
			var expresiones []ast.Expr
			switch nodo := n.(type) {
			case *ast.IndexExpr:
				// Sólo índices de mapas de textos, nunca claves de configuración/tokens.
				if id, ok := nodo.X.(*ast.Ident); ok && (id.Name == "textos" || id.Name == "traduccion") {
					expresiones = []ast.Expr{nodo.Index}
				}
			case *ast.CallExpr:
				nombre := ""
				switch fun := nodo.Fun.(type) {
				case *ast.Ident:
					nombre = fun.Name
				case *ast.SelectorExpr:
					nombre = fun.Sel.Name
				}
				if nombre == "responderError" || esError(nodo.Fun) {
					expresiones = nodo.Args
					indice := 0
					if nombre == "responderError" {
						indice = 2
					}
					if len(nodo.Args) > indice {
						if lit, ok := nodo.Args[indice].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							valor, _ := strconv.Unquote(lit.Value)
							if !conjunto.MatchString(valor) {
								fmt.Fprintln(os.Stderr, posiciones.Position(lit.Pos()), "error sin clave:", valor)
								fallos = true
							}
						}
					}
				}
			}
			for _, expr := range expresiones {
				if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					valor, _ := strconv.Unquote(lit.Value)
					if conjunto.MatchString(valor) {
						claves = append(claves, valor)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if fallos {
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(claves)
}

// La marca sólo alcanza su línea o la función a cuya declaración está asociada.
func marcasTecnicas(archivo *ast.File, posiciones *token.FileSet) func(ast.Node) bool {
	lineas := map[int]bool{}
	var marcas []token.Pos
	for _, grupo := range archivo.Comments {
		for _, comentario := range grupo.List {
			if strings.TrimSpace(comentario.Text) == "//etf:tecnico" {
				lineas[posiciones.Position(comentario.Pos()).Line] = true
				marcas = append(marcas, comentario.Pos())
			}
		}
	}
	var funciones []*ast.FuncDecl
	for _, declaracion := range archivo.Decls {
		funcion, ok := declaracion.(*ast.FuncDecl)
		if !ok {
			continue
		}
		marcada := false
		limite := funcion.End()
		if funcion.Body != nil && len(funcion.Body.List) > 0 {
			limite = funcion.Body.List[0].Pos()
		}
		for _, marca := range marcas {
			if posiciones.Position(marca).Line == posiciones.Position(funcion.Pos()).Line && marca > funcion.Pos() && marca < limite {
				marcada = true
			}
		}
		if funcion.Doc != nil {
			for _, comentario := range funcion.Doc.List {
				marcada = marcada || lineas[posiciones.Position(comentario.Pos()).Line]
			}
		}
		if marcada {
			funciones = append(funciones, funcion)
		}
	}
	return func(n ast.Node) bool {
		if lineas[posiciones.Position(n.Pos()).Line] || lineas[posiciones.Position(n.End()).Line] {
			return true
		}
		for _, funcion := range funciones {
			if n.Pos() >= funcion.Pos() && n.End() <= funcion.End() {
				return true
			}
		}
		return false
	}
}

func esError(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return id.Name == "errors" && sel.Sel.Name == "New" || id.Name == "fmt" && sel.Sel.Name == "Errorf"
}

// Sigue valores locales al escribir una respuesta: una palabra tampoco puede ir a mano.
func literalesVisibles(expr ast.Expr, vistos map[*ast.Object]bool, informar func(*ast.BasicLit)) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			informar(e)
		}
	case *ast.ParenExpr:
		literalesVisibles(e.X, vistos, informar)
	case *ast.BinaryExpr:
		literalesVisibles(e.X, vistos, informar)
		literalesVisibles(e.Y, vistos, informar)
	case *ast.CallExpr:
		// Conversiones a []byte y formateo; una llamada al puerto Textos ya traduce.
		conversion := false
		if _, ok := e.Fun.(*ast.ArrayType); ok {
			conversion = true
		}
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "fmt" && (sel.Sel.Name == "Sprint" || sel.Sel.Name == "Sprintf") {
				conversion = true
			}
		}
		if conversion {
			for _, arg := range e.Args {
				literalesVisibles(arg, vistos, informar)
			}
		}
	case *ast.Ident:
		if e.Obj == nil || vistos[e.Obj] {
			return
		}
		vistos[e.Obj] = true
		switch d := e.Obj.Decl.(type) {
		case *ast.ValueSpec:
			for i, nombre := range d.Names {
				if nombre.Name == e.Name && i < len(d.Values) {
					literalesVisibles(d.Values[i], vistos, informar)
				}
			}
		case *ast.AssignStmt:
			for i, nombre := range d.Lhs {
				if id, ok := nombre.(*ast.Ident); ok && id.Name == e.Name && i < len(d.Rhs) {
					literalesVisibles(d.Rhs[i], vistos, informar)
				}
			}
		}
	}
}
