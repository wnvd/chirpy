package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/wnvd/chirpy/internal/auth"
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
func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {
	type reqObject struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	req := &reqObject{}

	if err := decoder.Decode(req); err != nil {
		log.Printf("Failed to decode request body")
		ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	// Get the jwt token from the header
	jwtToken, err := auth.GetAuthToken(r.Header)
	if err != nil {
		log.Printf("Unable to get bearer token: %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Bad Request")
		return
	}

	// validating the jwt token the getting the user UUID
	userUUID, err := auth.ValidateJWT(jwtToken, cfg.jwtSecret)
	if err != nil {
		log.Printf("Unable to validate token: %v", err)
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized Request")
		return
	}

	if len(req.Body) > maxMsgLength {
		ErrorResponse(w, http.StatusBadRequest, "chirp body length exceed limit")
		return
	}

	// check if user with request uuid exist
	_, err = cfg.database.GetUserById(r.Context(), userUUID)
	if err != nil {
		log.Printf("Failed to user by ID %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
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
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// chirp := Chirp{
	// 	Id:        createdChirp.ID,
	// 	CreatedAt: createdChirp.CreatedAt,
	// 	UpdatedAt: createdChirp.UpdatedAt,
	// 	Body:      createdChirp.Body,
	// 	UserId:    createdChirp.UserID,
	// }

	// response, err := json.Marshal(chirp)
	// if err != nil {
	// 	log.Printf("Failed to marshal response body %v", err)
	// 	ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
	// 	return
	// }
	// w.WriteHeader(http.StatusCreated)
	// w.Write([]byte(response))

	JSONResponse(w, http.StatusCreated, Chirp{
		Id:        createdChirp.ID,
		CreatedAt: createdChirp.CreatedAt,
		UpdatedAt: createdChirp.UpdatedAt,
		Body:      createdChirp.Body,
		UserId:    createdChirp.UserID,
	})
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

	fmt.Println("actually here")
	chirps, err := c.database.GetAllChirps(r.Context())
	if err != nil {
		log.Printf("Failed to all the chirps from the database")
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if len(chirps) == 0 {
		log.Printf("No chirps in the database")
		ErrorResponse(w, http.StatusNotFound, "No Chirps available")
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
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)
	// w.Write(data)

	fmt.Println(respChirp)
	fmt.Println(string(data))
	JSONResponse(w, http.StatusOK, respChirp)
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
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	chirp, err := cfg.database.GetChirpById(r.Context(), chirpId)
	if err != nil {
		log.Printf("Unable to get chirp data by Id: %v", err)
		ErrorResponse(w, http.StatusNotFound, "Invalid chirp Id")
		return
	}

	// response, err := json.Marshal(
	// 	Chirp{
	// 		Id:        chirp.ID,
	// 		CreatedAt: chirp.CreatedAt,
	// 		UpdatedAt: chirp.UpdatedAt,
	// 		Body:      chirp.Body,
	// 		UserId:    chirp.UserID,
	// 	})
	//
	// if err != nil {
	// 	log.Printf("Failed to marshal response body %v", err)
	// 	ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
	// 	return
	// }
	//
	// w.WriteHeader(http.StatusOK)
	// w.Write([]byte(response))
	JSONResponse(w, http.StatusOK, Chirp{
		Id:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID,
	})
}

/*
* Deletes chirp by Id
*
* path : DELETE /api/chirps/{id}
*
 */

func (cfg *apiConfig) deleteChirpsByIdHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	token, err := auth.GetAuthToken(r.Header)
	if err != nil {
		log.Printf("Unable to get bearer token: %v", err)
		ErrorResponse(w, http.StatusUnauthorized, "Internal Server Error")
		return
	}

	// getting user uuid
	userUUID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("Unable to validate jwt token: %v", err)
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	// getting chirp id provided in client url
	chirpId, err := uuid.Parse(r.PathValue("chirpId"))
	if err != nil {
		log.Printf("Unable to parse uuid: %v", err)
		ErrorResponse(w, http.StatusUnauthorized, "Internal Server Error")
		return
	}

	/*
	 * we have to do a lot of checking and send different error codes back,
	 * first check if the user, chirp exist and then delete the chirp with
	 * related to that user.
	 *
	 */

	chirp, err := cfg.database.GetChirpById(r.Context(), chirpId)
	if err != nil {
		log.Printf("Unable to get chirp details from the database : %v", err)
		ErrorResponse(w, http.StatusNotFound, "Not Found")
		return
	}

	// checking if user ID provided != chirp author ID
	if userUUID != chirp.UserID {
		log.Printf("chirp author id and user id are equal")
		ErrorResponse(w, http.StatusForbidden, "Forbidden")
		return
	}

	if err := cfg.database.DeleteChirpById(r.Context(), database.DeleteChirpByIdParams{
		ID:     chirp.ID,
		UserID: chirp.UserID,
	}); err != nil {
		log.Printf("Unable to get chirp details from the database : %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Interval Server Error")
		return
	}

	JSONResponse(w, http.StatusNoContent, nil)
}
