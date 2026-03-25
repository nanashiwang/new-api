package common

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

var disallowedJSONCalls = []string{
	"Marshal",
	"MarshalIndent",
	"NewDecoder",
	"NewEncoder",
	"Unmarshal",
	"Valid",
}

func TestCoreLayersAvoidDirectEncodingJSONCalls(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	repoRoot := filepath.Dir(filepath.Dir(filename))
	dirs := []string{"controller", "service", "model"}

	for _, dir := range dirs {
		root := filepath.Join(repoRoot, dir)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if filepath.Base(path) == "json_guard_test.go" {
				return nil
			}
			checkCoreJSONFile(t, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}

func checkCoreJSONFile(t *testing.T, path string) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	jsonImportNames := make(map[string]struct{})
	for _, imp := range file.Imports {
		if imp.Path == nil || imp.Path.Value != "\"encoding/json\"" {
			continue
		}
		name := "json"
		if imp.Name != nil && imp.Name.Name != "." && imp.Name.Name != "_" {
			name = imp.Name.Name
		}
		jsonImportNames[name] = struct{}{}
	}
	if len(jsonImportNames) == 0 {
		return
	}

	ast.Inspect(file, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if _, ok := jsonImportNames[pkg.Name]; !ok {
			return true
		}
		if slices.Contains(disallowedJSONCalls, sel.Sel.Name) {
			pos := fset.Position(sel.Pos())
			t.Errorf("%s uses encoding/json.%s directly; use common/json.go wrapper instead", pos, sel.Sel.Name)
		}
		return true
	})
}
