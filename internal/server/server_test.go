package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMetricValidatorMW(t *testing.T) {
	srv, err := NewServer()
	assert.NoError(t, err)

	ts := httptest.NewServer(srv.srv.Handler)
	defer ts.Close()

	testCases := []struct {
		name   string
		url    string
		method string
		status int
	}{
		{"ValidMetricCounter", "/update/counter/someCounter/1", http.MethodPost, http.StatusOK},
		{"ValidMetricGauge", "/update/gauge/someGauge/1", http.MethodPost, http.StatusOK},
		{"InvalidMetricType", "/value/invalidType/someGauge", http.MethodGet, http.StatusBadRequest},
		{"NonExistentMetricName", "/update/counter/nonExistent", http.MethodGet, http.StatusNotFound},
		{"EmptyMetricName", "/value/counter/", http.MethodGet, http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, ts.URL+tc.url, nil)
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)

			defer resp.Body.Close()

			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.status, resp.StatusCode)
		})
	}
}

func TestRouter(t *testing.T) {
	strg := storage.NewMemStorage()

	router := newRouter(&routerConfig{
		storage: strg,
		logger:  zap.NewNop(), // disable logging
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	type want struct {
		contentType  string
		statusCode   int
		response     string
		wantResponse bool
	}

	testCases := []struct {
		name   string
		method string
		url    string
		want   want
	}{
		{
			name:   "UpdateCounterMetric",
			method: http.MethodPost,
			url:    "/update/counter/testCounter/1",
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
			},
		},
		{
			name:   "GetCounterMetric",
			method: http.MethodGet,
			url:    "/value/counter/testCounter",
			want: want{
				contentType:  "text/plain",
				statusCode:   http.StatusOK,
				response:     "1",
				wantResponse: true,
			},
		},
		{
			name:   "UpdateCounterMetricBy2",
			method: http.MethodPost,
			url:    "/update/counter/testCounter/2",
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
			},
		},
		{
			name:   "GetUpdatedCounterMetric",
			method: http.MethodGet,
			url:    "/value/counter/testCounter",
			want: want{
				contentType:  "text/plain",
				statusCode:   http.StatusOK,
				response:     "3",
				wantResponse: true,
			},
		},
		{
			name:   "UpdateGaugeMetric",
			method: http.MethodPost,
			url:    "/update/gauge/testGauge/3.140000",
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
			},
		},
		{
			name:   "GetGaugeMetric",
			method: http.MethodGet,
			url:    "/value/gauge/testGauge",
			want: want{
				contentType:  "text/plain",
				statusCode:   http.StatusOK,
				response:     "3.14",
				wantResponse: true,
			},
		},
		{
			name:   "GetNonExistingCounter",
			method: http.MethodGet,
			url:    "/value/counter/NonExistingCounter",
			want: want{
				contentType:  "text/plain; charset=utf-8",
				statusCode:   http.StatusNotFound,
				response:     storage.ErrMetricNotFound.Error() + "\n",
				wantResponse: true,
			},
		},
		{
			name:   "GetNonExistingGauge",
			method: http.MethodGet,
			url:    "/value/gauge/NonExistingGauge",
			want: want{
				contentType:  "text/plain; charset=utf-8",
				statusCode:   http.StatusNotFound,
				response:     storage.ErrMetricNotFound.Error() + "\n",
				wantResponse: true,
			},
		},
		{
			name:   "GetMetricWithInvalidType",
			method: http.MethodGet,
			url:    "/value/invalid/testCounter",
			want: want{
				contentType:  "text/plain; charset=utf-8",
				statusCode:   http.StatusBadRequest,
				response:     errormsg.ErrMetricInvalidType.Error() + "\n",
				wantResponse: true,
			},
		},
		{
			name:   "GetMetricWithEmptyName",
			method: http.MethodGet,
			url:    "/value/counter/",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, ts.URL+tc.url, nil) //nolint:noctx
			require.NoError(t, err)

			req.Header.Set("Accept-Encoding", "")

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)

			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.want.statusCode, resp.StatusCode)

			if tc.want.wantResponse {
				assert.Equal(t, tc.want.response, string(body))
			}
		})
	}
}
