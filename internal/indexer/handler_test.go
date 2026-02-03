package indexer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mouse/internal/logging"
)

func TestHandlerRequiresQuery(t *testing.T) {
	h := NewHandler(nil, logging.New("test"))
	req := httptest.NewRequest(http.MethodGet, "/index/search", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	h = NewHandler(&Indexer{}, logging.New("test"))
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/index/search", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestReindexHandlerRequiresIndexer(t *testing.T) {
	h := NewReindexHandler(nil, logging.New("test"))
	req := httptest.NewRequest(http.MethodPost, "/index/reindex", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
