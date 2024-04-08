//nolint:noctx
package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andymarkow/go-metrics-collector/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newChiHTTPRequest(method, url string, urlParams map[string]string, body io.Reader) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range urlParams {
		rctx.URLParams.Add(k, v)
	}

	req := httptest.NewRequest(method, url, body)

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// TestParseGaugeMetricValue tests the parseGaugeMetricValue function.
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

// TestGetAllMetrics tests the GetAllMetrics handler.
func TestGetAllMetrics(t *testing.T) {
	type fields struct {
		storage *storage.MemStorage
	}

	type want struct {
		contentType string
		statusCode  int
	}

	strg := storage.NewMemStorage()
	strg.SetCounter("testCounter", 1)
	strg.SetGauge("testGauge", 3.14)

	testCases := []struct {
		name   string
		url    string
		fields fields
		want   want
	}{
		{
			name: "GetAllMetrics",
			url:  "/",
			fields: fields{
				storage: strg,
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := &Handlers{
				storage: tc.fields.storage,
			}

			req, err := http.NewRequest(http.MethodGet, tc.url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			h.GetAllMetrics(w, req)

			resp := w.Result()

			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.want.statusCode, resp.StatusCode)

			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.NotEqual(t, "", string(body))
		})
	}
}

// TestGetMetric tests the GetMetric handler.
func TestGetMetric(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		response    string
	}

	strg := storage.NewMemStorage()
	strg.SetCounter("testCounter", 1)
	strg.SetGauge("testGauge", 3.14)

	h := NewHandlers(strg)

	testCases := []struct {
		name       string
		metricType string
		metricName string
		want       want
	}{
		{
			name:       "GetCounterMetric",
			metricType: "counter",
			metricName: "testCounter",
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
				response:    "1",
			},
		},
		{
			name:       "GetGaugeMetric",
			metricType: "gauge",
			metricName: "testGauge",
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
				response:    "3.14",
			},
		},
		{
			name:       "EmptyMetricName",
			metricType: "counter",
			metricName: "",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				response:    storage.ErrMetricNotFound.Error() + "\n",
			},
		},
		{
			name:       "NonExistingCounterMetric",
			metricType: "counter",
			metricName: "nonexistingCounter",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				response:    storage.ErrMetricNotFound.Error() + "\n",
			},
		},
		{
			name:       "NonExistingGaugeMetric",
			metricType: "gauge",
			metricName: "nonexistingGauge",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				response:    storage.ErrMetricNotFound.Error() + "\n",
			},
		},
		{
			name:       "InvalidMetricType",
			metricType: "invalid",
			metricName: "testCounter",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    ErrMetricInvalidType.Error() + "\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := newChiHTTPRequest(http.MethodGet, "/value/{metricType}/{metricName}", map[string]string{
				"metricName": tc.metricName,
				"metricType": tc.metricType,
			}, nil)

			w := httptest.NewRecorder()

			h.GetMetric(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.want.statusCode, resp.StatusCode)
		})
	}
}

// TestUpdateMetric tests the UpdateMetric handler.
func TestUpdateMetric(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		response    string
	}

	type metric struct {
		name  string
		kind  string
		value string
	}

	strg := storage.NewMemStorage()

	h := NewHandlers(strg)

	testCases := []struct {
		name   string
		metric metric
		want   want
	}{
		{
			name: "UpdateCounterMetric",
			metric: metric{
				name:  "testCounter",
				kind:  "counter",
				value: "1",
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
				response:    "OK",
			},
		},
		{
			name: "UpdateGaugeMetric",
			metric: metric{
				name:  "testGauge",
				kind:  "gauge",
				value: "3.14",
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
				response:    "OK",
			},
		},
		{
			name: "EmptyMetricType",
			metric: metric{
				name:  "testCounter",
				kind:  "",
				value: "1",
			},
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    ErrMetricInvalidType.Error() + "\n",
			},
		},
		{
			name: "EmptyMetricValue",
			metric: metric{
				name:  "testCounter",
				kind:  "counter",
				value: "",
			},
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    ErrMetricEmptyValue.Error() + "\n",
			},
		},
		{
			name: "InvalidMetricType",
			metric: metric{
				name:  "testCounter",
				kind:  "invalid",
				value: "3.14",
			},
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    ErrMetricInvalidType.Error() + "\n",
			},
		},
		{
			name: "InvalidMetricValue",
			metric: metric{
				name:  "testGauge",
				kind:  "gauge",
				value: "3.14e",
			},
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    ErrMetricInvalidValue.Error() + "\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := newChiHTTPRequest(http.MethodPost, "/{metricType}/{metricName}/{metricValue}", map[string]string{
				"metricName":  tc.metric.name,
				"metricType":  tc.metric.kind,
				"metricValue": tc.metric.value,
			}, nil)

			w := httptest.NewRecorder()

			h.UpdateMetric(w, req)

			resp := w.Result()
			assert.Equal(t, tc.want.statusCode, resp.StatusCode)
			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))

			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.want.response, string(body))
		})
	}
}
