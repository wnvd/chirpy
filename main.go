package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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
		log.Printf("unable to connect to DB: %v")
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

	// validate json
	mux.HandleFunc("POST /api/chirps", cfg.chirpHandler)

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
	if os.Getenv("PLATFROM") == "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(http.StatusText(http.StatusForbidden)))
		return
	}

	cfg.resetMetrics()
	// delete all users from the database
	cfg.database.DeleteAllUsers(r.Context())
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

// path: POST /api/chirp
func (cfg *apiConfig) chirpHandler(w http.ResponseWriter, r *http.Request) {
	type reqObject struct {
		Body   string `json:"body"`
		UserID string `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	req := &reqObject{}

	if err := decoder.Decode(req); err != nil {
		log.Printf("Failed to decode request body")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	if len(req.Body) > maxMsgLength {
		errResponseHandle(Rejected, "Chirp too long", w, r)
		return
	}

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		log.Printf("Failed to parse uuid %v", err)
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	// check if user with request uuid exist
	_, err = cfg.database.GetUserById(r.Context(), userUUID)
	if err != nil {
		log.Printf("Failed to user by ID %v", err)
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	// running through profanity filter
	cleaned_body := replaceProfane(req.Body)
	chirpParams := database.CreateChirpParams{
		ID:     uuid.New(),
		Body:   cleaned_body,
		UserID: userUUID,
	}

	createdChirp, err := cfg.database.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		log.Printf("Failed to create chirp %v", err)
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	type Chirp struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}

	chirp := Chirp{
		Id:        createdChirp.ID,
		CreatedAt: createdChirp.CreatedAt,
		UpdatedAt: createdChirp.UpdatedAt,
		Body:      createdChirp.Body,
		UserId:    createdChirp.UserID,
	}

	response, err := json.Marshal(chirp)
	if err != nil {
		log.Printf("Failed to marshal response body %v", err)
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(response))
}

// Reponse Types for Error

type ResponseError int

const (
	ServerError ResponseError = iota
	Rejected
)

// TODO: you probably need to split it into
// error response and json response
// helper function to send error response
func errResponseHandle(respType ResponseError, respMsg string, w http.ResponseWriter, r *http.Request) {
	type errorResponse struct {
		Value string `json:"value"`
	}
	errResp := errorResponse{}
	if respType == ServerError {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if respType == Rejected {
		w.WriteHeader(http.StatusBadRequest)
	}
	errResp.Value = respMsg

	response, err := json.Marshal(errResp)
	if err != nil {
		log.Printf("Failed to marshal json error response!")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	w.Write([]byte(response))
}

// helper function to replace profane
func replaceProfane(body string) string {
	wordFilter := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	filtered_words := make([]string, 0)
	for word := range strings.SplitSeq(body, " ") {
		// converting word to lower case for lookup
		// and only using the original word casing
		wordLowerCase := strings.ToLower(word)
		if _, exist := wordFilter[wordLowerCase]; exist {
			filtered_words = append(filtered_words, "****")
			continue
		}
		filtered_words = append(filtered_words, word)
	}

	new_body := strings.Join(filtered_words, " ")

	return new_body
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) createUserHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	type reqObject struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	req := &reqObject{}
	if err := decoder.Decode(req); err != nil {
		log.Printf("Failed to decode request body")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	if !strings.Contains(req.Email, "@") {
		log.Printf("Invalid Email")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	userParams := database.CreateUserParams{
		ID:    uuid.New(),
		Email: req.Email,
	}

	user, err := cfg.database.CreateUser(r.Context(), userParams)
	if err != nil {
		log.Printf("Failed to create User %v", err)
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	responseBody := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	response, err := json.Marshal(responseBody)
	if err != nil {
		log.Printf("Failed to decode request body")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(response))
}
