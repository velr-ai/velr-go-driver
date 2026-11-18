package velr

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestPublicAPIDocumentation(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		check := func(name string, documented bool) {
			if ast.IsExported(name) && !documented {
				t.Errorf("%s: exported API %s needs caller documentation", entry.Name(), name)
			}
		}
		for _, declaration := range file.Decls {
			switch node := declaration.(type) {
			case *ast.FuncDecl:
				check(node.Name.Name, node.Doc != nil)
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						check(spec.Name.Name, spec.Doc != nil || node.Doc != nil)
						if fields, ok := spec.Type.(*ast.StructType); ok && ast.IsExported(spec.Name.Name) {
							for _, field := range fields.Fields.List {
								for _, name := range field.Names {
									check(name.Name, field.Doc != nil || field.Comment != nil)
								}
							}
						}
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							check(name.Name, spec.Doc != nil || spec.Comment != nil || node.Doc != nil)
						}
					}
				}
			}
		}
	}
}
