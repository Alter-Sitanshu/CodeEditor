package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/Alter-Sitanshu/CodeEditor/internal/store"
	"github.com/google/uuid"
)

type UserPayload struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"pass"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
}

func (app *Application) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload UserPayload
	readJSON(w, r, &payload)
	user := store.User{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Email:     payload.Email,
		Age:       payload.Age,
	}

	if err := user.Password.Encrypt(payload.Password); err != nil {
		jsonResponse(w, http.StatusBadRequest, "error encoding password")
		return
	}

	ctx := r.Context()
	plainToken := uuid.New().String()
	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])
	err := app.database.UserStore.CreateAndInvite(ctx, &user, hashToken, app.config.tokencfg.expiry)
	if err != nil {
		switch err {
		case store.ErrDupliMail:
			jsonResponse(w, http.StatusBadRequest, "email taken. try another mailid")
		default:
			jsonResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	jsonResponse(w, http.StatusCreated, "user created")
}
