package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_NoDB(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if resp.DB != "disconnected" {
		t.Errorf("expected db=disconnected, got %s", resp.DB)
	}
}

func TestItemsHandler_NoDB(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	w := httptest.NewRecorder()
	itemsHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestItemsHandler_BadMethod(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodDelete, "/items", nil)
	w := httptest.NewRecorder()
	itemsHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestItemsHandler_InvalidBody(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(`{bad}`))
	w := httptest.NewRecorder()
	itemsHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestGetEnv_Default(t *testing.T) {
	val := getEnv("DEFINITELY_NOT_SET_XYZ", "default")
	if val != "default" {
		t.Errorf("expected default, got %s", val)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	db = nil
	handler := loggingMiddleware(healthHandler)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestResponseWriter_DefaultStatus(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		status:         200,
	}
	if rw.status != 200 {
		t.Errorf("expected default status 200, got %d", rw.status)
	}
}