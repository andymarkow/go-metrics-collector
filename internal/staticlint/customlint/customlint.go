// Package customlint contains custom analyzers.
package customlint

import (
	gocritic "github.com/go-critic/go-critic/checkers/analyzer"
	"github.com/kisielk/errcheck/errcheck"
	"honnef.co/go/tools/unused"

	"golang.org/x/tools/go/analysis"
)

// GetAnalyzers returns a slice of custom analyzers that should be run.
func GetAnalyzers() []*analysis.Analyzer {
	analyzers := make([]*analysis.Analyzer, 0)

	// Adding custom analyzers.
	analyzers = append(analyzers,
		gocritic.Analyzer,        // Adding gocritic analyzer
		errcheck.Analyzer,        // Adding errcheck analyzer
		unused.Analyzer.Analyzer, // Adding unused analyzer
	)

	return analyzers
}
