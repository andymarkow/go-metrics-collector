// Package staticlint provides staticlint analyzers.
package staticlint

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/andymarkow/go-metrics-collector/internal/staticlint/analysislint"
	"github.com/andymarkow/go-metrics-collector/internal/staticlint/customlint"
	"github.com/andymarkow/go-metrics-collector/internal/staticlint/noexitmain"
	"github.com/andymarkow/go-metrics-collector/internal/staticlint/staticchecklint"
)

// Staticlint contains all the analyzers registered in this package.
type Staticlint struct {
	Analyzers []*analysis.Analyzer
}

// NewStaticlint constructs a new Staticlint.
func NewStaticlint() *Staticlint {
	analyzers := make([]*analysis.Analyzer, 0)

	// Add analysis/passes analizers.
	analyzers = append(analyzers, analysislint.GetAnalyzers()...)

	// Add staticchecklint analyzers.
	analyzers = append(analyzers, staticchecklint.GetAnalyzers()...)

	// Add customlint analyzers.
	analyzers = append(analyzers, customlint.GetAnalyzers()...)

	// Add noexitmain analyzer.
	analyzers = append(analyzers, noexitmain.Analyzer)

	return &Staticlint{
		Analyzers: analyzers,
	}
}

// Run runs all the analyzers registered in s.Analyzers and outputs the results.
func (s *Staticlint) Run() {
	multichecker.Main(s.Analyzers...)
}
