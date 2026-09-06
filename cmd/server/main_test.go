package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestResponseUsesCurrentUTCOnEachRequest(t *testing.T) {
	current := time.Date(2026, 9, 5, 12, 30, 0, 123, time.FixedZone("BRT", -3*3600))
	h := newHandler(func() time.Time { return current }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/projeto-korp", nil))
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || len(body) != 2 || body["nome"] != "Projeto Korp" || body["horario"] != current.UTC().Format(time.RFC3339Nano) {
			t.Fatalf("unexpected response: %s", w.Body.String())
		}
		if w.Header().Get("Content-Type") != "application/json" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing response headers")
		}
		current = current.Add(time.Minute)
	}
}

func TestRoutingAndMetrics(t *testing.T) {
	h := newHandler(time.Now, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/projeto-korp", 200}, {"POST", "/projeto-korp", 405},
		{"GET", "/missing/123", 404}, {"GET", "/healthz", 200}, {"POST", "/healthz", 405}, {"POST", "/metrics", 405},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status {
			t.Fatalf("%s %s: %d", tc.method, tc.path, w.Code)
		}
		if tc.status == 405 && w.Header().Get("Allow") != "GET" {
			t.Fatal("missing Allow")
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	for _, want := range []string{
		`http_requests_total{method="GET",route="/projeto-korp",status="200"} 1`,
		`http_requests_total{method="POST",route="/projeto-korp",status="405"} 1`,
		`http_requests_total{method="GET",route="unmatched",status="404"} 1`,
		`http_request_duration_seconds_count{route="/projeto-korp"} 2`,
	} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("missing metric %s", want)
		}
	}
	if strings.Contains(w.Body.String(), `route="/healthz"`) || strings.Contains(w.Body.String(), "/missing/123") {
		t.Fatal("probe counted or unbounded route label")
	}
}

func TestConcurrentRequests(t *testing.T) {
	h := newHandler(time.Now, slog.New(slog.NewTextHandler(io.Discard, nil)))
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/projeto-korp", nil))
		}()
	}
	wg.Wait()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(w.Body.String(), `http_requests_total{method="GET",route="/projeto-korp",status="200"} 100`) {
		t.Fatal("lost concurrent increments")
	}
}
