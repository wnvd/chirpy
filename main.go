package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/wnvd/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	database       *database.Queries
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
	// loading env vars
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("unable to connect to DB: %v", err)
	}
	dbQueries := database.New(dbConn)

	mux := http.NewServeMux()
	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		database:       dbQueries,
	}

	// -------------------------------------------------
	// ------------------ End Points -------------------
	// -------------------------------------------------

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

	// create a chirp
	mux.HandleFunc("POST /api/chirps", cfg.chirpHandler)

	// get all chirps
	mux.HandleFunc("GET /api/chirps", cfg.getChirpsHandler)

	// get a chirp by Id
	mux.HandleFunc("GET /api/chirps/{chirpId}", cfg.getChirpsByIdHandler)

	// create user
	mux.HandleFunc("POST /api/users", cfg.createUserHandler)

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

// path: /checkhealth
func checkHealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8;")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
