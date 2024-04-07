package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andymarkow/go-metrics-collector/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestParseGaugeMetricValue(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"ValidInput", "3.14", 3.14, false},
		{"InvalidInput", "invalid", 0.0, true},
		{"EmptyInput", "", 0.0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			result, err := parseGaugeMetricValue(tc.input)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Equal(t, 0.0, result)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestHandlers_UpdateMetric(t *testing.T) {
	type fields struct {
		storage *storage.MemStorage
	}

	type want struct {
		contentType string
		statusCode  int
	}

	tests := []struct {
		name   string
		fields fields
		url    string
		want   want
	}{
		{
			name: "test",
			url:  "/update/counter/test/1",
			fields: fields{
				storage: &storage.MemStorage{},
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(_ *testing.T) {
			h := &Handlers{
				storage: tc.fields.storage,
			}

			request := httptest.NewRequest(http.MethodPost, tc.url, nil)
			w := httptest.NewRecorder()

			h.UpdateMetric(w, request)
		})
	}
}
