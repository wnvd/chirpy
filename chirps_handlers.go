package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/wnvd/chirpy/internal/database"
)

type Chirp struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}

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

/*
* Gets all chips from the database
* by defualt in ascending order
*
* path : GET /api/chirps
*
 */
func (c *apiConfig) getChirpsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	chirps, err := c.database.GetAllChirps(r.Context())
	if err != nil {
		log.Printf("Failed to all the chirps from the database")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	respChirp := make([]Chirp, len(chirps))

	for i, chirp := range chirps {
		respChirp[i] = Chirp{
			Id:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserId:    chirp.UserID,
		}
	}

	data, err := json.Marshal(respChirp)
	if err != nil {
		log.Printf("Failed to all the chirps from the database")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

/*
* Gets all chips from the database
* by defualt in ascending order
*
* path : GET /api/chirps/{id}
*
 */
func (cfg *apiConfig) getChirpsByIdHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	chirpId, err := uuid.Parse(r.PathValue("chirpId"))
	if err != nil {
		log.Printf("Unable to parse uuid: %v", err)
		errResponseHandle(ServerError, "Server Error", w, r)
		return
	}

	chirp, err := cfg.database.GetChirpById(r.Context(), chirpId)
	if err != nil {
		log.Printf("Unable to get chirp data by Id: %v", err)
		errResponseHandle(NotFound, "Invalid chirp id", w, r)
		return
	}

	response, err := json.Marshal(
		Chirp{
			Id:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserId:    chirp.UserID,
		})

	if err != nil {
		log.Printf("Failed to marshal response body %v", err)
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}
