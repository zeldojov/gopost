package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllowedMethod(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		nextCalled     bool
	}{
		{
			name:           "GET allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			nextCalled:     true,
		},
		{
			name:           "POST allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			nextCalled:     true,
		},
		{
			name:           "PUT not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
		{
			name:           "DELETE not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
		{
			name:           "PATCH not allowed",
			method:         http.MethodPatch,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := AllowedMethod(next)

			req := httptest.NewRequest(tt.method, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}

			if nextCalled != tt.nextCalled {
				t.Fatalf(
					"expected nextCalled=%v, got %v",
					tt.nextCalled,
					nextCalled,
				)
			}
		})
	}
}
