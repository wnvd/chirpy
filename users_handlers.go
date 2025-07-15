package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/wnvd/chirpy/internal/database"
)

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

// helper function to replace profanity
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
