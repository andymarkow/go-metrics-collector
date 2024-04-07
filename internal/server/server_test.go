package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{"ValidMetricCounter", "/update/counter/someCounter/1", "POST", http.StatusOK},
		{"ValidMetricGauge", "/update/gauge/someGauge/1", "POST", http.StatusOK},
		{"InvalidMetricType", "/value/invalidType/someGauge", "GET", http.StatusBadRequest},
		{"NonExistentMetricName", "/update/counter/nonExistent", "GET", http.StatusNotFound},
		{"EmptyMetricName", "/value/counter/", "GET", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, ts.URL+tc.url, nil) //nolint:noctx
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
