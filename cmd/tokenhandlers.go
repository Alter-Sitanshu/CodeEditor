package main

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenResponse struct {
	Token string `json:"token"`
}

type SignInPayload struct {
	Email    string `json:"email"`
	Password string `json:"pass"`
}

func (app *Application) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (app *Application) TokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload SignInPayload
	err := readJSON(w, r, &payload)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, "Could not parse payload")
	}
	// TODO: Change after DB
	userID := 1
	claims := jwt.MapClaims{
		"sub": userID,
		"iss": "GOSocial",
		"aud": "GOSocial",
		"exp": time.Now().Add(app.config.tokencfg.expiry).Unix(),
		"nbf": time.Now().Unix(),
		"iat": time.Now().Unix(),
	}
	token, err := app.authenticator.GenerateToken(claims)
	resp := TokenResponse{
		Token: token,
	}
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError,
			"Server Error: Could not generate token")
		return
	}

	jsonResponse(w, http.StatusOK, resp)
}
