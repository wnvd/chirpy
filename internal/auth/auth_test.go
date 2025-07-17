package auth

import (
	"testing"
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
