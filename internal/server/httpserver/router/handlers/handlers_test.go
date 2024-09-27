//nolint:noctx
package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
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
func TestGetAllMetricsHandler(t *testing.T) {
	type fields struct {
		storage *storage.MemStorage
	}

	type want struct {
		contentType string
		statusCode  int
	}

	strg := storage.NewMemStorage()

	ctx := context.Background()

	err := strg.SetCounter(ctx, "testCounter", 1)
	require.NoError(t, err)

	err = strg.SetGauge(ctx, "testGauge", 3.14)
	require.NoError(t, err)

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
				contentType: "text/html",
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

			defer func() {
				require.NoError(t, resp.Body.Close())
			}()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.NotEqual(t, "", string(body))
		})
	}
}

// TestGetMetric tests the GetMetric handler.
func TestGetMetricHandler(t *testing.T) {
	type want struct {
		contentType string
		response    string
		statusCode  int
	}

	strg := storage.NewMemStorage()

	ctx := context.Background()

	err := strg.SetCounter(ctx, "testCounter", 1)
	require.NoError(t, err)

	err = strg.SetGauge(ctx, "testGauge", 3.14)
	require.NoError(t, err)

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
				response:    errormsg.ErrMetricInvalidType.Error() + "\n",
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
			defer func() {
				require.NoError(t, resp.Body.Close())
			}()

			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.want.statusCode, resp.StatusCode)
		})
	}
}

// TestUpdateMetric tests the UpdateMetric handler.
func TestUpdateMetricHandler(t *testing.T) {
	type want struct {
		contentType string
		response    string
		statusCode  int
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
				response:    errormsg.ErrMetricInvalidType.Error() + "\n",
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
				response:    errormsg.ErrMetricEmptyValue.Error() + "\n",
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
				response:    errormsg.ErrMetricInvalidType.Error() + "\n",
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
				response:    errormsg.ErrMetricInvalidValue.Error() + "\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := newChiHTTPRequest(http.MethodPost, "/update/{metricType}/{metricName}/{metricValue}", map[string]string{
				"metricName":  tc.metric.name,
				"metricType":  tc.metric.kind,
				"metricValue": tc.metric.value,
			}, nil)

			w := httptest.NewRecorder()

			h.UpdateMetric(w, req)

			resp := w.Result()
			assert.Equal(t, tc.want.statusCode, resp.StatusCode)
			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))

			defer func() {
				require.NoError(t, resp.Body.Close())
			}()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.want.response, string(body))
		})
	}
}

// TestGetMetricJSONHandler tests the GetMetricJSON handler.
func TestGetMetricJSONHandler(t *testing.T) {
	type want struct {
		contentType string
		response    string
		statusCode  int
	}

	strg := storage.NewMemStorage()

	ctx := context.Background()

	err := strg.SetCounter(ctx, "testCounter", 1)
	require.NoError(t, err)

	err = strg.SetGauge(ctx, "testGauge", 3.14)
	require.NoError(t, err)

	h := NewHandlers(strg)

	testCases := []struct {
		name string
		body string
		want want
	}{
		{
			name: "GetCounterMetric",
			body: `{"id": "testCounter", "type": "counter"}`,
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				response:    `{"id": "testCounter", "type": "counter", "delta": 1}`,
			},
		},
		{
			name: "GetGaugeMetric",
			body: `{"id": "testGauge", "type": "gauge"}`,
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				response:    `{"id": "testGauge", "type": "gauge", "value": 3.14}`,
			},
		},
		{
			name: "EmptyRequestPayload",
			body: "",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "EmptyMetricName",
			body: `{"id": "", "type": "counter"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "EmptyMetricType",
			body: `{"id": "testCounter", "type": ""}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "NonExistingCounterMetric",
			body: `{"id": "nonexistingCounter", "type": "counter"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				response:    "",
			},
		},
		{
			name: "NonExistingGaugeMetric",
			body: `{"id": "nonexistingGauge", "type": "gauge"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				response:    "",
			},
		},
		{
			name: "InvalidMetricType",
			body: `{"id": "testGauge", "type": "invalid"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "InvalidJSONPayload",
			body: `{"id": "testGauge", "type": "counter}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusInternalServerError,
				response:    "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := newChiHTTPRequest(http.MethodPost, "/value", nil, strings.NewReader(tc.body))

			w := httptest.NewRecorder()

			h.GetMetricJSON(w, req)

			resp := w.Result()
			defer func() {
				require.NoError(t, resp.Body.Close())
			}()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.want.statusCode, resp.StatusCode)

			if tc.want.response != "" {
				assert.JSONEq(t, tc.want.response, string(body))
			}
		})
	}
}

// TestUpdateMetricJSONHandler tests the UpdateMetricJSON handler.
func TestUpdateMetricJSONHandler(t *testing.T) {
	type want struct {
		contentType string
		response    string
		statusCode  int
	}

	strg := storage.NewMemStorage()

	h := NewHandlers(strg)

	testCases := []struct {
		name string
		body string
		want want
	}{
		{
			name: "UpdateCounterMetric",
			body: `{"id": "testCounter", "type": "counter", "delta": 1}`,
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				response:    `{"id": "testCounter", "type": "counter", "delta": 1}`,
			},
		},
		{
			name: "UpdateGaugeMetric",
			body: `{"id": "testGauge", "type": "gauge", "value": 3.14}`,
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				response:    `{"id": "testGauge", "type": "gauge", "value": 3.14}`,
			},
		},
		{
			name: "EmptyRequestPayload",
			body: "",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "EmptyMetricName",
			body: `{"id": "", "type": "gauge", "value": 3.14}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "EmptyMetricType",
			body: `{"id": "testCounter", "type": "", "delta": 1}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "EmptyCounterDelta",
			body: `{"id": "testCounter", "type": "counter"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "EmptyGaugeValue",
			body: `{"id": "testGauge", "type": "gauge"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "InvalidMetricType",
			body: `{"id": "testGauge", "type": "invalid", "value": 3.14}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "InvalidCounterDelta",
			body: `{"id": "testCounter", "type": "counter", "delta": "1"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "InvalidGaugeValue",
			body: `{"id": "testGauge", "type": "gauge", "value": "3.14"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
		{
			name: "InvalidJSONPayload",
			body: `{"id": "testGauge", "type": "gauge", "value": "3.14}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := newChiHTTPRequest(http.MethodPost, "/update", nil, strings.NewReader(tc.body))

			w := httptest.NewRecorder()

			h.UpdateMetricJSON(w, req)

			resp := w.Result()
			defer func() {
				require.NoError(t, resp.Body.Close())
			}()

			assert.Equal(t, tc.want.statusCode, resp.StatusCode)
			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tc.want.response != "" {
				assert.JSONEq(t, tc.want.response, string(body))
			}
		})
	}
}
