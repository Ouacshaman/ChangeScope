// Package parser extracts changed Go symbols from source files using AST analysis.
package parser

import (
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/Ouacshaman/changescope/internal/differ"
)

// Symbol represents a Go function or method that was identified as changed.
type Symbol struct {
	Name     string // function or method name, e.g. "UpdateTask"
	Receiver string // receiver type name, e.g. "TaskService"; empty for package-level funcs
	File     string // file path within the repository
}

// Extract parses the Go source file at filePath and returns all FuncDecl
// nodes whose line range intersects at least one of the provided changed ranges.
// It uses the standard go/parser and go/ast packages — no regex.
func Extract(filePath string, ranges []differ.LineRange) ([]Symbol, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		funcStart := fset.Position(fn.Pos()).Line
		funcEnd := fset.Position(fn.End()).Line

		if !intersects(funcStart, funcEnd, ranges) {
			continue
		}

		sym := Symbol{
			Name: fn.Name.Name,
			File: filePath,
		}

		// Extract the receiver type name if present.
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			sym.Receiver = receiverTypeName(fn.Recv.List[0].Type)
		}

		symbols = append(symbols, sym)
	}

	return symbols, nil
}

// intersects reports whether [funcStart, funcEnd] overlaps any range in ranges.
func intersects(funcStart, funcEnd int, ranges []differ.LineRange) bool {
	for _, r := range ranges {
		if funcEnd >= r.Start && funcStart <= r.End {
			return true
		}
	}
	return false
}

// receiverTypeName extracts the type name from a receiver field expression,
// handling both value receivers (T) and pointer receivers (*T).
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}
