// Package noexitmain provides noexitmain analyzer.
package noexitmain

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer provides noexitmain analyzer.
var Analyzer = &analysis.Analyzer{ //nolint:gochecknoglobals
	Name: "noexitmain",
	Doc:  "check for os.Exit call in the main function",
	Run:  run,
}

// run checks for the analyzer.
func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// Walk through the file.
		ast.Inspect(file, func(node ast.Node) bool {
			// Find the function declaration.
			if x, ok := node.(*ast.FuncDecl); ok {
				// Check if the function is 'main' and contains os.Exit call.
				if x.Name.String() == "main" && isFuncContainsOSExit(x) {
					pass.Reportf(x.Pos(), "main function should not contain os.Exit call")
				}
			}

			return true
		})
	}

	return nil, nil //nolint:nilnil
}

// isFuncContainsOSExit checks if the given function declaration contains a call to os.Exit.
//
//nolint:nestif
func isFuncContainsOSExit(x *ast.FuncDecl) bool {
	// Walk through the function statements.
	for _, stmt := range x.Body.List {
		// Check if the statement is an expression statement.
		if exp, ok := stmt.(*ast.ExprStmt); ok {
			// Check if the expression is a call expression (function call).
			if call, ok := exp.X.(*ast.CallExpr); ok {
				// Check if this is a selector.
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					// Check if there is an identifier.
					if ident, ok := selector.X.(*ast.Ident); ok &&
						// Check if the function is os.Exit.
						ident.Name == "os" && selector.Sel.Name == "Exit" {
						return true
					}
				}
			}
		}
	}

	return false
}
