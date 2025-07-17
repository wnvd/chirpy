package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/wnvd/chirpy/internal/auth"
	"github.com/wnvd/chirpy/internal/database"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type UserLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cfg *apiConfig) createUserHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	decoder := json.NewDecoder(r.Body)
	req := &UserLogin{}
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

	if len(req.Password) < 4 {
		log.Printf("Invalid Password")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Unable to hash password, %v", err)
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	userParams := database.CreateUserParams{
		ID:             uuid.New(),
		Email:          req.Email,
		HashedPassword: hashedPassword,
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

func (cfg *apiConfig) userLoginHandler(
	w http.ResponseWriter,
	r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	req := &UserLogin{}
	if err := decoder.Decode(req); err != nil {
		log.Printf("Failed to decode request body")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}

	if !strings.Contains(req.Email, "@") {
		log.Printf("Invalid Email")
		errResponseHandle(Rejected, "Not a valid email", w, r)
		return
	}

	if len(req.Password) < 4 {
		log.Printf("Invalid Password")
		errResponseHandle(Rejected, "Not a valid password", w, r)
		return
	}

	user, err := cfg.database.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		log.Printf("Unable to get user from the database by Id %v", err)
		errResponseHandle(Unauthorized, "Incorrect email or password", w, r)
		return
	}

	if err := auth.CheckPasswordHash(
		req.Password,
		user.HashedPassword,
	); err != nil {
		log.Printf("Unable to get user from the database by Id %v", err)
		errResponseHandle(Unauthorized, "Incorrect email or password", w, r)
		return
	}

	response, err := json.Marshal(User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})
	if err != nil {
		log.Printf("Failed to decode request body")
		errResponseHandle(ServerError, "Something went wrong", w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
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
