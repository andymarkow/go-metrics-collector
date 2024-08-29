package models

import (
	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
)

// Metrics is a model for metrics.
type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

// Validate performs basic validation of the Metrics object.
// It checks that the ID field is not empty and that the MType field
// is either "counter" or "gauge". If either of these conditions are
// not met, an error will be returned.
func (m *Metrics) Validate() error {
	if m.ID == "" {
		return errormsg.ErrMetricEmptyName
	}

	switch m.MType {
	case "counter", "gauge":
	default:
		return errormsg.ErrMetricInvalidType
	}

	return nil
}

// ValidateUpdate performs basic validation of the Metrics object, but with
// the logic of Delta and Value switched. It checks that the ID field is not
// empty and that the MType field is either "counter" or "gauge". If either of
// these conditions are not met, an error will be returned.
func (m *Metrics) ValidateUpdate() error {
	if m.ID == "" {
		return errormsg.ErrMetricEmptyName
	}

	switch m.MType {
	case "counter":
		if m.Delta == nil {
			return errormsg.ErrMetricEmptyDelta
		}

	case "gauge":
		if m.Value == nil {
			return errormsg.ErrMetricEmptyValue
		}

	default:
		return errormsg.ErrMetricInvalidType
	}

	return nil
}
