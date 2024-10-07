// Package models provides models for HTTP requests.
package models

import (
	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/monitor/metrics"
)

// Metrics is a model for metrics.
type Metrics struct {
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
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
	case string(metrics.MetricCounter), string(metrics.MetricGauge):
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
	case string(metrics.MetricCounter):
		if m.Delta == nil {
			return errormsg.ErrMetricEmptyDelta
		}

	case string(metrics.MetricGauge):
		if m.Value == nil {
			return errormsg.ErrMetricEmptyValue
		}

	default:
		return errormsg.ErrMetricInvalidType
	}

	return nil
}

func NewMetrics(id, mType string, delta *int64, value *float64) (Metrics, error) {
	if id == "" {
		return Metrics{}, errormsg.ErrMetricEmptyName
	}

	switch mType {
	case string(metrics.MetricCounter):
		if delta == nil {
			return Metrics{}, errormsg.ErrMetricEmptyDelta
		}
	case string(metrics.MetricGauge):
		if value == nil {
			return Metrics{}, errormsg.ErrMetricEmptyValue
		}
	default:
		return Metrics{}, errormsg.ErrMetricInvalidType
	}

	return Metrics{
		ID:    id,
		MType: mType,
		Delta: delta,
		Value: value,
	}, nil
}
