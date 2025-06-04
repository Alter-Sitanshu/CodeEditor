package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"time"

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

type TokenPayload struct {
	Token string `json:"token"`
}

func getUserFromctx(r *http.Request) *store.User {
	user := r.Context().Value(userctx).(*store.User)
	return user
}

func (app *Application) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromctx(r)
	jsonResponse(w, http.StatusOK, user)
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
		log.Println(err.Error())
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
			log.Println(store.ErrDupliMail.Error())
			jsonResponse(w, http.StatusBadRequest, "email taken. try another mailid")
		default:
			log.Println(err.Error())
			jsonResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	jsonResponse(w, http.StatusCreated, "user created")
}

func (app *Application) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload TokenPayload
	readJSON(w, r, &payload)

	ctx := r.Context()
	err := app.database.UserStore.ActivateUser(ctx, payload.Token, time.Now())

	if err != nil {
		switch err {
		case store.ErrTokenExpired:
			log.Println(store.ErrTokenExpired.Error())
			jsonResponse(w, http.StatusBadRequest, "token expired/invalid token")
		default:
			log.Println(err.Error())
			jsonResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	jsonResponse(w, http.StatusOK, "user activated")
}
