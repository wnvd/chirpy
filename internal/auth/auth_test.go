package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {

	password := "this_is_a_secret"
	hashedPassword, err := HashPassword(password)
	if err != nil {
		t.Errorf("An error occured while hashing password %v", err)
		return
	}
	if len(hashedPassword) == 0 {
		t.Errorf("Hashed Passwod length is 0")
		return
	}

	if err := CheckPasswordHash(password, hashedPassword); err != nil {
		t.Errorf("An error occured while comparing password and hashed password")
		return
	}

}

func TestJWTToken(t *testing.T) {
	userId := uuid.New()
	tokenSecret := "this_is_a_secret"
	expresIn := time.Hour * 2

	token, err := MakeJWT(
		userId,
		tokenSecret,
		expresIn,
	)
	if err != nil {
		t.Errorf("unable to create a jwt token")
		return
	}

	validatedId, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Errorf("unable to validate at token")
		return
	}
	if userId != validatedId {
		t.Errorf("generate Token != validated token")
		return
	}
}
