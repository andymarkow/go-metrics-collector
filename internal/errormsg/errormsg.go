// Package errormsg provides error messages.
package errormsg

import "errors"

var (
	ErrMetricInvalidType    = errors.New("invalid metric type")
	ErrMetricInvalidDelta   = errors.New("invalid metric delta")
	ErrMetricInvalidValue   = errors.New("invalid metric value")
	ErrMetricEmptyName      = errors.New("empty metric name")
	ErrMetricEmptyValue     = errors.New("empty metric value")
	ErrMetricEmptyDelta     = errors.New("empty metric delta")
	ErrEmptyRequestPayload  = errors.New("empty request payload")
	ErrHashSumValueMismatch = errors.New("hash sum value mismatch")
)
