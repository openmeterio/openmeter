// Command modelaliases generates the public model aliases for generated API transports.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/tools/imports"
)

const legacyModelsImport = "github.com/openmeterio/openmeter/api/models"

func main() {
	client := flag.Bool("client", false, "generate Cloud models and aliases from api/client/go")
	pkg := flag.String("package", "api", "destination server package")
	modelsImport := flag.String("models-import", legacyModelsImport, "shared models package import path")
	flag.Parse()
	if err := generate(*client, *pkg, *modelsImport); err != nil {
		fmt.Fprintln(os.Stderr, "modelaliases:", err)
		os.Exit(1)
	}
}

func generate(client bool, pkg, modelsImport string) error {
	serverPath := "models/models.gen.go"
	if client {
		serverPath = "../../models/models.gen.go"
	}

	shared, err := os.ReadFile(serverPath)
	if err != nil {
		return err
	}

	models, err := declarations(serverPath, shared)
	if err != nil {
		return err
	}

	if client {
		pkg = "openmeter"
		cmd := exec.Command("go", "tool", "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen",
			"--config=../../models/codegen.yaml", "../../openapi.cloud.yaml")
		cmd.Stderr = os.Stderr
		source, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("generate Cloud models: %w", err)
		}

		cloud, err := declarations("openapi.cloud.yaml", source)
		if err != nil {
			return err
		}

		var extra []byte
		extra, models, err = mergeModels(models, cloud)
		if err != nil {
			return err
		}

		if err := os.WriteFile("../../models/cloud.gen.go", extra, 0o644); err != nil {
			return err
		}
	}

	file, err := aliases(pkg, modelsImport, models)
	if err != nil {
		return err
	}

	return os.WriteFile("models.aliases.gen.go", file, 0o644)
}

type declaration struct {
	kind        token.Token
	name        string
	source      string
	methodCount int
	authKey     bool
}

func declarations(filename string, src []byte) ([]declaration, error) {
	f, err := parser.ParseFile(token.NewFileSet(), filename, src, 0)
	if err != nil {
		return nil, err
	}

	var decls []declaration
	methods := make(map[string]int)
	authKeys := make(map[string]bool)
	add := func(kind token.Token, name string, node ast.Node) error {
		var buf bytes.Buffer
		if err := format.Node(&buf, token.NewFileSet(), node); err != nil {
			return err
		}

		decls = append(decls, declaration{kind: kind, name: name, source: buf.String()})

		return nil
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				var name string
				switch s := spec.(type) {
				case *ast.ImportSpec:
					name = s.Path.Value
				case *ast.TypeSpec:
					name = s.Name.Name
				case *ast.ValueSpec:
					if d.Tok != token.CONST || len(s.Names) != 1 || len(s.Values) != 1 {
						return nil, fmt.Errorf("%s: unsupported declaration of %s", filename, s.Names[0])
					}

					name = s.Names[0].Name
					// oapi-codegen emits a typed "<scheme>.Scopes" constant and a
					// private string key for each security scheme.
					key, ok := s.Type.(*ast.Ident)
					value, literal := s.Values[0].(*ast.BasicLit)
					if ok && literal && !ast.IsExported(key.Name) && strings.HasSuffix(key.Name, "ContextKey") &&
						strings.HasSuffix(name, "Scopes") && strings.HasSuffix(value.Value, `.Scopes"`) {
						authKeys[name], authKeys[key.Name] = true, true
					}
				default:
					return nil, fmt.Errorf("%s: unsupported %T", filename, spec)
				}

				if err := add(d.Tok, name, spec); err != nil {
					return nil, err
				}
			}
		case *ast.FuncDecl:
			// Aliases inherit methods. Compare them because two types with the
			// same fields can still have different JSON or union behavior.
			if d.Recv == nil {
				return nil, fmt.Errorf("%s: unexpected model function %s", filename, d.Name)
			}

			receiver := d.Recv.List[0].Type
			if ptr, ok := receiver.(*ast.StarExpr); ok {
				receiver = ptr.X
			}

			id, ok := receiver.(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("%s: unsupported receiver for %s", filename, d.Name)
			}

			if err := add(token.FUNC, id.Name+"."+d.Name.Name, d); err != nil {
				return nil, err
			}

			methods[id.Name]++
		}
	}

	// Comparing each Cloud method catches changed and missing implementations.
	// Count them too: adding MarshalJSON only on the shared type would silently
	// change how the Cloud alias serializes.
	for i, d := range decls {
		decls[i].authKey = authKeys[d.name]
		if d.kind == token.TYPE {
			decls[i].methodCount = methods[d.name]
		}
	}

	return decls, nil
}

// The server keeps its original surface. Cloud gets its own security keys and
// all model exports from either input. Keep Cloud-only definitions in their own
// file so the imports used by their fields and methods stay file-scoped.
func mergeModels(server, cloud []declaration) ([]byte, []declaration, error) {
	shared := make(map[string]declaration, len(server))
	for _, d := range server {
		shared[d.name] = d
	}

	seen := make(map[string]bool, len(cloud))
	var extra bytes.Buffer
	fmt.Fprint(&extra, "// Code generated by modelaliases. DO NOT EDIT.\n\npackage models\n")
	for _, d := range cloud {
		seen[d.name] = true
		if d.authKey {
			continue
		}

		if d.kind == token.IMPORT {
			fmt.Fprintf(&extra, "import %s\n", d.source)
			continue
		}

		if original, ok := shared[d.name]; ok {
			if d != original {
				return nil, nil, fmt.Errorf("openapi.cloud.yaml: %s differs from shared models", d.name)
			}

			continue
		}

		if d.kind == token.FUNC {
			fmt.Fprintln(&extra, d.source)
		} else {
			fmt.Fprintf(&extra, "%s %s\n", d.kind, d.source)
		}
	}

	client := append([]declaration(nil), cloud...)
	for _, d := range server {
		if !seen[d.name] && !d.authKey {
			client = append(client, d)
		}
	}

	// Trim imports used only by declarations already emitted in models.gen.go.
	formatted, err := imports.Process("../../models/cloud.gen.go", extra.Bytes(), nil)

	return formatted, client, err
}

func aliases(pkg, modelsImport string, decls []declaration) ([]byte, error) {
	var out bytes.Buffer
	fmt.Fprintf(&out, "// Code generated by modelaliases. DO NOT EDIT.\n\npackage %s\n\nimport models %q\n\n", pkg, modelsImport)
	for _, d := range decls {
		if pkg == "openmeter" && d.authKey {
			fmt.Fprintf(&out, "%s %s\n", d.kind, d.source)
			continue
		}

		if (d.kind == token.TYPE || d.kind == token.CONST) && ast.IsExported(d.name) {
			fmt.Fprintf(&out, "%s %s = models.%s\n", d.kind, d.name, d.name)
		}
	}

	return format.Source(out.Bytes())
}
