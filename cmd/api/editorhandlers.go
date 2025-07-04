package main

import (
	"log"
	"net/http"

	"github.com/Alter-Sitanshu/CodeEditor/internal/sockets"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (app *Application) EditorRoomHandler(w http.ResponseWriter, r *http.Request) {
	room := getRoomFromctx(r)
	user := getUserFromctx(r)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err.Error())
		jsonResponse(w, http.StatusInternalServerError, "error establishing connection")
		return
	}

	clientConnection := &sockets.Connection{
		RoomID: room.Id,
		UserID: user.Id,
		Conn:   conn,
		Send:   make(chan []byte),
	}

	app.hub.Register <- clientConnection
	log.Println("Client connected: ", conn.LocalAddr())

	go app.hub.ReadMessagesWithVoice(clientConnection, app.vcm)
	go app.hub.WriteMessages(clientConnection)
}

func (app *Application) ExecuteCodeHandler(w http.ResponseWriter, r *http.Request) {
	var req sockets.ExecuteRequest
	readJSON(w, r, &req)
	if req.Code == "" {
		log.Printf("blank code submitted")
		jsonResponse(w, http.StatusBadRequest, "blank code not allowed")
	}
	response, err := app.executor.ExecuteCode(req)
	if err != nil {
		log.Printf("error while executing code")
		jsonResponse(w, http.StatusInternalServerError, *response)
	}

	jsonResponse(w, http.StatusOK, *response)
}
