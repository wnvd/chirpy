package main

import (
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricHits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) resetMetrics() {
	cfg.fileserverHits.Store(0)
}

func main() {
	mux := http.NewServeMux()
	cfg := apiConfig{}

	// home page
	mux.Handle("/app/", cfg.middlewareMetricHits(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	// metrics
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Values("Content-Type: text/plain; charset=utf-8;")
		w.WriteHeader(http.StatusOK)
		hitCount := cfg.fileserverHits.Load()
		response := fmt.Sprintf("Hits: %v", hitCount)
		w.Write([]byte(response))
	})

	// reset metrics
	mux.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		cfg.resetMetrics()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8;")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Metrics have been reset!"))
	})

	// server logo
	mux.Handle("/app/assets/logo.png", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

	// health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8;")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	fmt.Printf("Server is listening to incomming requests...\n")
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

}
