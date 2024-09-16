// Package staticchecklint provides staticcheck.io analyzers.
package staticchecklint

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/staticcheck"
)

// GetAnalyzers returns a slice of staticcheck analyzers that should be run.
//
// All checks can be found here: https://staticcheck.dev/docs/checks/
func GetAnalyzers() []*analysis.Analyzer {
	analyzers := make([]*analysis.Analyzer, 0)

	// Adding staticcheck analyzers.
	for _, v := range staticcheck.Analyzers {
		// Adding all checks starting with 'SA*'.
		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			analyzers = append(analyzers, v.Analyzer)
		}

		// Adding all checks starting with 'ST*'.
		if strings.HasPrefix(v.Analyzer.Name, "ST") {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	return analyzers
}
