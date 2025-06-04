package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/Alter-Sitanshu/CodeEditor/internal/store"
	"github.com/go-chi/chi/v5"
)

type RoomPayload struct {
	Name     string `json:"name"`
	Language string `json:"language"`
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
	ctx := r.Context()
	roomIdParam := chi.URLParam(r, "id")
	roomId, err := strconv.ParseInt(roomIdParam, 10, 64)

	if err != nil {
		log.Println(err.Error())
		jsonResponse(w, http.StatusBadRequest, err.Error())
	}

	room, err := app.database.RoomStore.GetRoomById(ctx, roomId)
	if err != nil {
		log.Println(err.Error())
		jsonResponse(w, http.StatusInternalServerError, err.Error())
	}

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
