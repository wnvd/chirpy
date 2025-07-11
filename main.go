package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	mux.Handle("/app", cfg.middlewareMetricHits(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	// metrics
	mux.HandleFunc("GET /admin/metrics", cfg.showMetricsHandler)

	// reset metrics
	mux.HandleFunc("POST /admin/reset", cfg.resetMetricHandler)

	// server logo
	mux.Handle("/app/assets/logo.png", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

	// health check
	mux.HandleFunc("GET /api/healthz", checkHealthHandler)

	// validate json
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)

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

// path: /metrics
func (cfg *apiConfig) showMetricsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	hitCount := cfg.fileserverHits.Load()
	response := fmt.Sprintf(`<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>`, hitCount)
	w.Header().Values("Content-Type: text/html; charset=utf-8;")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

// path: /reset
func (cfg *apiConfig) resetMetricHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	cfg.resetMetrics()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8;")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Metrics have been reset!"))
}

// path: /checkhealth
func checkHealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8;")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

const (
	maxMsgLength = 140
)

// path: POST /api/validate_chip
func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type reqObject struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	req := &reqObject{}

	if err := decoder.Decode(req); err != nil {
		log.Printf("Failed to decode request body")
		errResponseHandle(ServerError, w, r)
		return
	}

	if len(req.Body) > maxMsgLength {
		errResponseHandle(Rejected, w, r)
		return
	}

	respBody := struct {
		Valid bool `json:"valid"`
	}{
		Valid: true,
	}
	response, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Failed to marshal response body")
		errResponseHandle(ServerError, w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

// Reponse Types for Error

type ResponseError int

const (
	ServerError ResponseError = iota
	Rejected
)

// helper function to send error response
func errResponseHandle(respType ResponseError, w http.ResponseWriter, r *http.Request) {
	type errorResponse struct {
		Value string `json:"value"`
	}
	errResp := errorResponse{}
	if respType == ServerError {
		w.WriteHeader(http.StatusInternalServerError)
		errResp.Value = "Something went wrong"
	}
	if respType == Rejected {
		w.WriteHeader(http.StatusBadRequest)
		errResp.Value = "Chirp is too long"
	}

	response, err := json.Marshal(errResp)
	if err != nil {
		log.Printf("Failed to marshal json error response!")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	w.Write([]byte(response))
}
