package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type response struct {
	Nome    string `json:"nome"`
	Horario string `json:"horario"`
}

func newHandler(now func() time.Time, logger *slog.Logger) http.Handler {
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total", Help: "Application requests, excluding health and metrics probes.",
	}, []string{"method", "route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds", Help: "Application response duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"})
	registry.MustRegister(requests, duration, collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	requests.WithLabelValues("GET", "/projeto-korp", "200").Add(0)
	metrics := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || r.URL.Path == "/healthz" {
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", "GET")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if r.URL.Path == "/metrics" {
				metrics.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
			return
		}
		start := time.Now()
		route, status := "unmatched", http.StatusNotFound
		if r.URL.Path == "/projeto-korp" {
			route = "/projeto-korp"
			if r.Method != http.MethodGet {
				status = http.StatusMethodNotAllowed
				w.Header().Set("Allow", "GET")
				http.Error(w, "method not allowed", status)
			} else {
				status = http.StatusOK
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "no-store")
				if err := json.NewEncoder(w).Encode(response{"Projeto Korp", now().UTC().Format(time.RFC3339Nano)}); err != nil {
					logger.Error("write response", "error", err)
				}
			}
		} else {
			http.NotFound(w, r)
		}
		method := r.Method
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		default:
			method = "OTHER"
		}
		requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
		duration.WithLabelValues(route).Observe(time.Since(start).Seconds())
		logger.Info("request", "method", method, "route", route, "status", status, "duration_ms", time.Since(start).Milliseconds())
	})
}

func run() error {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://127.0.0.1:8080/healthz")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("health status: %d", resp.StatusCode)
		}
		return nil
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server := &http.Server{Addr: ":8080", Handler: newHandler(time.Now, logger),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errorsCh := make(chan error, 1)
	go func() { logger.Info("server started", "address", server.Addr); errorsCh <- server.ListenAndServe() }()
	select {
	case err := <-errorsCh:
		return err
	case <-ctx.Done():
		logger.Info("server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func main() {
	if err := run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
