package main

import (
	"net/http"
	"time"

	"log"

	"github.com/wnvd/chirpy/internal/auth"
)

// path: POST /api/refresh
func (cfg *apiConfig) refreshTokenHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	oldToken, err := auth.GetAuthToken(r.Header)
	if err != nil {
		log.Printf("Unable to get bearer token: %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Bad Request")
		return
	}
	if len(oldToken) == 0 {
		log.Printf("refresh token is empty %v", err)
		ErrorResponse(w, http.StatusUnauthorized, "Bad Request")
		return
	}

	user, err := cfg.database.GetUserFromRefreshToken(r.Context(), oldToken)
	if err != nil {
		log.Printf("unable to get user from token: %v", err)
		ErrorResponse(w, http.StatusUnauthorized, "Bad Request")
		return
	}

	newToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Duration(time.Hour))
	if err != nil {
		log.Printf("Unable to create token %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	type refreshToken struct {
		Token string `json:"token"`
	}

	JSONResponse(w, http.StatusOK, refreshToken{
		Token: newToken,
	})
}

// path: Post /api/revoke
func (cfg *apiConfig) revokeTokenHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	refreshToken, err := auth.GetAuthToken(r.Header)
	if err != nil {
		log.Printf("Unable to get bearer token: %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if err := cfg.database.RevokeToken(
		r.Context(),
		refreshToken,
	); err != nil {
		log.Printf("Unable to revoke refresh token %v", err)
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	JSONResponse(w, http.StatusNoContent, nil)

}
