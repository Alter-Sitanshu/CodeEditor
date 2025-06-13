package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Alter-Sitanshu/CodeEditor/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RoomPayload struct {
	Name     string `json:"name"`
	Language string `json:"language"`
}

func getRoomFromctx(r *http.Request) *store.Room {
	room := r.Context().Value(roomctx).(*store.Room)
	return room
}

func (app *Application) GetUserRoomsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromctx(r)
	ctx := r.Context()

	rooms, err := app.database.RoomStore.GetUserRooms(ctx, user)
	if err != nil {
		log.Println(err.Error())
		jsonResponse(w, http.StatusInternalServerError, err.Error())
	}

	jsonResponse(w, http.StatusOK, rooms)
}

func (app *Application) GetRoomHandler(w http.ResponseWriter, r *http.Request) {
	room := getRoomFromctx(r)

	jsonResponse(w, http.StatusOK, room)
}

func (app *Application) CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromctx(r)
	var payload RoomPayload
	readJSON(w, r, &payload)

	room := store.Room{
		Name:     payload.Name,
		Author:   user,
		Language: payload.Language,
	}
	ctx := r.Context()
	err := app.database.RoomStore.Create(ctx, &room)
	if err != nil {
		log.Println(err.Error())
		jsonResponse(w, http.StatusInternalServerError, "error creating room")
	}

	jsonResponse(w, http.StatusCreated, "room created")
}

func (app *Application) RequestRoomHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromctx(r)
	room := getRoomFromctx(r)
	roleidparam := chi.URLParam(r, "roleid")
	roleid, err := strconv.ParseInt(roleidparam, 10, 64)
	if err != nil {
		log.Println(err.Error())
		jsonResponse(w, http.StatusBadRequest, "role id is not valid")
		return
	}
	ctx := r.Context()
	plainToken := uuid.New().String()
	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])
	app.database.RoomStore.CreateNewJoinToken(
		ctx, expiry, room.Id, user.Id, roleid, hashToken,
	)
	// TODO: Send mail to the author.
	// Assuming the mail is accepted
	jsonResponse(w, http.StatusOK, plainToken) // sending the token via json for now

}

func (app *Application) AcceptMemberHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := chi.URLParam(r, "token")
	err := app.database.RoomStore.AcceptJoinRequest(ctx, token, time.Now())

	if err != nil {
		switch err {
		case store.ErrInvalidRole:
			jsonResponse(w, http.StatusBadRequest, "invalid role")
		default:
			log.Println(err.Error())
			jsonResponse(w, http.StatusInternalServerError, "error adding member")
		}
		return
	}

	jsonResponse(w, http.StatusCreated, "member added")
}
