package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

func TestHandlers_UpdateMetric(t *testing.T) {
	type fields struct {
		memStorage *storage.MemStorage
	}

	type want struct {
		contentType string
		statusCode  int
	}

	tests := []struct {
		name    string
		fields  fields
		urlPath string
		want    want
	}{
		{
			name:    "test",
			urlPath: "/update/counter/test/1",
			fields: fields{
				memStorage: &storage.MemStorage{},
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handlers{
				memStorage: tt.fields.memStorage,
			}

			request := httptest.NewRequest(http.MethodPost, tt.urlPath, nil)
			w := httptest.NewRecorder()

			h.UpdateMetric(w, request)
		})
	}
}
