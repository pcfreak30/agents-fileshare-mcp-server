package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestServerError(t *testing.T) {
	w := httptest.NewRecorder()
	ServerError(w, zap.NewNop(), "internal error", "test log", errTest{})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	body := w.Body.String()
	if body != "internal error\n" {
		t.Errorf("body = %q, want %q", body, "internal error\n")
	}
}

type errTest struct{}

func (errTest) Error() string { return "test error" }
