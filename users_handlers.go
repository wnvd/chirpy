package main

import (
	"database/sql"
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
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	if !strings.Contains(req.Email, "@") {
		log.Printf("Invalid Email")
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	if len(req.Password) < 4 {
		log.Printf("Invalid Password")
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Unable to hash password, %v", err)
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
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
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	// response, err := json.Marshal(responseBody)
	// if err != nil {
	// 	log.Printf("Failed to decode request body")
	// 	// errResponseHandle(ServerError, "Something went wrong", w, r)
	// 	ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
	// 	return
	// }

	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusCreated)
	// w.Write([]byte(response))
	JSONResponse(w, http.StatusCreated, User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})
}

func (cfg *apiConfig) userLoginHandler(
	w http.ResponseWriter,
	r *http.Request) {

	type loginResponse struct {
		User
		Token        string `json:"token,omitempty"`
		RefreshToken string `json:"refresh_token,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	req := &UserLogin{}
	if err := decoder.Decode(req); err != nil {
		log.Printf("Failed to decode request body")
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	if !strings.Contains(req.Email, "@") {
		log.Printf("Invalid Email")
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusBadRequest, "Incorrect email or password")
		return
	}

	if len(req.Password) < 4 {
		log.Printf("Invalid Password")
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusBadRequest, "Incorrect email or password")
		return
	}

	user, err := cfg.database.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	if err := auth.CheckPasswordHash(
		req.Password,
		user.HashedPassword,
	); err != nil {
		// errResponseHandle(ServerError, "Something went wrong", w, r)
		ErrorResponse(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	// setting up token expire time
	tokenExpiresIn := time.Duration(time.Hour)
	// create a JWT/Access token
	jwtToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, tokenExpiresIn)
	if err != nil {
		log.Panicf("Unable to create token %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Unable to create token")
		return
	}

	// setting up refresh token
	genRefreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Panicf("Unable to create refresh token %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// adding refresh token in database
	createdRefreshToken, err := cfg.database.CreateRefreshToken(r.Context(),
		database.CreateRefreshTokenParams{
			Token:     genRefreshToken,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
			RevokedAt: sql.NullTime{Valid: false, Time: time.Time{}},
		})
	if err != nil {
		log.Panicf("Unable to set refresh token %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	// response, err := json.Marshal(User{
	// 	ID:        user.ID,
	// 	CreatedAt: user.CreatedAt,
	// 	UpdatedAt: user.UpdatedAt,
	// 	Email:     user.Email,
	// })
	// if err != nil {
	// 	log.Printf("Failed to decode request body")
	// 	errResponseHandle(ServerError, "Something went wrong", w, r)
	// 	return
	// }
	// w.WriteHeader(http.StatusOK)
	// w.Write([]byte(response))

	JSONResponse(w, http.StatusOK, loginResponse{
		User: User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		},
		Token:        jwtToken,
		RefreshToken: createdRefreshToken.Token,
	})
}

// path: PUT /api/users
func (cfg *apiConfig) updateUserHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Unable to get bearer token: %v", err)
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized Request")
		return
	}

	// validating the jwt token the getting the user UUID
	userUUID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		log.Printf("Unable to validate token: %v", err)
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized Request")
		return
	}

	decoder := json.NewDecoder(r.Body)
	req := &UserLogin{}
	if err := decoder.Decode(req); err != nil {
		log.Printf("Failed to decode request body")
		ErrorResponse(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	if !strings.Contains(req.Email, "@") {
		log.Printf("Invalid Email")
		ErrorResponse(w, http.StatusBadRequest, "Incorrect email or password")
		return
	}

	if len(req.Password) < 4 {
		log.Printf("Invalid Password")
		ErrorResponse(w, http.StatusBadRequest, "Incorrect email or password")
		return
	}

	hasedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Failed to hash password %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	updatedUser, err := cfg.database.UpdateUserById(
		r.Context(),
		database.UpdateUserByIdParams{
			ID:             userUUID,
			Email:          req.Email,
			HashedPassword: hasedPassword,
		},
	)
	if err != nil {
		ErrorResponse(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	JSONResponse(w, http.StatusOK, User {
		ID: updatedUser.ID,
		Email: updatedUser.Email,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
	})
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
